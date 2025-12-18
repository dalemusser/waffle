# middleware

HTTP middleware for CORS, content validation, request limits, and error handling.

## Overview

The `middleware` package provides common HTTP middleware for WAFFLE applications. It includes CORS handling (config-driven or manual), content type validation, request body size limits, and JSON error handlers for 404/405 responses.

## Import

```go
import "github.com/dalemusser/waffle/middleware"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // CORS from configuration
    r.Use(middleware.CORSFromConfig(core))

    // Limit request body size (1MB)
    r.Use(middleware.LimitBodySize(1 << 20))

    // JSON error responses for 404/405
    r.NotFound(middleware.NotFoundHandler(logger))
    r.MethodNotAllowed(middleware.MethodNotAllowedHandler(logger))

    // JSON API routes requiring JSON content type
    r.Route("/api", func(r chi.Router) {
        r.Use(middleware.RequireJSON())
        r.Post("/items", createItem)
    })

    return r, nil
}
```

## CORS

### CORSFromConfig

**Location:** `cors.go`

```go
func CORSFromConfig(coreCfg *config.CoreConfig) func(next http.Handler) http.Handler
```

Returns CORS middleware configured from `CoreConfig.CORS`. If `EnableCORS` is false, returns a no-op middleware, making it safe to call unconditionally.

**Example:**

```go
r.Use(middleware.CORSFromConfig(core))
```

See [config](../config/config.md) for CORS configuration options.

### CORS

**Location:** `cors.go`

```go
func CORS(opts CORSOptions) func(next http.Handler) http.Handler
```

Returns CORS middleware with explicit options. Use when you need custom settings not driven by config.

**CORSOptions:**

```go
type CORSOptions struct {
    AllowedOrigins   []string // Origins allowed to make requests
    AllowedMethods   []string // HTTP methods allowed (default: GET, POST, OPTIONS)
    AllowedHeaders   []string // Headers client can use (default: Accept, Authorization, Content-Type)
    ExposedHeaders   []string // Headers safe to expose to client
    AllowCredentials bool     // Allow cookies/auth (cannot be true with "*" origins)
    MaxAge           int      // Preflight cache duration in seconds (default: 300)
}
```

**Example:**

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins:   []string{"https://app.example.com", "https://admin.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           600,
}))
```

### CORSPermissive

**Location:** `cors.go`

```go
func CORSPermissive() func(next http.Handler) http.Handler
```

Returns a permissive CORS middleware for development or internal APIs.

**Policy:**
- Allow all origins (`*`)
- Allow GET, POST, PUT, PATCH, DELETE, OPTIONS
- Allow common headers including Authorization
- No credentials (required when using `*` origins)
- 5-minute preflight cache

**Warning:** Do not use in production with sensitive APIs.

**Example:**

```go
// Development only
if cfg.Env == "dev" {
    r.Use(middleware.CORSPermissive())
}
```

## Content Validation

### RequireJSON

**Location:** `contenttype.go`

```go
func RequireJSON() func(next http.Handler) http.Handler
```

Middleware that ensures requests have a JSON Content-Type (`application/json` or `*+json`). Returns 415 Unsupported Media Type with a JSON error body if validation fails.

**Accepts:**
- `application/json`
- `application/json; charset=utf-8`
- `application/problem+json`
- Any content type ending in `+json`

**Example:**

```go
r.Route("/api", func(r chi.Router) {
    r.Use(middleware.RequireJSON())
    r.Post("/users", createUser)
    r.Put("/users/{id}", updateUser)
})
```

**Error response:**

```json
{
  "error": "unsupported_media_type",
  "message": "Content-Type must be application/json"
}
```

## Request Limits

### LimitBodySize

**Location:** `sizelimit.go`

```go
func LimitBodySize(maxBytes int64) func(next http.Handler) http.Handler
```

Middleware that limits request body size. If the body exceeds `maxBytes`, reading will fail with an error. If `maxBytes <= 0`, returns a no-op middleware.

Apply early in the middleware chain to prevent processing oversized payloads.

**Example:**

```go
// Global 1MB limit
r.Use(middleware.LimitBodySize(1 << 20))

// Different limits per route group
r.Route("/api", func(r chi.Router) {
    r.Use(middleware.LimitBodySize(100 << 10)) // 100KB for API
    r.Post("/items", createItem)
})

r.Route("/upload", func(r chi.Router) {
    r.Use(middleware.LimitBodySize(50 << 20)) // 50MB for uploads
    r.Post("/", uploadFile)
})
```

## Error Handlers

### NotFoundHandler

**Location:** `notfound.go`

```go
func NotFoundHandler(logger *zap.Logger) http.HandlerFunc
```

Returns a handler for 404 responses that logs the request and returns a JSON error body. Designed for `chi.Router.NotFound()`.

**Example:**

```go
r.NotFound(middleware.NotFoundHandler(logger))
```

**Response:**

```json
{
  "error": "not_found",
  "message": "The requested resource was not found"
}
```

### MethodNotAllowedHandler

**Location:** `notfound.go`

```go
func MethodNotAllowedHandler(logger *zap.Logger) http.HandlerFunc
```

Returns a handler for 405 responses that logs the request and returns a JSON error body. Designed for `chi.Router.MethodNotAllowed()`.

**Example:**

```go
r.MethodNotAllowed(middleware.MethodNotAllowedHandler(logger))
```

**Response:**

```json
{
  "error": "method_not_allowed",
  "message": "The requested HTTP method is not allowed for this resource"
}
```

## Patterns

### Complete API Setup

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Recovery and logging (from logging package)
    r.Use(logging.Recoverer(logger))
    r.Use(logging.RequestLogger(logger))

    // CORS
    r.Use(middleware.CORSFromConfig(core))

    // Global body size limit
    r.Use(middleware.LimitBodySize(1 << 20)) // 1MB

    // JSON error handlers
    r.NotFound(middleware.NotFoundHandler(logger))
    r.MethodNotAllowed(middleware.MethodNotAllowedHandler(logger))

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(middleware.RequireJSON())

        r.Route("/users", func(r chi.Router) {
            r.Get("/", listUsers)
            r.Post("/", createUser)
            r.Get("/{id}", getUser)
            r.Put("/{id}", updateUser)
            r.Delete("/{id}", deleteUser)
        })
    })

    return r, nil
}
```

### Environment-Specific CORS

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Use config-driven CORS in production, permissive in development
    if core.Env == "prod" {
        r.Use(middleware.CORSFromConfig(core))
    } else {
        r.Use(middleware.CORSPermissive())
    }

    // ... routes
    return r, nil
}
```

### Route-Specific Body Limits

```go
r.Route("/api", func(r chi.Router) {
    // Small limit for regular API calls
    r.Use(middleware.LimitBodySize(100 << 10)) // 100KB

    r.Post("/comments", createComment)
    r.Post("/posts", createPost)
})

r.Route("/files", func(r chi.Router) {
    // Larger limit for file uploads
    r.Use(middleware.LimitBodySize(100 << 20)) // 100MB

    r.Post("/upload", uploadFile)
})
```

## See Also

- [config](../config/config.md) — CORS configuration options
- [logging](../logging/logging.md) — Request logging and panic recovery
- [httputil](../httputil/httputil.md) — JSON response helpers
- [auth](../auth/auth.md) — Authentication middleware

