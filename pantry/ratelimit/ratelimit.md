# ratelimit

Token bucket rate limiting for WAFFLE applications.

## Overview

The `ratelimit` package provides HTTP middleware and standalone rate limiters using the token bucket algorithm. Supports per-IP, per-path, or custom key-based rate limiting.

## Import

```go
import "github.com/dalemusser/waffle/ratelimit"
```

---

## Simple

**Location:** `ratelimit.go`

```go
func Simple(rate float64, burst int) func(http.Handler) http.Handler
```

Returns a simple rate limit middleware with IP-based limiting and sensible defaults.

**Parameters:**
- `rate` — Requests allowed per second
- `burst` — Maximum burst size (bucket capacity)

**Example:**

```go
// Allow 10 requests/second with burst of 20 per IP
r.Use(ratelimit.Simple(10, 20))
```

---

## Middleware

**Location:** `ratelimit.go`

```go
func Middleware(cfg Config) func(http.Handler) http.Handler
```

Returns rate limit middleware with full configuration control.

**Example:**

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:       100,
    Burst:      200,
    KeyFunc:    ratelimit.IPKeyFunc,
    TTL:        time.Hour,
    StatusCode: http.StatusTooManyRequests,
    Message:    "rate limit exceeded",
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health"
    },
}))
```

---

## Config

**Location:** `ratelimit.go`

```go
type Config struct {
    Rate       float64                                        // Requests per second
    Burst      int                                            // Maximum burst size
    KeyFunc    KeyFunc                                        // Key extraction function (default: IPKeyFunc)
    TTL        time.Duration                                  // Time to keep inactive keys (default: 1 hour)
    StatusCode int                                            // HTTP status when limited (default: 429)
    Message    string                                         // Response body when limited
    OnLimited  func(w http.ResponseWriter, r *http.Request)   // Custom handler when limited
    Skip       func(r *http.Request) bool                     // Skip rate limiting for request
}
```

---

## DefaultConfig

**Location:** `ratelimit.go`

```go
func DefaultConfig() Config
```

Returns sensible defaults:

| Setting | Default |
|---------|---------|
| Rate | 10 req/s |
| Burst | 20 |
| KeyFunc | IPKeyFunc |
| TTL | 1 hour |
| StatusCode | 429 |
| Message | "rate limit exceeded" |

---

## Key Functions

Key functions extract a rate limit key from HTTP requests.

### IPKeyFunc

**Location:** `ratelimit.go`

```go
func IPKeyFunc(r *http.Request) string
```

Returns the client IP address. Checks `X-Forwarded-For` and `X-Real-IP` headers before falling back to `RemoteAddr`.

### PathKeyFunc

**Location:** `ratelimit.go`

```go
func PathKeyFunc(r *http.Request) string
```

Returns the request path. Use for per-endpoint rate limiting.

### MethodPathKeyFunc

**Location:** `ratelimit.go`

```go
func MethodPathKeyFunc(r *http.Request) string
```

Returns method + path (e.g., "GET /api/users"). Use for per-endpoint-per-method limiting.

### IPPathKeyFunc

**Location:** `ratelimit.go`

```go
func IPPathKeyFunc(r *http.Request) string
```

Returns IP + path. Use for per-IP-per-endpoint limiting.

### Custom Key Function

```go
// Rate limit by API key header
func APIKeyFunc(r *http.Request) string {
    if key := r.Header.Get("X-API-Key"); key != "" {
        return key
    }
    return ratelimit.IPKeyFunc(r)
}

r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:    100,
    Burst:   200,
    KeyFunc: APIKeyFunc,
}))
```

---

## Limiter

**Location:** `ratelimit.go`

```go
func New(rate float64, burst int) *Limiter
```

Creates a standalone rate limiter (not per-key).

**Methods:**

```go
func (l *Limiter) Allow() bool      // Check if one request is allowed
func (l *Limiter) AllowN(n int) bool // Check if n requests are allowed
func (l *Limiter) Tokens() float64   // Get current available tokens
```

**Example:**

```go
limiter := ratelimit.New(10, 20) // 10 req/s, burst of 20

if limiter.Allow() {
    // Process request
} else {
    // Rate limited
}
```

---

## KeyLimiter

**Location:** `ratelimit.go`

```go
func NewKeyLimiter(rate float64, burst int, ttl time.Duration) *KeyLimiter
```

Creates a rate limiter that tracks limits per key with automatic cleanup.

**Methods:**

```go
func (kl *KeyLimiter) Allow(key string) bool      // Check if request for key is allowed
func (kl *KeyLimiter) AllowN(key string, n int) bool // Check if n requests for key are allowed
func (kl *KeyLimiter) Size() int                  // Get number of tracked keys
```

**Example:**

```go
limiter := ratelimit.NewKeyLimiter(10, 20, time.Hour)

