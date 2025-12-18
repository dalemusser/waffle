// audit/audit.go
package audit

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Event represents an audit log event.
type Event struct {
	// ID is a unique identifier for the event.
	ID string `json:"id"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Action is what happened (e.g., "user.login", "document.create").
	Action string `json:"action"`

	// Actor is who performed the action.
	Actor *Actor `json:"actor,omitempty"`

	// Resource is what was acted upon.
	Resource *Resource `json:"resource,omitempty"`

	// Context provides additional contextual information.
	Context *EventContext `json:"context,omitempty"`

	// Outcome is the result of the action.
	Outcome Outcome `json:"outcome"`

	// Reason explains the outcome (especially for failures).
	Reason string `json:"reason,omitempty"`

	// Changes tracks what was modified.
	Changes []Change `json:"changes,omitempty"`

	// Metadata holds additional arbitrary data.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Tags for categorization and filtering.
	Tags []string `json:"tags,omitempty"`
}

// Actor represents who performed an action.
type Actor struct {
	// ID is the unique identifier (user ID, service ID, etc.).
	ID string `json:"id"`

	// Type is the kind of actor (e.g., "user", "service", "system").
	Type string `json:"type"`

	// Name is a human-readable name.
	Name string `json:"name,omitempty"`

	// Email is the actor's email (if applicable).
	Email string `json:"email,omitempty"`

	// IP is the actor's IP address.
	IP string `json:"ip,omitempty"`

	// UserAgent is the actor's user agent string.
	UserAgent string `json:"user_agent,omitempty"`

	// SessionID is the actor's session identifier.
	SessionID string `json:"session_id,omitempty"`

	// Metadata holds additional actor data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Resource represents what was acted upon.
type Resource struct {
	// ID is the unique identifier.
	ID string `json:"id"`

	// Type is the kind of resource (e.g., "user", "document", "order").
	Type string `json:"type"`

	// Name is a human-readable name.
	Name string `json:"name,omitempty"`

	// Metadata holds additional resource data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EventContext provides contextual information about the event.
type EventContext struct {
	// RequestID is the unique request identifier.
	RequestID string `json:"request_id,omitempty"`

	// TraceID for distributed tracing.
	TraceID string `json:"trace_id,omitempty"`

	// SpanID for distributed tracing.
	SpanID string `json:"span_id,omitempty"`

	// Service is the service that generated the event.
	Service string `json:"service,omitempty"`

	// Environment (e.g., "production", "staging").
	Environment string `json:"environment,omitempty"`

	// Version is the application version.
	Version string `json:"version,omitempty"`

	// Location is where the action was performed from.
	Location *Location `json:"location,omitempty"`
}

// Location represents geographic location.
type Location struct {
	Country   string  `json:"country,omitempty"`
	Region    string  `json:"region,omitempty"`
	City      string  `json:"city,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// Outcome represents the result of an action.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomePending Outcome = "pending"
	OutcomeUnknown Outcome = "unknown"
)

// Change represents a modification to a resource.
type Change struct {
	// Field is the name of the changed field.
	Field string `json:"field"`

	// OldValue is the value before the change.
	OldValue any `json:"old_value,omitempty"`

	// NewValue is the value after the change.
	NewValue any `json:"new_value,omitempty"`
}

// Logger is the audit logging interface.
type Logger interface {
	// Log records an audit event.
	Log(ctx context.Context, event *Event) error

	// LogAsync records an audit event asynchronously.
	LogAsync(ctx context.Context, event *Event)

	// Query retrieves audit events matching the criteria.
	Query(ctx context.Context, query *Query) (*QueryResult, error)

	// Close closes the logger and flushes any pending events.
	Close() error
}

// AuditLogger is the main audit logging implementation.
type AuditLogger struct {
	mu     sync.RWMutex
	store  Store
	config Config

	// Async processing
	eventCh chan *Event
	done    chan struct{}
	wg      sync.WaitGroup

	// Hooks
	beforeLog []BeforeLogHook
	afterLog  []AfterLogHook

	// ID generator
	idGen IDGenerator
}

