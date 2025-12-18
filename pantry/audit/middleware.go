// audit/middleware.go
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// sensitiveQueryParams is the default list of query parameter names whose values
// should be redacted in audit logs. These commonly contain secrets or tokens.
var sensitiveQueryParams = map[string]bool{
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"api_key":       true,
	"apikey":        true,
	"key":           true,
	"secret":        true,
	"password":      true,
	"passwd":        true,
	"auth":          true,
	"authorization": true,
	"bearer":        true,
	"session":       true,
	"session_id":    true,
	"sessionid":     true,
	"jwt":           true,
	"credential":    true,
	"credentials":   true,
	"client_secret": true,
	"code":          true, // OAuth authorization code
	"state":         true, // OAuth state (can be sensitive)
}

// sanitizeQuery redacts sensitive query parameter values from a raw query string.
// Returns the sanitized query string with sensitive values replaced by "[REDACTED]".
func sanitizeQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// If we can't parse it, redact the entire query to be safe
		return "[REDACTED]"
	}

	for key := range values {
		// Check lowercase version of key against sensitive params
		if sensitiveQueryParams[strings.ToLower(key)] {
			values.Set(key, "[REDACTED]")
		}
	}

	return values.Encode()
}

// MiddlewareConfig configures the audit middleware.
type MiddlewareConfig struct {
	// Logger is the audit logger to use.
	Logger Logger

	// ActorExtractor extracts actor information from requests.
	// If nil, uses default header-based extraction.
	ActorExtractor func(r *http.Request) *Actor

	// ActionMapper maps requests to action names.
	// If nil, uses default method + path mapping.
	ActionMapper func(r *http.Request) string

	// ResourceExtractor extracts resource information from requests.
	ResourceExtractor func(r *http.Request) *Resource

	// ShouldAudit determines if a request should be audited.
	// If nil, all requests are audited.
	ShouldAudit func(r *http.Request) bool

	// SkipPaths are paths that should not be audited.
	SkipPaths []string

	// SkipMethods are HTTP methods that should not be audited.
	// Default: GET, HEAD, OPTIONS for some configurations.
	SkipMethods []string

	// CaptureRequestBody captures the request body.
	CaptureRequestBody bool

	// CaptureResponseBody captures the response body.
	CaptureResponseBody bool

	// MaxBodySize is the maximum body size to capture.
	// Default: 10KB
	MaxBodySize int

	// UserIDHeader is the header containing the user ID.
	// Default: "X-User-ID"
	UserIDHeader string

	// RequestIDHeader is the header containing the request ID.
	// Default: "X-Request-ID"
	RequestIDHeader string

	// Tags are added to all audit events.
	Tags []string
}

// DefaultMiddlewareConfig returns sensible defaults.
func DefaultMiddlewareConfig(logger Logger) MiddlewareConfig {
	return MiddlewareConfig{
		Logger:          logger,
		MaxBodySize:     10 * 1024, // 10KB
		UserIDHeader:    "X-User-ID",
		RequestIDHeader: "X-Request-ID",
	}
}

// Middleware creates HTTP middleware for audit logging.
func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = 10 * 1024
	}
	if cfg.UserIDHeader == "" {
		cfg.UserIDHeader = "X-User-ID"
	}
	if cfg.RequestIDHeader == "" {
		cfg.RequestIDHeader = "X-Request-ID"
	}

	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[strings.ToUpper(m)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if should skip
			if skipPaths[r.URL.Path] || skipMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.ShouldAudit != nil && !cfg.ShouldAudit(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Start timing
			start := time.Now()

			// Capture request body if needed
			var requestBody []byte
			if cfg.CaptureRequestBody && r.Body != nil {
				body, _ := io.ReadAll(io.LimitReader(r.Body, int64(cfg.MaxBodySize)))
				requestBody = body
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// Wrap response writer to capture status and body
			wrapped := &responseWriter{
				ResponseWriter: w,
				captureBody:    cfg.CaptureResponseBody,
				maxBodySize:    cfg.MaxBodySize,
			}

			// Add logger to context
			ctx := WithLogger(r.Context(), cfg.Logger)
			r = r.WithContext(ctx)

			// Serve request
			next.ServeHTTP(wrapped, r)

			// Build audit event
			event := buildEventFromRequest(r, wrapped, cfg, start, requestBody)

			// Log asynchronously
			if cfg.Logger != nil {
				cfg.Logger.LogAsync(r.Context(), event)
			}
		})
	}
}