if limiter.Allow(userID) {
    // Process request
} else {
    // Rate limited
}
```

---

## MiddlewareWithLimiter

**Location:** `ratelimit.go`

```go
func MiddlewareWithLimiter(limiter *KeyLimiter, cfg Config) func(http.Handler) http.Handler
```

Returns middleware using a provided KeyLimiter. Useful when you need access to the limiter for metrics or management.

**Example:**

```go
limiter := ratelimit.NewKeyLimiter(100, 200, time.Hour)

// Use in middleware
r.Use(ratelimit.MiddlewareWithLimiter(limiter, ratelimit.Config{
    Rate:  100,
    Burst: 200,
}))

// Monitor limiter size
metrics.Gauge("ratelimit_keys", float64(limiter.Size()))
```

---

## WAFFLE Integration

### Basic Usage

```go
func SetupRoutes(r chi.Router, db DBDeps, logger *zap.Logger) {
    // Global rate limit: 100 req/s per IP
    r.Use(ratelimit.Simple(100, 200))

    r.Get("/", homeHandler)
    r.Route("/api", func(r chi.Router) {
        r.Get("/users", listUsers)
        r.Post("/users", createUser)
    })
}
```

### Per-Route Rate Limiting

```go
func SetupRoutes(r chi.Router, db DBDeps, logger *zap.Logger) {
    r.Get("/", homeHandler)

    r.Route("/api", func(r chi.Router) {
        // Standard API rate limit
        r.Use(ratelimit.Simple(100, 200))
        r.Get("/users", listUsers)
    })

    r.Route("/auth", func(r chi.Router) {
        // Stricter limit for auth endpoints
        r.Use(ratelimit.Simple(5, 10))
        r.Post("/login", loginHandler)
        r.Post("/register", registerHandler)
    })
}
```

### Skip Health Checks

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:  100,
    Burst: 200,
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health" || r.URL.Path == "/ready"
    },
}))
```

### Custom Response

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:  100,
    Burst: 200,
    OnLimited: func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Retry-After", "1")
        w.WriteHeader(http.StatusTooManyRequests)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "rate limit exceeded",
            "retry_after": "1s",
        })
    },
}))
```

### With Logging

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:  100,
    Burst: 200,
    OnLimited: func(w http.ResponseWriter, r *http.Request) {
        logger.Warn("rate limit exceeded",
            zap.String("ip", ratelimit.IPKeyFunc(r)),
            zap.String("path", r.URL.Path),
        )
        w.Header().Set("Retry-After", "1")
        w.WriteHeader(http.StatusTooManyRequests)
        w.Write([]byte("rate limit exceeded"))
    },
}))
```

---

## Token Bucket Algorithm

The rate limiter uses the token bucket algorithm:

1. A bucket holds tokens up to the **burst** capacity
2. Tokens are added at the **rate** per second
3. Each request consumes one token
4. If no tokens are available, the request is rejected

**Example:** With `rate=10` and `burst=20`:
- Bucket starts full with 20 tokens
- Client can immediately make 20 requests (burst)
- After burst, limited to 10 requests/second
- If idle, bucket refills up to 20 tokens

---

## Rate Limiting Strategies

### Per-IP (Default)

Best for public APIs where you want to limit abuse per client.

```go
r.Use(ratelimit.Simple(100, 200))
```

### Per-Endpoint

Protect specific expensive endpoints.

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:    10,
    Burst:   20,
    KeyFunc: ratelimit.PathKeyFunc,
}))
```

### Per-User

Rate limit authenticated users by their ID.

```go
r.Use(ratelimit.Middleware(ratelimit.Config{
    Rate:  100,
    Burst: 200,
    KeyFunc: func(r *http.Request) string {
        if userID := r.Context().Value("user_id"); userID != nil {
            return userID.(string)
        }
        return ratelimit.IPKeyFunc(r)
    },
}))
```

### Tiered Limits

Different limits for different user tiers.

```go
freeLimiter := ratelimit.NewKeyLimiter(10, 20, time.Hour)
proLimiter := ratelimit.NewKeyLimiter(100, 200, time.Hour)

r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := getUserID(r)
        limiter := freeLimiter
        if isPro(r) {
            limiter = proLimiter
        }

        if !limiter.Allow(key) {
            w.WriteHeader(http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
})
```

---

## See Also

- [middleware](../middleware/middleware.md) — Other HTTP middleware
- [router](../router/router.md) — Chi router setup
