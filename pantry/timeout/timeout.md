# timeout

Request timeout middleware and utilities for WAFFLE applications.

## Overview

The `timeout` package provides:
- **Middleware** — Enforces request timeouts with graceful handling
- **Context helpers** — Work with deadlines and remaining time
- **HTTP clients** — Pre-configured clients with sensible timeouts
- **Skippers** — Skip timeout for WebSocket, SSE, or specific paths

## Import

```go
import "github.com/dalemusser/waffle/timeout"
```

---

## Quick Start

```go
r := chi.NewRouter()

// Add 30-second timeout to all requests
r.Use(timeout.Simple())

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    // Check remaining time
    remaining := timeout.Remaining(r.Context())
    log.Printf("Time remaining: %v", remaining)

    // Long operation respects context
    result, err := doWork(r.Context())
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(result)
})
```

---

## Middleware

### Simple

**Location:** `timeout.go`

```go
func Simple() func(http.Handler) http.Handler
```

Returns middleware with 30-second timeout and default error handling.

### WithTimeout

```go
func WithTimeout(d time.Duration) func(http.Handler) http.Handler
```

Returns middleware with a custom timeout duration.

### Middleware with Config

```go
func Middleware(cfg Config) func(http.Handler) http.Handler
```

**Config:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Timeout | time.Duration | 30s | Maximum request duration |
| OnTimeout | func(w, r) | nil | Custom timeout handler |
| Skipper | func(r) bool | nil | Skip timeout for certain requests |
| ErrorHandler | func(w, r, err) | nil | Handle panics during request |

### Examples

```go
// Default 30-second timeout
r.Use(timeout.Simple())

// Custom timeout
r.Use(timeout.WithTimeout(60 * time.Second))

// Full configuration
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 10 * time.Second,
    OnTimeout: func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusGatewayTimeout)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "request timed out",
        })
    },
    Skipper: timeout.SkipWebSocket,
}))
```

---

## Skippers

Skip timeout for specific request types:

```go
// Skip WebSocket upgrades
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 30 * time.Second,
    Skipper: timeout.SkipWebSocket,
}))

// Skip Server-Sent Events
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 30 * time.Second,
    Skipper: timeout.SkipSSE,
}))

// Skip specific paths
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 30 * time.Second,
    Skipper: timeout.SkipPaths("/ws", "/events", "/long-poll"),
}))

// Skip certain HTTP methods
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 30 * time.Second,
    Skipper: timeout.SkipMethods("OPTIONS"),
}))

// Combine skippers
r.Use(timeout.Middleware(timeout.Config{
    Timeout: 30 * time.Second,
    Skipper: timeout.CombineSkippers(
        timeout.SkipWebSocket,
        timeout.SkipSSE,
        timeout.SkipPaths("/health"),
    ),
}))
```

---

## Context Helpers

**Location:** `context.go`

### Checking Deadline

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Get deadline
    deadline, ok := timeout.FromContext(ctx)
    if ok {
        log.Printf("Request deadline: %v", deadline)
    }

    // Check if deadline exists
    if timeout.HasDeadline(ctx) {
        log.Println("Request has a deadline")
    }

    // Check remaining time
    remaining := timeout.Remaining(ctx)
    log.Printf("Time remaining: %v", remaining)

    // Check if already expired
    if timeout.IsExpired(ctx) {
        return // Don't bother processing
    }
}
```

### Modifying Timeouts

```go
// Use shorter of existing deadline or new timeout
ctx, cancel := timeout.WithShorter(ctx, 5*time.Second)
defer cancel()

// Reserve time for cleanup (shrink remaining by 2 seconds)
ctx, cancel := timeout.ShrinkBy(ctx, 2*time.Second)
defer cancel()

// Propagate timeout to child operation
ctx, cancel := timeout.PropagateTimeout(ctx)
defer cancel()
```

### Running with Timeout

```go
// Run a function with timeout
err := timeout.Run(ctx, 5*time.Second, func(ctx context.Context) error {
    return doExpensiveOperation(ctx)
})
if err == context.DeadlineExceeded {
    log.Println("Operation timed out")
}

// Run with result
result, err := timeout.RunWithResult(ctx, 5*time.Second, func(ctx context.Context) (Data, error) {
    return fetchData(ctx)
})
```

---

## HTTP Clients

**Location:** `client.go`

### Pre-configured Clients

```go
// Default client (30s total, 10s dial/TLS/headers)
client := timeout.Client()

// Quick client for internal services (5s total)
client := timeout.QuickClient()

