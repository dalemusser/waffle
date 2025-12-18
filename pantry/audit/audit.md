# audit - Audit Logging

The `audit` package provides comprehensive audit logging for Go applications, tracking who did what, when, and to what resources.

## Features

- Structured audit events with actors, resources, and outcomes
- Change tracking for resource modifications
- Multiple storage backends (memory, file, writer, channel)
- Asynchronous logging with buffering
- HTTP middleware for automatic request auditing
- Query API for searching audit logs
- Fluent event builder API
- Thread-safe design

## Installation

```go
import "github.com/yourusername/waffle/audit"
```

## Quick Start

```go
package main

import (
    "context"
    "net/http"

    "github.com/yourusername/waffle/audit"
)

func main() {
    // Create audit logger with memory store
    store := audit.NewMemoryStore(10000)
    logger := audit.NewLogger(audit.Config{
        Store:       store,
        Service:     "my-app",
        Environment: "production",
    })
    defer logger.Close()

    // Set as global logger
    audit.SetGlobal(logger)

    // Log an event
    audit.NewEvent("user.login").
        WithActorUser("user-123", "John Doe", "john@example.com").
        WithIP("192.168.1.1").
        Success().
        LogAsync(context.Background(), logger)

    // Use middleware for HTTP audit logging
    mw := audit.Middleware(audit.DefaultMiddlewareConfig(logger))

    http.Handle("/", mw(http.HandlerFunc(handler)))
    http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Audit specific actions
    auditor := audit.NewRequestAuditor(audit.Global(), r)
    auditor.LogCreate("document", "doc-456")

    w.Write([]byte("OK"))
}
```

## Event Structure

Every audit event contains:

```go
type Event struct {
    ID        string         // Unique event identifier
    Timestamp time.Time      // When the event occurred
    Action    string         // What happened (e.g., "user.login")
    Actor     *Actor         // Who performed the action
    Resource  *Resource      // What was acted upon
    Context   *EventContext  // Additional context
    Outcome   Outcome        // success, failure, pending, unknown
    Reason    string         // Explanation (especially for failures)
    Changes   []Change       // What was modified
    Metadata  map[string]any // Additional data
    Tags      []string       // Categorization tags
}
```

### Actor

Who performed the action:

```go
type Actor struct {
    ID        string         // Unique identifier
    Type      string         // "user", "service", "system"
    Name      string         // Human-readable name
    Email     string         // Email address
    IP        string         // IP address
    UserAgent string         // User agent string
    SessionID string         // Session identifier
    Metadata  map[string]any // Additional data
}
```

### Resource

What was acted upon:

```go
type Resource struct {
    ID       string         // Unique identifier
    Type     string         // "user", "document", "order"
    Name     string         // Human-readable name
    Metadata map[string]any // Additional data
}
```

### Outcome

Result of the action:

```go
const (
    OutcomeSuccess Outcome = "success"
    OutcomeFailure Outcome = "failure"
    OutcomePending Outcome = "pending"
    OutcomeUnknown Outcome = "unknown"
)
```

## Event Builder

Use the fluent builder API:

```go
// Simple success event
audit.NewEvent("user.login").
    WithActorUser("user-123", "John Doe", "john@example.com").
    Success().
    Log(ctx, logger)

// Failure with reason
audit.NewEvent("payment.process").
    WithActorUser("user-123", "John", "").
    WithResourceID("payment", "pay-789").
    Failure("insufficient funds").
    LogAsync(ctx, logger)

// Resource modification with changes
audit.NewEvent("profile.update").
    WithActorUser("user-123", "John", "john@example.com").
    WithResourceID("user", "user-123").
    WithChange("email", "old@example.com", "new@example.com").
    WithChange("name", "John", "Johnny").
    Success().
    Log(ctx, logger)

// Full event with all details
audit.NewEvent("document.share").
    WithActorUser("user-123", "John", "john@example.com").
    WithResourceName("document", "doc-456", "Q4 Report.pdf").
    WithRequestID("req-abc").
    WithTraceID("trace-xyz").
    WithIP("192.168.1.1").
    WithUserAgent("Mozilla/5.0...").
    WithTags("sharing", "external").
    WithMetadata("shared_with", "alice@external.com").
    WithMetadata("permission", "view").
    Success().
    LogAsync(ctx, logger)
```

