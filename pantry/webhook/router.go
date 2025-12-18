package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Handler is a function that handles a webhook event.
type Handler func(ctx context.Context, event *RawEvent) error

// HandlerFunc is an HTTP handler function for webhooks.
type HandlerFunc func(w http.ResponseWriter, r *http.Request, event *RawEvent)

// Router routes incoming webhook events to handlers based on event type.
type Router struct {
	mu            sync.RWMutex
	handlers      map[string][]Handler
	httpHandlers  map[string][]HandlerFunc
	verifier      Verifier
	maxBodySize   int64
	errorHandler  func(w http.ResponseWriter, r *http.Request, err error)
	eventParser   func([]byte) (*RawEvent, error)
	beforeHandler func(ctx context.Context, event *RawEvent) error
	afterHandler  func(ctx context.Context, event *RawEvent, err error)
}

// RouterConfig configures the webhook router.
type RouterConfig struct {
	// Verifier is used to verify incoming webhook signatures.
	// If nil, signatures are not verified.
	Verifier Verifier

	// MaxBodySize is the maximum allowed request body size.
	// Default: 10MB
	MaxBodySize int64

	// ErrorHandler is called when an error occurs.
	// If nil, a default error response is sent.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// EventParser parses the webhook payload into an event.
	// If nil, the default JSON parser is used.
	EventParser func([]byte) (*RawEvent, error)

	// BeforeHandler is called before any event handler.
	// Can be used for logging, metrics, or additional validation.
	BeforeHandler func(ctx context.Context, event *RawEvent) error

	// AfterHandler is called after event handlers complete.
	// Can be used for logging or cleanup.
	AfterHandler func(ctx context.Context, event *RawEvent, err error)
}

// NewRouter creates a new webhook router.
func NewRouter(cfg RouterConfig) *Router {
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = MaxBodySize
	}

	if cfg.EventParser == nil {
		cfg.EventParser = defaultEventParser
	}

	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultErrorHandler
	}

	return &Router{
		handlers:      make(map[string][]Handler),
		httpHandlers:  make(map[string][]HandlerFunc),
		verifier:      cfg.Verifier,
		maxBodySize:   cfg.MaxBodySize,
		errorHandler:  cfg.ErrorHandler,
		eventParser:   cfg.EventParser,
		beforeHandler: cfg.BeforeHandler,
		afterHandler:  cfg.AfterHandler,
	}
}

// defaultEventParser parses a JSON webhook payload.
func defaultEventParser(data []byte) (*RawEvent, error) {
	// Try to parse as a Payload first
	var payload Payload
	if err := json.Unmarshal(data, &payload); err == nil && payload.Event.Type != "" {
		return &payload.Event, nil
	}

	// Fall back to parsing as a direct event
	var event RawEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	// If no type, try to extract from common fields
	if event.Type == "" {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err == nil {
			// Try common type field names
			for _, field := range []string{"type", "event_type", "event", "action"} {
				if typeData, ok := raw[field]; ok {
					var eventType string
					if json.Unmarshal(typeData, &eventType) == nil {
						event.Type = eventType
						break
					}
				}
			}
		}
	}

	// Store the entire payload as data if not already set
	if event.Data == nil {
		event.Data = data
	}

	return &event, nil
}

// defaultErrorHandler sends a JSON error response.
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusInternalServerError

	switch err {
	case ErrInvalidSignature, ErrMissingSignature:
		statusCode = http.StatusUnauthorized
	case ErrTimestampExpired, ErrMissingTimestamp:
		statusCode = http.StatusBadRequest
	case ErrInvalidPayload:
		statusCode = http.StatusBadRequest
	case ErrNoHandlerFound:
		statusCode = http.StatusNotFound
	case ErrRequestBodyTooLarge:
		statusCode = http.StatusRequestEntityTooLarge
	}

	WriteError(w, statusCode, err)
}

// On registers a handler for an event type pattern.
// Patterns can include wildcards:
//   - "order.created" - exact match
//   - "order.*" - matches any event starting with "order."
//   - "*" - matches all events
func (r *Router) On(pattern string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[pattern] = append(r.handlers[pattern], handler)
}

// OnHTTP registers an HTTP handler for an event type pattern.
func (r *Router) OnHTTP(pattern string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.httpHandlers[pattern] = append(r.httpHandlers[pattern], handler)
}

// Handle processes an incoming webhook event.
func (r *Router) Handle(ctx context.Context, event *RawEvent) error {
	handlers := r.getHandlers(event.Type)

	if len(handlers) == 0 {
		return ErrNoHandlerFound
	}

	// Call before handler
	if r.beforeHandler != nil {
		if err := r.beforeHandler(ctx, event); err != nil {
			return err
		}
	}

	// Call all matching handlers
	var lastErr error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			lastErr = err
		}
	}

	// Call after handler
	if r.afterHandler != nil {
		r.afterHandler(ctx, event, lastErr)
	}

	return lastErr
}

// getHandlers returns all handlers matching the event type.
func (r *Router) getHandlers(eventType string) []Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var handlers []Handler

	// Collect matching handlers
	for pattern, hs := range r.handlers {
		if matchPattern(pattern, eventType) {
			handlers = append(handlers, hs...)
		}
	}

	return handlers
}

