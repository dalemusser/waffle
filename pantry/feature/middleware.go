// feature/middleware.go
package feature

import (
	"encoding/json"
	"net/http"
	"strings"
)

// MiddlewareConfig configures the feature flag middleware.
type MiddlewareConfig struct {
	// Manager is the feature flag manager to use.
	Manager *Manager

	// ContextBuilder extracts evaluation context from the request.
	// If nil, a default builder using headers is used.
	ContextBuilder func(r *http.Request) *EvalContext

	// UserIDHeader is the header to extract user ID from.
	// Default: "X-User-ID"
	UserIDHeader string

	// GroupsHeader is the header to extract groups from (comma-separated).
	// Default: "X-User-Groups"
	GroupsHeader string
}

// DefaultMiddlewareConfig returns sensible defaults.
func DefaultMiddlewareConfig(manager *Manager) MiddlewareConfig {
	return MiddlewareConfig{
		Manager:      manager,
		UserIDHeader: "X-User-ID",
		GroupsHeader: "X-User-Groups",
	}
}

// Middleware creates HTTP middleware that extracts feature flag context.
func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.UserIDHeader == "" {
		cfg.UserIDHeader = "X-User-ID"
	}
	if cfg.GroupsHeader == "" {
		cfg.GroupsHeader = "X-User-Groups"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var evalCtx *EvalContext

			if cfg.ContextBuilder != nil {
				evalCtx = cfg.ContextBuilder(r)
			} else {
				evalCtx = buildContextFromRequest(r, cfg)
			}

			ctx := WithContext(r.Context(), evalCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// buildContextFromRequest extracts evaluation context from request headers.
func buildContextFromRequest(r *http.Request, cfg MiddlewareConfig) *EvalContext {
	ctx := NewEvalContext()

	// Extract user ID
	if userID := r.Header.Get(cfg.UserIDHeader); userID != "" {
		ctx.UserID = userID
	}

	// Extract groups
	if groups := r.Header.Get(cfg.GroupsHeader); groups != "" {
		ctx.Groups = strings.Split(groups, ",")
		for i := range ctx.Groups {
			ctx.Groups[i] = strings.TrimSpace(ctx.Groups[i])
		}
	}

	return ctx
}

// RequireFeature returns middleware that only allows access if a flag is enabled.
func RequireFeature(manager *Manager, flag string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := FromContext(r.Context())
			if !manager.IsEnabledFor(flag, ctx) {
				http.Error(w, "Feature not available", http.StatusNotFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireFeatureFunc returns middleware with a custom handler for disabled flags.
func RequireFeatureFunc(manager *Manager, flag string, onDisabled http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := FromContext(r.Context())
			if !manager.IsEnabledFor(flag, ctx) {
				if onDisabled != nil {
					onDisabled(w, r)
				} else {
					http.Error(w, "Feature not available", http.StatusNotFound)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Handler returns an http.Handler that checks a feature flag.
// If enabled, calls the enabled handler; otherwise calls the disabled handler.
func Handler(manager *Manager, flag string, enabled, disabled http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := FromContext(r.Context())
		if manager.IsEnabledFor(flag, ctx) {
			enabled.ServeHTTP(w, r)
		} else if disabled != nil {
			disabled.ServeHTTP(w, r)
		} else {
			http.Error(w, "Feature not available", http.StatusNotFound)
		}
	})
}

// HandlerFunc is like Handler but for http.HandlerFunc.
func HandlerFunc(manager *Manager, flag string, enabled, disabled http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := FromContext(r.Context())
		if manager.IsEnabledFor(flag, ctx) {
			enabled(w, r)
		} else if disabled != nil {
			disabled(w, r)
		} else {
			http.Error(w, "Feature not available", http.StatusNotFound)
		}
	}
}

// AdminHandler returns an HTTP handler for managing feature flags.
// This provides a simple REST API for flag management.
type AdminHandler struct {
	manager *Manager
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(manager *Manager) *AdminHandler {
	return &AdminHandler{manager: manager}
}

// ServeHTTP handles admin requests.
func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodPut:
		h.handlePut(w, r)
	case http.MethodDelete:
		h.handleDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")

	w.Header().Set("Content-Type", "application/json")

	if key == "" {
		// List all flags
		flags := h.manager.All()
		json.NewEncoder(w).Encode(flags)
		return
	}

	// Get specific flag
	flag, exists := h.manager.Get(key)
	if !exists {
		http.Error(w, "Flag not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(flag)
}

func (h *AdminHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var flag Flag
	if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.manager.Register(&flag); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(flag)
}

func (h *AdminHandler) handlePut(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")
	if key == "" {
		http.Error(w, "Flag key required", http.StatusBadRequest)
		return
	}

	var update struct {
		Enabled    *bool `json:"enabled,omitempty"`
		Percentage *int  `json:"percentage,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if update.Enabled != nil {
		if err := h.manager.SetEnabled(key, *update.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	if update.Percentage != nil {
		if err := h.manager.SetPercentage(key, *update.Percentage); err != nil {
			if err == ErrInvalidPercentage {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, err.Error(), http.StatusNotFound)
			}
			return
		}
	}

	flag, _ := h.manager.Get(key)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flag)
}

func (h *AdminHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")
	if key == "" {
		http.Error(w, "Flag key required", http.StatusBadRequest)
		return
	}

	if err := h.manager.Delete(key); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EvaluateHandler returns a handler for evaluating flags via HTTP.
// Useful for client-side feature flag checking.
type EvaluateHandler struct {
	manager *Manager
}

// NewEvaluateHandler creates a new evaluate handler.
func NewEvaluateHandler(manager *Manager) *EvaluateHandler {
	return &EvaluateHandler{manager: manager}
}

// ServeHTTP handles evaluation requests.
// POST /evaluate with {"flags": ["flag1", "flag2"], "context": {...}}
func (h *EvaluateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Flags   []string       `json:"flags"`
		Context *EvalContext   `json:"context,omitempty"`
		All     bool           `json:"all,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := req.Context
	if ctx == nil {
		ctx = FromContext(r.Context())
	}

	results := make(map[string]bool)

	if req.All {
		// Return all flags
		for _, flag := range h.manager.All() {
			results[flag.Key] = h.manager.IsEnabledFor(flag.Key, ctx)
		}
	} else {
		// Return requested flags
		for _, key := range req.Flags {
			results[key] = h.manager.IsEnabledFor(key, ctx)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ClientConfig generates a configuration object for client-side use.
func ClientConfig(manager *Manager, ctx *EvalContext, flags ...string) map[string]bool {
	result := make(map[string]bool)

	if len(flags) == 0 {
		// Return all flags
		for _, flag := range manager.All() {
			result[flag.Key] = manager.IsEnabledFor(flag.Key, ctx)
		}
	} else {
		// Return requested flags
		for _, key := range flags {
			result[key] = manager.IsEnabledFor(key, ctx)
		}
	}

	return result
}

// ClientConfigJSON returns client config as JSON.
func ClientConfigJSON(manager *Manager, ctx *EvalContext, flags ...string) ([]byte, error) {
	return json.Marshal(ClientConfig(manager, ctx, flags...))
}