### Builder Methods

| Method | Description |
|--------|-------------|
| `Success()` | Set outcome to success |
| `Failure(reason)` | Set outcome to failure with reason |
| `WithActor(actor)` | Set full actor |
| `WithActorUser(id, name, email)` | Set user actor |
| `WithActorService(id, name)` | Set service actor |
| `WithActorSystem()` | Set system actor |
| `WithResource(resource)` | Set full resource |
| `WithResourceID(type, id)` | Set resource by type and ID |
| `WithResourceName(type, id, name)` | Set resource with name |
| `WithContext(ctx)` | Set event context |
| `WithRequestID(id)` | Set request ID |
| `WithTraceID(id)` | Set trace ID |
| `WithIP(ip)` | Set actor IP |
| `WithUserAgent(ua)` | Set user agent |
| `WithChange(field, old, new)` | Add a field change |
| `WithChanges(changes)` | Add multiple changes |
| `WithMetadata(key, value)` | Add metadata |
| `WithMetadataMap(map)` | Add multiple metadata |
| `WithTags(tags...)` | Add tags |
| `WithReason(reason)` | Set reason |
| `Build()` | Get the event |
| `Log(ctx, logger)` | Log synchronously |
| `LogAsync(ctx, logger)` | Log asynchronously |

## Logger Configuration

```go
logger := audit.NewLogger(audit.Config{
    // Storage backend
    Store: store,

    // Service identification
    Service:     "my-app",
    Environment: "production",
    Version:     "1.0.0",

    // Async processing
    BufferSize: 1000,  // Event buffer size
    Workers:    2,     // Async workers

    // Custom ID generator
    IDGenerator: func() string {
        return uuid.New().String()
    },
})
```

### Hooks

```go
// Before logging
logger.OnBeforeLog(func(ctx context.Context, event *audit.Event) error {
    // Validate, enrich, or filter events
    if event.Actor == nil {
        return errors.New("actor required")
    }
    return nil
})

// After logging
logger.OnAfterLog(func(ctx context.Context, event *audit.Event, err error) {
    if err != nil {
        log.Printf("Failed to log audit event: %v", err)
    }
})
```

## Storage Backends

### Memory Store

For testing and development:

```go
store := audit.NewMemoryStore(10000) // Max 10000 events
```

### File Store

JSON lines format with rotation:

```go
store, err := audit.NewFileStore(audit.FileStoreConfig{
    Path:     "/var/log/audit/events.jsonl",
    MaxSize:  100 * 1024 * 1024, // 100MB
    Rotation: 5,                  // Keep 5 rotated files
})
```

### Writer Store

Write to any io.Writer:

```go
// Write to stdout
store := audit.NewWriterStore(os.Stdout)

// Write to custom writer
store := audit.NewWriterStore(myWriter)
```

### Channel Store

Stream events to a channel:

```go
ch := make(chan *audit.Event, 100)
store := audit.NewChannelStore(ch)

// Process events from channel
go func() {
    for event := range ch {
        // Send to external system
        sendToSplunk(event)
    }
}()
```

### Multi Store

Write to multiple backends:

```go
store := audit.NewMultiStore(
    audit.NewMemoryStore(1000),        // Recent events in memory
    fileStore,                          // Persistent storage
    audit.NewChannelStore(streamCh),   // Real-time streaming
)
```

### Batch Store

Batch events for efficiency:

```go
store := audit.NewBatchStore(
    fileStore,
    100,            // Batch size
    time.Second,    // Flush interval
)
```

### Filter Store

Filter events before storing:

```go
store := audit.NewFilterStore(fileStore, func(e *audit.Event) bool {
    // Only store non-GET requests
    if method, ok := e.Metadata["http.method"].(string); ok {
        return method != "GET"
    }
    return true
})
```

### Transform Store

Transform events before storing:

```go
store := audit.NewTransformStore(fileStore, func(e *audit.Event) *audit.Event {
    // Redact sensitive data
    if e.Actor != nil {
        e.Actor.IP = maskIP(e.Actor.IP)
    }
    return e
})
```

### Null Store

Discard all events:

```go
store := audit.NewNullStore()
```

## HTTP Middleware

Automatically audit HTTP requests:

```go
cfg := audit.MiddlewareConfig{
    Logger: logger,

    // Skip certain paths
    SkipPaths: []string{"/health", "/metrics"},

    // Skip read-only methods
    SkipMethods: []string{"GET", "HEAD", "OPTIONS"},

    // Capture request/response bodies
    CaptureRequestBody:  true,
    CaptureResponseBody: true,
    MaxBodySize:         10 * 1024, // 10KB

    // Header configuration
    UserIDHeader:    "X-User-ID",
    RequestIDHeader: "X-Request-ID",

    // Add tags to all events
    Tags: []string{"api"},
}

mw := audit.Middleware(cfg)
router.Use(mw)
```

### Custom Actor Extraction

```go
cfg := audit.MiddlewareConfig{
    Logger: logger,
    ActorExtractor: func(r *http.Request) *audit.Actor {
        // Extract from JWT or session
        user := getUserFromContext(r.Context())
        return &audit.Actor{
            ID:    user.ID,
            Type:  "user",
            Name:  user.Name,
            Email: user.Email,
        }
    },
}
```

### Custom Action Mapping

```go
cfg := audit.MiddlewareConfig{
    Logger: logger,
    ActionMapper: func(r *http.Request) string {
        // Map routes to actions
        switch {
        case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/users"):
            return "user.create"
        case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/users"):
            return "user.delete"
        default:
            return r.Method + " " + r.URL.Path
        }
    },
}
```

### Conditional Auditing

```go
cfg := audit.MiddlewareConfig{
    Logger: logger,
    ShouldAudit: func(r *http.Request) bool {
        // Only audit authenticated requests
        return r.Header.Get("Authorization") != ""
    },
}
```

## Request Auditor

Audit specific actions within handlers:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    auditor := audit.NewRequestAuditor(audit.Global(), r)

    // Set actor from context
    user := getUserFromContext(r.Context())
    auditor.WithActor(&audit.Actor{
        ID:    user.ID,
        Type:  "user",
        Name:  user.Name,
        Email: user.Email,
    })

    // Audit CRUD operations
    auditor.LogCreate("document", "doc-123")
    auditor.LogRead("document", "doc-123")
    auditor.LogUpdate("document", "doc-123", []audit.Change{
        {Field: "title", OldValue: "Old Title", NewValue: "New Title"},
    })
    auditor.LogDelete("document", "doc-123")

    // Custom action
    auditor.Log("document.share",
        &audit.Resource{Type: "document", ID: "doc-123"},
        audit.OutcomeSuccess,
        map[string]any{"shared_with": "alice@example.com"},
    )
}
```

## Querying Audit Logs

```go
// Query by actor
result, _ := logger.Query(ctx, &audit.Query{
    ActorID: "user-123",
    Limit:   100,
})

// Query by resource
result, _ := logger.Query(ctx, &audit.Query{
    ResourceType: "document",
    ResourceID:   "doc-456",
})

// Query by action (supports wildcards)
result, _ := logger.Query(ctx, &audit.Query{
    Action: "user.*",  // All user actions
})

// Query by time range
result, _ := logger.Query(ctx, &audit.Query{
    From: time.Now().Add(-24 * time.Hour),
    To:   time.Now(),
})

// Query by outcome
result, _ := logger.Query(ctx, &audit.Query{
    Outcome: audit.OutcomeFailure,
})

// Combined query with pagination
result, _ := logger.Query(ctx, &audit.Query{
    ActorID:      "user-123",
    Actions:      []string{"document.create", "document.update"},
    From:         time.Now().Add(-7 * 24 * time.Hour),
    Outcome:      audit.OutcomeSuccess,
    Limit:        50,
    Offset:       100,
    OrderBy:      "timestamp",
    OrderDesc:    true,
})