// getHTTPHandlers returns all HTTP handlers matching the event type.
func (r *Router) getHTTPHandlers(eventType string) []HandlerFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var handlers []HandlerFunc

	for pattern, hs := range r.httpHandlers {
		if matchPattern(pattern, eventType) {
			handlers = append(handlers, hs...)
		}
	}

	return handlers
}

// matchPattern checks if an event type matches a pattern.
func matchPattern(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}

	if pattern == eventType {
		return true
	}

	// Wildcard matching: "order.*" matches "order.created"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		if strings.HasPrefix(eventType, prefix+".") || eventType == prefix {
			return true
		}
	}

	// More flexible wildcard: "order*" matches "order.created", "order_created"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(eventType, prefix) {
			return true
		}
	}

	return false
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// Verify signature if verifier is set
	if r.verifier != nil {
		if err := r.verifier.Verify(req); err != nil {
			r.errorHandler(w, req, err)
			return
		}
	}

	// Read body
	body, err := ReadBody(req, r.maxBodySize)
	if err != nil {
		r.errorHandler(w, req, err)
		return
	}

	// Reset body for downstream handlers
	ResetBody(req, body)

	// Parse event
	event, err := r.eventParser(body)
	if err != nil {
		r.errorHandler(w, req, ErrInvalidPayload)
		return
	}

	// Add event and payload to context
	ctx = ContextWithEvent(ctx, event)
	ctx = ContextWithPayload(ctx, body)

	// Get HTTP handlers
	httpHandlers := r.getHTTPHandlers(event.Type)

	// If we have HTTP handlers, use those
	if len(httpHandlers) > 0 {
		// Call before handler
		if r.beforeHandler != nil {
			if err := r.beforeHandler(ctx, event); err != nil {
				r.errorHandler(w, req, err)
				return
			}
		}

		for _, handler := range httpHandlers {
			handler(w, req.WithContext(ctx), event)
		}

		// Call after handler
		if r.afterHandler != nil {
			r.afterHandler(ctx, event, nil)
		}
		return
	}

	// Fall back to regular handlers
	handlers := r.getHandlers(event.Type)
	if len(handlers) == 0 && len(httpHandlers) == 0 {
		r.errorHandler(w, req, ErrNoHandlerFound)
		return
	}

	// Call before handler
	if r.beforeHandler != nil {
		if err := r.beforeHandler(ctx, event); err != nil {
			r.errorHandler(w, req, err)
			return
		}
	}

	// Call handlers
	var lastErr error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			lastErr = err
		}
	}

	// Call after handler
	if r.afterHandler != nil {
		r.afterHandler(ctx, event, lastErr)
	}

	if lastErr != nil {
		r.errorHandler(w, req, lastErr)
		return
	}

	WriteSuccess(w, "Event processed")
}

// HandleFunc returns an http.HandlerFunc for the router.
func (r *Router) HandleFunc() http.HandlerFunc {
	return r.ServeHTTP
}

// Routes returns a list of registered patterns.
func (r *Router) Routes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	patterns := make(map[string]bool)
	for pattern := range r.handlers {
		patterns[pattern] = true
	}
	for pattern := range r.httpHandlers {
		patterns[pattern] = true
	}

	result := make([]string, 0, len(patterns))
	for pattern := range patterns {
		result = append(result, pattern)
	}
	sort.Strings(result)
	return result
}

// Clear removes all registered handlers.
func (r *Router) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = make(map[string][]Handler)
	r.httpHandlers = make(map[string][]HandlerFunc)
}

// Group creates a handler group with a common prefix.
type Group struct {
	router *Router
	prefix string
}

// Group creates a new handler group with a prefix.
func (r *Router) Group(prefix string) *Group {
	return &Group{
		router: r,
		prefix: prefix,
	}
}

// On registers a handler with the group's prefix.
func (g *Group) On(pattern string, handler Handler) {
	fullPattern := g.prefix
	if pattern != "" && pattern != "*" {
		fullPattern = g.prefix + "." + pattern
	} else if pattern == "*" {
		fullPattern = g.prefix + ".*"
	}
	g.router.On(fullPattern, handler)
}

// OnHTTP registers an HTTP handler with the group's prefix.
func (g *Group) OnHTTP(pattern string, handler HandlerFunc) {
	fullPattern := g.prefix
	if pattern != "" && pattern != "*" {
		fullPattern = g.prefix + "." + pattern
	} else if pattern == "*" {
		fullPattern = g.prefix + ".*"
	}
	g.router.OnHTTP(fullPattern, handler)
}

// TypedHandler creates a handler that parses the event data into a specific type.
func TypedHandler[T any](handler func(ctx context.Context, event *RawEvent, data T) error) Handler {
	return func(ctx context.Context, event *RawEvent) error {
		var data T
		if err := event.ParseData(&data); err != nil {
			return err
		}
		return handler(ctx, event, data)
	}
}

// TypedHTTPHandler creates an HTTP handler that parses the event data into a specific type.
func TypedHTTPHandler[T any](handler func(w http.ResponseWriter, r *http.Request, event *RawEvent, data T)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, event *RawEvent) {
		var data T
		if err := event.ParseData(&data); err != nil {
			WriteError(w, http.StatusBadRequest, err)
			return
		}
		handler(w, r, event, data)
	}
}