// buildEventFromRequest creates an audit event from a request/response.
func buildEventFromRequest(r *http.Request, rw *responseWriter, cfg MiddlewareConfig, start time.Time, requestBody []byte) *Event {
	// Determine action
	action := r.Method + " " + r.URL.Path
	if cfg.ActionMapper != nil {
		action = cfg.ActionMapper(r)
	}

	// Extract actor
	var actor *Actor
	if cfg.ActorExtractor != nil {
		actor = cfg.ActorExtractor(r)
	} else {
		actor = extractActorFromRequest(r, cfg)
	}

	// Extract resource
	var resource *Resource
	if cfg.ResourceExtractor != nil {
		resource = cfg.ResourceExtractor(r)
	}

	// Determine outcome
	outcome := OutcomeSuccess
	if rw.status >= 400 {
		outcome = OutcomeFailure
	}

	// Build event
	event := &Event{
		Timestamp: start,
		Action:    action,
		Actor:     actor,
		Resource:  resource,
		Outcome:   outcome,
		Context: &EventContext{
			RequestID: r.Header.Get(cfg.RequestIDHeader),
		},
		Tags: cfg.Tags,
		Metadata: map[string]any{
			"http.method":      r.Method,
			"http.path":        r.URL.Path,
			"http.query":       sanitizeQuery(r.URL.RawQuery),
			"http.status":      rw.status,
			"http.duration_ms": time.Since(start).Milliseconds(),
		},
	}

	// Add request body if captured
	if len(requestBody) > 0 {
		// Try to parse as JSON
		var jsonBody any
		if err := json.Unmarshal(requestBody, &jsonBody); err == nil {
			event.Metadata["http.request_body"] = jsonBody
		} else {
			event.Metadata["http.request_body"] = string(requestBody)
		}
	}

	// Add response body if captured
	if len(rw.body) > 0 {
		var jsonBody any
		if err := json.Unmarshal(rw.body, &jsonBody); err == nil {
			event.Metadata["http.response_body"] = jsonBody
		} else {
			event.Metadata["http.response_body"] = string(rw.body)
		}
	}

	// Add failure reason for errors
	if outcome == OutcomeFailure {
		event.Reason = http.StatusText(rw.status)
	}

	return event
}

// extractActorFromRequest extracts actor from request headers.
func extractActorFromRequest(r *http.Request, cfg MiddlewareConfig) *Actor {
	actor := &Actor{
		Type: "user",
		IP:   getClientIP(r),
	}

	if userID := r.Header.Get(cfg.UserIDHeader); userID != "" {
		actor.ID = userID
	}

	if ua := r.UserAgent(); ua != "" {
		actor.UserAgent = ua
	}

	return actor
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// responseWriter wraps http.ResponseWriter to capture status and body.
type responseWriter struct {
	http.ResponseWriter
	status      int
	written     int64
	captureBody bool
	maxBodySize int
	body        []byte
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	if w.captureBody && len(w.body) < w.maxBodySize {
		remaining := w.maxBodySize - len(w.body)
		if len(b) <= remaining {
			w.body = append(w.body, b...)
		} else {
			w.body = append(w.body, b[:remaining]...)
		}
	}

	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// RequestAuditor provides a simple interface for auditing within handlers.
type RequestAuditor struct {
	logger  Logger
	request *http.Request
	actor   *Actor
}

// NewRequestAuditor creates an auditor for a specific request.
func NewRequestAuditor(logger Logger, r *http.Request) *RequestAuditor {
	return &RequestAuditor{
		logger:  logger,
		request: r,
		actor:   extractActorFromRequest(r, MiddlewareConfig{UserIDHeader: "X-User-ID"}),
	}
}

// WithActor sets the actor for this auditor.
func (a *RequestAuditor) WithActor(actor *Actor) *RequestAuditor {
	a.actor = actor
	return a
}

// Log logs an audit event.
func (a *RequestAuditor) Log(action string, resource *Resource, outcome Outcome, metadata map[string]any) error {
	event := &Event{
		Action:   action,
		Actor:    a.actor,
		Resource: resource,
		Outcome:  outcome,
		Context: &EventContext{
			RequestID: a.request.Header.Get("X-Request-ID"),
		},
		Metadata: metadata,
	}

	return a.logger.Log(a.request.Context(), event)
}

// LogSuccess logs a successful action.
func (a *RequestAuditor) LogSuccess(action string, resource *Resource) error {
	return a.Log(action, resource, OutcomeSuccess, nil)
}

// LogFailure logs a failed action.
func (a *RequestAuditor) LogFailure(action string, resource *Resource, reason string) error {
	event := &Event{
		Action:   action,
		Actor:    a.actor,
		Resource: resource,
		Outcome:  OutcomeFailure,
		Reason:   reason,
		Context: &EventContext{
			RequestID: a.request.Header.Get("X-Request-ID"),
		},
	}
	return a.logger.Log(a.request.Context(), event)
}

// LogCreate logs a resource creation.
func (a *RequestAuditor) LogCreate(resourceType, resourceID string) error {
	return a.LogSuccess(resourceType+".create", &Resource{Type: resourceType, ID: resourceID})
}

// LogRead logs a resource read.
func (a *RequestAuditor) LogRead(resourceType, resourceID string) error {
	return a.LogSuccess(resourceType+".read", &Resource{Type: resourceType, ID: resourceID})
}

// LogUpdate logs a resource update.
func (a *RequestAuditor) LogUpdate(resourceType, resourceID string, changes []Change) error {
	event := &Event{
		Action:   resourceType + ".update",
		Actor:    a.actor,
		Resource: &Resource{Type: resourceType, ID: resourceID},
		Outcome:  OutcomeSuccess,
		Changes:  changes,
		Context: &EventContext{
			RequestID: a.request.Header.Get("X-Request-ID"),
		},
	}
	return a.logger.Log(a.request.Context(), event)
}

// LogDelete logs a resource deletion.
func (a *RequestAuditor) LogDelete(resourceType, resourceID string) error {
	return a.LogSuccess(resourceType+".delete", &Resource{Type: resourceType, ID: resourceID})
}

// AuditHandler wraps a handler with audit logging for specific actions.
func AuditHandler(logger Logger, action string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &responseWriter{ResponseWriter: w}
		start := time.Now()

		handler.ServeHTTP(wrapped, r)

		outcome := OutcomeSuccess
		if wrapped.status >= 400 {
			outcome = OutcomeFailure
		}

		event := NewEvent(action).
			WithActor(extractActorFromRequest(r, MiddlewareConfig{UserIDHeader: "X-User-ID"})).
			WithRequestID(r.Header.Get("X-Request-ID")).
			WithMetadata("http.status", wrapped.status).
			WithMetadata("http.duration_ms", time.Since(start).Milliseconds())

		if outcome == OutcomeSuccess {
			event.Success()
		} else {
			event.Failure(http.StatusText(wrapped.status))
		}

		event.LogAsync(r.Context(), logger)
	})
}

