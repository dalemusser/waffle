# router

Pre-configured Chi router with WAFFLE's standard middleware stack.

## Overview

The `router` package provides a factory function that creates a Chi router pre-wired with common middleware. This gives you a consistent baseline for HTTP handling across WAFFLE applications: request IDs, real IP detection, panic recovery, body limits, metrics, and request logging.

## Import

```go
import "github.com/dalemusser/waffle/router"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    // Create router with standard middleware
    r := router.New(core, logger)

    // Add your routes
    r.Get("/", homeHandler)
    r.Route("/api", func(r chi.Router) {
        r.Get("/users", listUsers)
        r.Post("/users", createUser)
    })

    return r, nil
}
```

## API

### New

**Location:** `router.go`

```go
func New(coreCfg *config.CoreConfig, logger *zap.Logger) chi.Router
```

Creates a Chi router with WAFFLE's standard middleware stack pre-configured.

**Middleware stack (in order):**

| Middleware | Source | Description |
|------------|--------|-------------|
| RequestID | chi | Generates unique request ID for each request |
| RealIP | chi | Extracts real client IP from proxy headers |
| Recoverer | logging | Catches panics, logs stack trace, returns 500 |
| LimitBodySize | middleware | Limits request body size (from `MaxRequestBodyBytes`) |
| HTTPMetrics | metrics | Records request duration for Prometheus |
| RequestLogger | logging | Logs each request with method, path, status, latency |

**Also configured:**
- `NotFound` handler returns JSON error response
- `MethodNotAllowed` handler returns JSON error response

**Not included (app-level decisions):**
- Health check endpoints
- Version endpoint
- pprof endpoints
- CORS middleware
- Authentication

## What You Get

### Request IDs

Every request gets a unique ID available via `middleware.GetReqID(r.Context())`. This ID is included in log entries for correlation.

### Real IP Detection

Client IP is extracted from `X-Forwarded-For`, `X-Real-IP`, or the connection's remote address, in that order.

### Panic Recovery

If a handler panics, the request returns HTTP 500 and the panic is logged with a full stack trace. The server stays running.

### Body Size Limits

Request bodies are limited to `CoreConfig.MaxRequestBodyBytes`. Set to 0 to disable.

### Metrics

Request durations are recorded in the `http_request_duration_seconds` histogram. Mount `/metrics` to expose to Prometheus.

### Request Logging

Each request is logged with:
- HTTP method and path
- Response status code and body size
- Request latency
- Client IP and user agent
- Request ID

### JSON Error Responses

404 and 405 responses return JSON:

```json
{"error": "not_found", "message": "The requested resource was not found"}
```

```json
{"error": "method_not_allowed", "message": "The requested HTTP method is not allowed for this resource"}
```

## Patterns

### Basic Application

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(core, logger)

    // Health check
    health.Mount(r, nil, logger)

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/items", listItems)
        r.Post("/items", createItem)
        r.Get("/items/{id}", getItem)
    })

    return r, nil
}
```

### With CORS

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(core, logger)

    // Add CORS after standard middleware
    r.Use(middleware.CORSFromConfig(core))

    // Routes...
    return r, nil
}
```

### With Authentication

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(core, logger)

    // Public routes
    r.Get("/", homeHandler)
    health.Mount(r, nil, logger)

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(googleAuth.RequireAuth("/login"))
        r.Get("/dashboard", dashboardHandler)
        r.Get("/profile", profileHandler)
    })

    // Admin routes
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.AdminKey, apikey.Options{Realm: "admin"}, logger))
        r.Mount("/admin", adminRoutes())
        pprof.Mount(r)
    })

    return r, nil
}
```

### Full-Featured Setup

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    // Register Prometheus collectors
    metrics.RegisterDefault(logger)

    // Create router with standard middleware
    r := router.New(core, logger)

    // CORS
    r.Use(middleware.CORSFromConfig(core))

    // Health endpoints
    health.MountAt(r, "/live", nil, logger)
    health.MountAt(r, "/ready", map[string]health.Check{
        "mongo": func(ctx context.Context) error {
            return db.Mongo.Ping(ctx, readpref.Primary())
        },
    }, logger)

    // Metrics endpoint
    r.Handle("/metrics", metrics.Handler())

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(middleware.RequireJSON())
        r.Route("/items", itemRoutes(db, logger))
    })

    // Protected debug endpoints
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.DebugKey, apikey.Options{
            Realm:      "debug",
            CookieName: "debug_auth",
        }, logger))
        pprof.Mount(r)
    })

    return r, nil
}
```

### Manual Router Setup

If you need more control, build the router manually instead of using `router.New`:

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Pick and choose middleware
    r.Use(chimw.RequestID)
    r.Use(logging.Recoverer(logger))
    r.Use(logging.RequestLogger(logger))

    // Skip metrics, body limits, etc.

    // Custom 404 handler
    r.NotFound(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "not found", 404)
    })

    // Routes...
    return r, nil
}
```

## See Also

- [config](../config/config.md) — CoreConfig and MaxRequestBodyBytes
- [logging](../logging/logging.md) — Recoverer and RequestLogger
- [metrics](../metrics/metrics.md) — HTTPMetrics middleware
- [middleware](../middleware/middleware.md) — LimitBodySize and error handlers
- [Chi documentation](https://go-chi.io/) — Router API reference