// Results
for _, event := range result.Events {
    fmt.Printf("%s: %s by %s\n",
        event.Timestamp.Format(time.RFC3339),
        event.Action,
        event.Actor.Name,
    )
}
fmt.Printf("Total: %d, HasMore: %v\n", result.Total, result.HasMore)
```

## Admin API

REST API for querying audit logs:

```go
admin := audit.NewAdminHandler(logger)
router.Mount("/admin/audit", admin)
```

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Query with URL params |
| POST | `/` | Query with JSON body |

**Examples:**

```bash
# Query by actor
curl "http://localhost:8080/admin/audit?actor_id=user-123"

# Query by time range
curl "http://localhost:8080/admin/audit?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z"

# Complex query
curl -X POST http://localhost:8080/admin/audit \
  -H "Content-Type: application/json" \
  -d '{
    "actor_id": "user-123",
    "resource_type": "document",
    "outcome": "success",
    "from": "2024-01-01T00:00:00Z",
    "limit": 50,
    "order_desc": true
  }'
```

## Common Actions

Pre-defined action constants:

```go
const (
    // User actions
    ActionUserLogin         = "user.login"
    ActionUserLogout        = "user.logout"
    ActionUserCreate        = "user.create"
    ActionUserUpdate        = "user.update"
    ActionUserDelete        = "user.delete"
    ActionUserPasswordReset = "user.password_reset"

    // Permission actions
    ActionPermissionGrant  = "permission.grant"
    ActionPermissionRevoke = "permission.revoke"

    // Resource actions
    ActionResourceCreate = "resource.create"
    ActionResourceRead   = "resource.read"
    ActionResourceUpdate = "resource.update"
    ActionResourceDelete = "resource.delete"

    // Other
    ActionSettingChange = "setting.change"
    ActionAPIKeyCreate  = "api_key.create"
    ActionAPIKeyRevoke  = "api_key.revoke"
    ActionExport        = "data.export"
    ActionImport        = "data.import"
)
```

## Helper Functions

```go
// Log user login
audit.LogUserLogin(ctx, logger,
    "user-123", "John Doe",
    "192.168.1.1", "Mozilla/5.0...",
    true, // success
)

// Log user logout
audit.LogUserLogout(ctx, logger, "user-123", "John Doe")

// Log resource change with diff
audit.LogResourceChange(ctx, logger,
    actor,
    "document", "doc-123",
    []audit.Change{
        {Field: "title", OldValue: "Old", NewValue: "New"},
    },
)

// Compute diff between two maps
oldData := map[string]any{"name": "John", "email": "john@old.com"}
newData := map[string]any{"name": "John", "email": "john@new.com"}
changes := audit.DiffChanges(oldData, newData)
// [Change{Field: "email", OldValue: "john@old.com", NewValue: "john@new.com"}]
```

## Context Integration

```go
// Add logger to context
ctx := audit.WithLogger(ctx, logger)

// Get logger from context
logger := audit.FromContext(ctx)

// Log using context
audit.LogFromContext(ctx, event)
```

## Global Logger

```go
// Set global logger
audit.SetGlobal(logger)

// Get global logger
logger := audit.Global()

// Log using global
audit.Log(ctx, event)
audit.LogAsync(ctx, event)
```

## Best Practices

1. **Use structured actions**: `resource.action` format (e.g., `user.create`, `document.share`)

2. **Always include actor**: Who performed the action

3. **Track changes**: Record before/after values for updates

4. **Use async logging**: Avoid blocking application flow

5. **Filter sensitive data**: Use TransformStore to redact PII

6. **Set up retention**: Use FileStore rotation or implement cleanup

7. **Monitor audit failures**: Use OnAfterLog hook

8. **Include request context**: Request ID, trace ID for correlation

## Thread Safety

All logger and store operations are thread-safe:

```go
// Safe concurrent logging
go logger.LogAsync(ctx, event1)
go logger.LogAsync(ctx, event2)
go logger.Query(ctx, query)
```

## Performance

- Async logging with configurable buffer and workers
- Batch store for high-throughput scenarios
- Memory store for development/testing
- File store with rotation for production