// AdminHandler provides an HTTP API for querying audit logs.
type AdminHandler struct {
	logger Logger
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(logger Logger) *AdminHandler {
	return &AdminHandler{logger: logger}
}

// ServeHTTP handles admin requests.
func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleQuery(w, r)
	case http.MethodPost:
		h.handleSearch(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	query := &Query{
		Limit: 100,
	}

	// Parse query parameters
	q := r.URL.Query()

	if v := q.Get("actor_id"); v != "" {
		query.ActorID = v
	}
	if v := q.Get("actor_type"); v != "" {
		query.ActorType = v
	}
	if v := q.Get("resource_id"); v != "" {
		query.ResourceID = v
	}
	if v := q.Get("resource_type"); v != "" {
		query.ResourceType = v
	}
	if v := q.Get("action"); v != "" {
		query.Action = v
	}
	if v := q.Get("outcome"); v != "" {
		query.Outcome = Outcome(v)
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.From = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.To = t
		}
	}

	result, err := h.logger.Query(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AdminHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	var query Query
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if query.Limit <= 0 {
		query.Limit = 100
	}

	result, err := h.logger.Query(r.Context(), &query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Common audit actions.
const (
	ActionUserLogin         = "user.login"
	ActionUserLogout        = "user.logout"
	ActionUserCreate        = "user.create"
	ActionUserUpdate        = "user.update"
	ActionUserDelete        = "user.delete"
	ActionUserPasswordReset = "user.password_reset"

	ActionPermissionGrant  = "permission.grant"
	ActionPermissionRevoke = "permission.revoke"

	ActionResourceCreate = "resource.create"
	ActionResourceRead   = "resource.read"
	ActionResourceUpdate = "resource.update"
	ActionResourceDelete = "resource.delete"

	ActionSettingChange = "setting.change"

	ActionAPIKeyCreate = "api_key.create"
	ActionAPIKeyRevoke = "api_key.revoke"

	ActionExport = "data.export"
	ActionImport = "data.import"
)

// LogUserLogin logs a user login event.
func LogUserLogin(ctx context.Context, logger Logger, userID, userName, ip, userAgent string, success bool) {
	event := NewEvent(ActionUserLogin).
		WithActorUser(userID, userName, "").
		WithIP(ip).
		WithUserAgent(userAgent)

	if success {
		event.Success()
	} else {
		event.Failure("invalid credentials")
	}

	event.LogAsync(ctx, logger)
}

// LogUserLogout logs a user logout event.
func LogUserLogout(ctx context.Context, logger Logger, userID, userName string) {
	NewEvent(ActionUserLogout).
		WithActorUser(userID, userName, "").
		Success().
		LogAsync(ctx, logger)
}

// LogResourceChange logs a resource change with before/after values.
func LogResourceChange(ctx context.Context, logger Logger, actor *Actor, resourceType, resourceID string, changes []Change) {
	NewEvent(resourceType + ".update").
		WithActor(actor).
		WithResourceID(resourceType, resourceID).
		WithChanges(changes).
		Success().
		LogAsync(ctx, logger)
}

// DiffChanges compares two values and returns the changes.
func DiffChanges(old, new map[string]any) []Change {
	var changes []Change

	// Find changed and new fields
	for k, newVal := range new {
		oldVal, exists := old[k]
		if !exists {
			changes = append(changes, Change{Field: k, OldValue: nil, NewValue: newVal})
		} else if !equal(oldVal, newVal) {
			changes = append(changes, Change{Field: k, OldValue: oldVal, NewValue: newVal})
		}
	}

	// Find deleted fields
	for k, oldVal := range old {
		if _, exists := new[k]; !exists {
			changes = append(changes, Change{Field: k, OldValue: oldVal, NewValue: nil})
		}
	}

	return changes
}

func equal(a, b any) bool {
	// Simple equality check
	return a == b
}