// Config configures the audit logger.
type Config struct {
	// Store is the storage backend.
	Store Store

	// Service is the name of this service.
	Service string

	// Environment (e.g., "production", "staging").
	Environment string

	// Version is the application version.
	Version string

	// BufferSize is the size of the async event buffer.
	// Default: 1000
	BufferSize int

	// Workers is the number of async workers.
	// Default: 2
	Workers int

	// IDGenerator generates unique event IDs.
	// Default: UUIDGenerator
	IDGenerator IDGenerator
}

// BeforeLogHook is called before an event is logged.
type BeforeLogHook func(ctx context.Context, event *Event) error

// AfterLogHook is called after an event is logged.
type AfterLogHook func(ctx context.Context, event *Event, err error)

// IDGenerator generates unique event IDs.
type IDGenerator func() string

// NewLogger creates a new audit logger.
func NewLogger(cfg Config) *AuditLogger {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.IDGenerator == nil {
		cfg.IDGenerator = DefaultIDGenerator
	}

	l := &AuditLogger{
		store:   cfg.Store,
		config:  cfg,
		eventCh: make(chan *Event, cfg.BufferSize),
		done:    make(chan struct{}),
		idGen:   cfg.IDGenerator,
	}

	// Start async workers
	for i := 0; i < cfg.Workers; i++ {
		l.wg.Add(1)
		go l.worker()
	}

	return l
}

// worker processes events from the async queue.
func (l *AuditLogger) worker() {
	defer l.wg.Done()

	for {
		select {
		case event := <-l.eventCh:
			l.processEvent(context.Background(), event)
		case <-l.done:
			// Drain remaining events
			for {
				select {
				case event := <-l.eventCh:
					l.processEvent(context.Background(), event)
				default:
					return
				}
			}
		}
	}
}

// processEvent logs a single event.
func (l *AuditLogger) processEvent(ctx context.Context, event *Event) {
	var err error

	// Run before hooks
	l.mu.RLock()
	hooks := l.beforeLog
	l.mu.RUnlock()

	for _, hook := range hooks {
		if hookErr := hook(ctx, event); hookErr != nil {
			err = hookErr
			break
		}
	}

	// Store the event
	if err == nil && l.store != nil {
		err = l.store.Store(ctx, event)
	}

	// Run after hooks
	l.mu.RLock()
	afterHooks := l.afterLog
	l.mu.RUnlock()

	for _, hook := range afterHooks {
		hook(ctx, event, err)
	}
}

// Log records an audit event synchronously.
func (l *AuditLogger) Log(ctx context.Context, event *Event) error {
	l.prepareEvent(event)
	l.processEvent(ctx, event)
	return nil
}

// LogAsync records an audit event asynchronously.
func (l *AuditLogger) LogAsync(ctx context.Context, event *Event) {
	l.prepareEvent(event)

	select {
	case l.eventCh <- event:
		// Event queued
	default:
		// Buffer full, log synchronously as fallback
		l.processEvent(ctx, event)
	}
}

// prepareEvent fills in default values.
func (l *AuditLogger) prepareEvent(event *Event) {
	if event.ID == "" {
		event.ID = l.idGen()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Outcome == "" {
		event.Outcome = OutcomeUnknown
	}

	// Add service context
	if event.Context == nil {
		event.Context = &EventContext{}
	}
	if event.Context.Service == "" {
		event.Context.Service = l.config.Service
	}
	if event.Context.Environment == "" {
		event.Context.Environment = l.config.Environment
	}
	if event.Context.Version == "" {
		event.Context.Version = l.config.Version
	}
}

// Query retrieves audit events.
func (l *AuditLogger) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	if l.store == nil {
		return nil, ErrStoreNotConfigured
	}
	return l.store.Query(ctx, query)
}

// OnBeforeLog adds a hook called before logging.
func (l *AuditLogger) OnBeforeLog(hook BeforeLogHook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.beforeLog = append(l.beforeLog, hook)
}

// OnAfterLog adds a hook called after logging.
func (l *AuditLogger) OnAfterLog(hook AfterLogHook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.afterLog = append(l.afterLog, hook)
}

