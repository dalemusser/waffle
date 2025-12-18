# requestid

Request ID middleware for distributed tracing in WAFFLE applications.

## Overview

The `requestid` package provides:
- **Middleware** — Assigns unique IDs to HTTP requests
- **Context helpers** — Get/set request IDs in context
- **Propagation** — Pass request IDs to downstream services
- **Logger integration** — Add request IDs to log entries

## Import

```go
import "github.com/dalemusser/waffle/requestid"
```

---

## Quick Start

```go
r := chi.NewRouter()

// Add request ID middleware
r.Use(requestid.Simple())

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    // Get request ID
    id := requestid.FromRequest(r)

    // Log with request ID
    logger := requestid.Logger(r.Context(), logger)
    logger.Info("handling request")
})
```

---

## Middleware

### Simple

**Location:** `requestid.go`

```go
func Simple() func(http.Handler) http.Handler
```

Returns middleware with default settings:
- Header: `X-Request-ID`
- Generator: UUID v4
- Trusts incoming header
- Adds ID to response header

### Middleware with Config

```go
func Middleware(cfg Config) func(http.Handler) http.Handler
```

**Config:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Header | string | "X-Request-ID" | HTTP header name |
| Generator | func() string | GenerateUUID | ID generator function |
| TrustProxy | bool | true | Trust incoming request ID |
| Validator | func(string) bool | length check | Validate incoming IDs |
| SetResponseHeader | bool | true | Add ID to response |

### Examples

```go
// Default configuration
r.Use(requestid.Simple())

// Custom header
r.Use(requestid.Middleware(requestid.Config{
    Header: "X-Correlation-ID",
}))

// Don't trust incoming headers (always generate new)
r.Use(requestid.Middleware(requestid.Config{
    TrustProxy: false,
}))

// Short IDs
r.Use(requestid.Middleware(requestid.Config{
    Generator: requestid.GenerateShort,
}))

// Prefixed IDs
r.Use(requestid.Middleware(requestid.Config{
    Generator: requestid.GeneratePrefixed("api-"),
}))

// Sequential IDs (useful for debugging)
r.Use(requestid.Middleware(requestid.Config{
    Generator: requestid.GenerateSequential("req"),
}))

// Strict UUID validation
r.Use(requestid.Middleware(requestid.Config{
    Validator: requestid.ValidateUUID,
}))
```

---

## Getting Request IDs

### From Context

```go
// Get from context
id := requestid.Get(ctx)

// Get from request
id := requestid.FromRequest(r)
```

### Setting Manually

```go
// Add request ID to context (for background jobs, etc.)
ctx = requestid.Set(ctx, "custom-request-id")
```

---

## ID Generators

**Location:** `requestid.go`

| Generator | Format | Example |
|-----------|--------|---------|
| `GenerateUUID` | UUID v4 | `550e8400-e29b-41d4-a716-446655440000` |
| `GenerateShort` | 16 hex chars | `a1b2c3d4e5f6g7h8` |
| `GenerateTimestamp` | Timestamp + random | `0123456789abcdef01234567` |
| `GenerateSequential(prefix)` | Prefix + timestamp + counter | `req-12345678-00000001` |
| `GeneratePrefixed(prefix)` | Prefix + UUID | `api-550e8400-e29b-...` |

### Custom Generator

```go
r.Use(requestid.Middleware(requestid.Config{
    Generator: func() string {
        return "myapp-" + time.Now().Format("20060102-150405") + "-" + randomSuffix()
    },
}))
```

---

## Validators

Validators check incoming request IDs before trusting them:

```go
// UUID format only
requestid.ValidateUUID

// Hexadecimal only
requestid.ValidateHex

// Alphanumeric with hyphens/underscores
requestid.ValidateAlphanumeric

// Custom validator
func(s string) bool {
    return strings.HasPrefix(s, "myapp-") && len(s) <= 64
}
```

---

## Logger Integration

**Location:** `logger.go`

### Add Request ID to Logger

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Create logger with request ID
    log := requestid.Logger(r.Context(), logger)

    log.Info("processing request")
    // Output: {"level":"info","msg":"processing request","request_id":"550e8400-..."}
}
```

### As a Field

```go
logger.Info("event happened",
    requestid.Field(ctx),
    zap.String("user_id", userID),
)
```

---

## Propagation

**Location:** `propagate.go`

### HTTP Client

Automatically add request IDs to outgoing requests:

```go
// Create client that propagates request IDs
client := requestid.Client()

// Make request - ID is automatically added from context
req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
resp, err := client.Do(req)
```

### With Custom Base Transport

```go
client := requestid.ClientWithBase(&http.Transport{
    MaxIdleConns: 100,
})
```

### Manual Propagation

```go
// Add to existing request
req, _ := http.NewRequest("GET", url, nil)
requestid.SetHeader(ctx, req)

// With custom header
requestid.SetHeaderWithName(ctx, req, "X-Correlation-ID")
```

### Extract from Response

```go
resp, _ := client.Do(req)
responseID := requestid.ExtractFromResponse(resp)
```

---

## WAFFLE Integration

### Complete Setup

```go
func main() {
    logger, _ := zap.NewProduction()

    r := chi.NewRouter()

    // Add request ID first (before logging middleware)
    r.Use(requestid.Simple())

    // Logging middleware uses request ID
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            log := requestid.Logger(r.Context(), logger)
            log.Info("request started",
                zap.String("method", r.Method),
                zap.String("path", r.URL.Path),
            )
            next.ServeHTTP(w, r)
        })
    })

    r.Get("/", handler)
    http.ListenAndServe(":8080", r)
}
```

### Propagating to Background Jobs

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Get request ID
    requestID := requestid.FromRequest(r)

    // Pass to background job
    jobs.Enqueue(&jobs.Job{
        Type: "process",
        Payload: JobPayload{
            RequestID: requestID,
            Data:      data,
        },
    })
}

func processJob(ctx context.Context, job *jobs.Job) error {
    payload := job.Payload.(JobPayload)

    // Add request ID to context
    ctx = requestid.Set(ctx, payload.RequestID)

    // Now all logs will include the original request ID
    log := requestid.Logger(ctx, logger)
    log.Info("processing job")

    return nil
}
```

### Calling External Services

```go
func callExternalAPI(ctx context.Context, data any) error {
    client := requestid.Client()

    body, _ := json.Marshal(data)
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.example.com", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // Request ID is automatically propagated
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```

### Microservices Tracing

```go
// Service A (gateway)
r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := requestid.Logger(ctx, logger)

    // Call Service B - request ID propagates automatically
    client := requestid.Client()
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://user-service/users/"+id, nil)
    resp, _ := client.Do(req)

    log.Info("fetched user from service B")
})

// Service B (user service)
r.Use(requestid.Simple()) // Trusts incoming X-Request-ID

r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    log := requestid.Logger(r.Context(), logger)
    log.Info("handling user request")
    // Logs show same request_id as Service A
})
```

---

## Headers

Common request ID header names:

| Header | Usage |
|--------|-------|
| `X-Request-ID` | General purpose (default) |
| `X-Correlation-ID` | Correlation across services |
| `X-Trace-ID` | Distributed tracing |
| `Request-Id` | Azure/Microsoft style |

```go
// AWS/Azure style
r.Use(requestid.Middleware(requestid.Config{
    Header: "X-Correlation-ID",
}))
```

---

## See Also

- [logging](../logging/logging.md) — Request logging middleware
- [middleware](../middleware/middleware.md) — HTTP middleware
- [jobs](../jobs/jobs.md) — Background job processing