// Long client for uploads/reports (5 min total)
client := timeout.LongClient()
```

### Custom Configuration

```go
client := timeout.NewClient(timeout.ClientConfig{
    Timeout:               60 * time.Second,  // Total request timeout
    DialTimeout:           10 * time.Second,  // Connection timeout
    TLSHandshakeTimeout:   10 * time.Second,  // TLS handshake timeout
    ResponseHeaderTimeout: 15 * time.Second,  // Time to first byte
    IdleConnTimeout:       90 * time.Second,  // Idle connection lifetime
    MaxIdleConns:          100,               // Max idle connections
    MaxIdleConnsPerHost:   10,                // Max idle per host
    MaxConnsPerHost:       0,                 // Max total per host (0=unlimited)
    ExpectContinueTimeout: 1 * time.Second,   // 100-continue timeout
})
```

### Per-Request Timeout

```go
// Using transport wrapper
transport := timeout.WithTransportTimeout(http.DefaultTransport, 10*time.Second)
client := &http.Client{Transport: transport}

// Using Do helper
resp, err := timeout.Do(ctx, client, req, 5*time.Second)

// Simple GET with timeout
resp, err := timeout.Get(ctx, "https://api.example.com/data", 10*time.Second)
```

---

## WAFFLE Integration

### Complete Setup

```go
func main() {
    r := chi.NewRouter()

    // Global timeout with exceptions
    r.Use(timeout.Middleware(timeout.Config{
        Timeout: 30 * time.Second,
        OnTimeout: func(w http.ResponseWriter, r *http.Request) {
            log.Printf("Request timed out: %s %s", r.Method, r.URL.Path)
            w.WriteHeader(http.StatusServiceUnavailable)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "request timeout",
            })
        },
        Skipper: timeout.CombineSkippers(
            timeout.SkipWebSocket,
            timeout.SkipSSE,
        ),
    }))

    r.Get("/api/data", dataHandler)
    r.Get("/events", sseHandler)      // No timeout (skipped)
    r.Get("/ws", websocketHandler)    // No timeout (skipped)

    http.ListenAndServe(":8080", r)
}
```

### Different Timeouts per Route

```go
r := chi.NewRouter()

// Default timeout
r.Use(timeout.WithTimeout(30 * time.Second))

// Quick endpoints
r.Group(func(r chi.Router) {
    r.Use(timeout.WithTimeout(5 * time.Second))
    r.Get("/health", healthHandler)
    r.Get("/ping", pingHandler)
})

// Slow endpoints
r.Group(func(r chi.Router) {
    r.Use(timeout.WithTimeout(5 * time.Minute))
    r.Post("/upload", uploadHandler)
    r.Get("/reports/{id}", reportHandler)
})
```

### Timeout-Aware Handlers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Check if we have enough time for the operation
    if timeout.Remaining(ctx) < 5*time.Second {
        http.Error(w, "insufficient time for operation", http.StatusServiceUnavailable)
        return
    }

    // Reserve time for response writing
    ctx, cancel := timeout.ShrinkBy(ctx, 1*time.Second)
    defer cancel()

    // Do work with remaining time
    result, err := doWork(ctx)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            // Already handled by middleware
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }

    json.NewEncoder(w).Encode(result)
}
```

### Calling External Services

```go
func callExternalAPI(ctx context.Context, data any) (*Response, error) {
    // Use pre-configured client
    client := timeout.Client()

    body, _ := json.Marshal(data)
    req, err := http.NewRequestWithContext(ctx, "POST", "https://api.example.com", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    // Client respects context deadline
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result Response
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

### Parallel Operations with Shared Deadline

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var wg sync.WaitGroup
    results := make(chan Result, 3)
    errors := make(chan error, 3)

    // All operations share the request deadline
    for _, service := range []string{"users", "orders", "inventory"} {
        wg.Add(1)
        go func(svc string) {
            defer wg.Done()
            result, err := callService(ctx, svc) // Respects deadline
            if err != nil {
                errors <- err
                return
            }
            results <- result
        }(service)
    }

    // Wait with timeout awareness
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // All completed
    case <-ctx.Done():
        // Timeout - partial results
    }
}
```

---

## Timeout Values

| Use Case | Recommended |
|----------|-------------|
| Health checks | 2-5 seconds |
| API calls (internal) | 5-10 seconds |
| API calls (external) | 10-30 seconds |
| Database queries | 5-30 seconds |
| File uploads | 1-10 minutes |
| Report generation | 1-5 minutes |
| WebSocket/SSE | No timeout (skip) |

---

## See Also

- [middleware](../middleware/middleware.md) — HTTP middleware
- [retry](../retry/retry.md) — Retry with backoff
- [requestid](../requestid/requestid.md) — Request tracing