// Close closes the logger and waits for pending events.
func (l *AuditLogger) Close() error {
	close(l.done)
	l.wg.Wait()
	return nil
}

// Query defines criteria for querying audit logs.
type Query struct {
	// ActorID filters by actor ID.
	ActorID string

	// ActorType filters by actor type.
	ActorType string

	// ResourceID filters by resource ID.
	ResourceID string

	// ResourceType filters by resource type.
	ResourceType string

	// Action filters by action (supports wildcards).
	Action string

	// Actions filters by multiple actions.
	Actions []string

	// Outcome filters by outcome.
	Outcome Outcome

	// Tags filters by tags (all must match).
	Tags []string

	// From is the start of the time range.
	From time.Time

	// To is the end of the time range.
	To time.Time

	// Limit is the maximum number of results.
	Limit int

	// Offset is the number of results to skip.
	Offset int

	// OrderBy is the field to sort by.
	OrderBy string

	// OrderDesc sorts in descending order.
	OrderDesc bool
}

// QueryResult contains query results.
type QueryResult struct {
	// Events are the matching events.
	Events []*Event

	// Total is the total number of matching events.
	Total int

	// HasMore indicates if there are more results.
	HasMore bool
}

// EventBuilder provides a fluent API for creating events.
type EventBuilder struct {
	event *Event
}

// NewEvent creates a new event builder.
func NewEvent(action string) *EventBuilder {
	return &EventBuilder{
		event: &Event{
			Action:   action,
			Metadata: make(map[string]any),
		},
	}
}

// Success sets the outcome to success.
func (b *EventBuilder) Success() *EventBuilder {
	b.event.Outcome = OutcomeSuccess
	return b
}

// Failure sets the outcome to failure with a reason.
func (b *EventBuilder) Failure(reason string) *EventBuilder {
	b.event.Outcome = OutcomeFailure
	b.event.Reason = reason
	return b
}

// WithActor sets the actor.
func (b *EventBuilder) WithActor(actor *Actor) *EventBuilder {
	b.event.Actor = actor
	return b
}

// WithActorUser sets a user actor.
func (b *EventBuilder) WithActorUser(id, name, email string) *EventBuilder {
	b.event.Actor = &Actor{
		ID:    id,
		Type:  "user",
		Name:  name,
		Email: email,
	}
	return b
}

// WithActorService sets a service actor.
func (b *EventBuilder) WithActorService(id, name string) *EventBuilder {
	b.event.Actor = &Actor{
		ID:   id,
		Type: "service",
		Name: name,
	}
	return b
}

// WithActorSystem sets a system actor.
func (b *EventBuilder) WithActorSystem() *EventBuilder {
	b.event.Actor = &Actor{
		ID:   "system",
		Type: "system",
		Name: "System",
	}
	return b
}

// WithResource sets the resource.
func (b *EventBuilder) WithResource(resource *Resource) *EventBuilder {
	b.event.Resource = resource
	return b
}

// WithResourceID sets a resource by type and ID.
func (b *EventBuilder) WithResourceID(resourceType, id string) *EventBuilder {
	b.event.Resource = &Resource{
		ID:   id,
		Type: resourceType,
	}
	return b
}

// WithResourceName sets a resource with name.
func (b *EventBuilder) WithResourceName(resourceType, id, name string) *EventBuilder {
	b.event.Resource = &Resource{
		ID:   id,
		Type: resourceType,
		Name: name,
	}
	return b
}

// WithContext sets the event context.
func (b *EventBuilder) WithContext(ctx *EventContext) *EventBuilder {
	b.event.Context = ctx
	return b
}

// WithRequestID sets the request ID.
func (b *EventBuilder) WithRequestID(id string) *EventBuilder {
	if b.event.Context == nil {
		b.event.Context = &EventContext{}
	}
	b.event.Context.RequestID = id
	return b
}

// WithTraceID sets the trace ID.
func (b *EventBuilder) WithTraceID(id string) *EventBuilder {
	if b.event.Context == nil {
		b.event.Context = &EventContext{}
	}
	b.event.Context.TraceID = id
	return b
}

// WithIP sets the actor's IP address.
func (b *EventBuilder) WithIP(ip string) *EventBuilder {
	if b.event.Actor == nil {
		b.event.Actor = &Actor{}
	}
	b.event.Actor.IP = ip
	return b
}

// WithUserAgent sets the actor's user agent.
func (b *EventBuilder) WithUserAgent(ua string) *EventBuilder {
	if b.event.Actor == nil {
		b.event.Actor = &Actor{}
	}
	b.event.Actor.UserAgent = ua
	return b
}

// WithChange records a field change.
func (b *EventBuilder) WithChange(field string, oldValue, newValue any) *EventBuilder {
	b.event.Changes = append(b.event.Changes, Change{
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
	})
	return b
}

// WithChanges records multiple changes.
func (b *EventBuilder) WithChanges(changes []Change) *EventBuilder {
	b.event.Changes = append(b.event.Changes, changes...)
	return b
}

// WithMetadata adds metadata.
func (b *EventBuilder) WithMetadata(key string, value any) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]any)
	}
	b.event.Metadata[key] = value
	return b
}

// WithMetadataMap adds multiple metadata entries.
func (b *EventBuilder) WithMetadataMap(metadata map[string]any) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]any)
	}
	for k, v := range metadata {
		b.event.Metadata[k] = v
	}
	return b
}

// WithTags adds tags.
func (b *EventBuilder) WithTags(tags ...string) *EventBuilder {
	b.event.Tags = append(b.event.Tags, tags...)
	return b
}

// WithReason sets the reason.
func (b *EventBuilder) WithReason(reason string) *EventBuilder {
	b.event.Reason = reason
	return b
}

// Build returns the constructed event.
func (b *EventBuilder) Build() *Event {
	return b.event
}

// Log logs the event using the given logger.
func (b *EventBuilder) Log(ctx context.Context, logger Logger) error {
	return logger.Log(ctx, b.event)
}

// LogAsync logs the event asynchronously.
func (b *EventBuilder) LogAsync(ctx context.Context, logger Logger) {
	logger.LogAsync(ctx, b.event)
}

// JSON returns the event as JSON.
func (e *Event) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// String returns the event as a JSON string.
func (e *Event) String() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// Context key for storing audit logger.
type contextKey struct{}

// WithLogger adds an audit logger to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the audit logger from context.
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(contextKey{}).(Logger); ok {
		return logger
	}
	return nil
}

// LogFromContext logs an event using the logger from context.
func LogFromContext(ctx context.Context, event *Event) error {
	logger := FromContext(ctx)
	if logger == nil {
		return ErrLoggerNotFound
	}
	return logger.Log(ctx, event)
}

// Global logger for convenience.
var (
	globalLogger Logger
	globalMu     sync.RWMutex
)

// SetGlobal sets the global audit logger.
func SetGlobal(logger Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// Global returns the global audit logger.
func Global() Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// Log logs an event using the global logger.
func Log(ctx context.Context, event *Event) error {
	logger := Global()
	if logger == nil {
		return ErrLoggerNotFound
	}
	return logger.Log(ctx, event)
}

// LogAsync logs an event asynchronously using the global logger.
func LogAsync(ctx context.Context, event *Event) {
	logger := Global()
	if logger != nil {
		logger.LogAsync(ctx, event)
	}
}

// DefaultIDGenerator generates UUIDs.
func DefaultIDGenerator() string {
	return generateUUID()
}

// generateUUID generates a simple UUID v4.
func generateUUID() string {
	b := make([]byte, 16)
	// Use time-based seed for simplicity (not cryptographically secure)
	t := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(t >> (i * 4))
		t = t*1103515245 + 12345
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return formatUUID(b)
}

func formatUUID(b []byte) string {
	const hex = "0123456789abcdef"
	buf := make([]byte, 36)

	buf[8] = '-'
	buf[13] = '-'
	buf[18] = '-'
	buf[23] = '-'

	idx := 0
	for i, v := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			idx++
		}
		buf[idx] = hex[v>>4]
		buf[idx+1] = hex[v&0x0f]
		idx += 2
	}

	return string(buf)
}
