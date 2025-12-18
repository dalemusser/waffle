# Middleware Examples

*Practical patterns for adding and using middleware in WAFFLE applications.*

WAFFLE uses Chi under the hood, which makes middleware composition easy and predictable. These examples show how to apply middleware at the **application level**, **feature level**, **route group level**, and **per-route level**.

### WAFFLE Middleware Chain Diagram

```mermaid
flowchart LR
    A["Incoming Request"] --> B["App-Wide Middleware<br/>(router.New)"]
    B --> C["Feature-Level Middleware<br/>(Routes)"]
    C --> D["Sub-Group Middleware<br/>(r.Group)"]
    D --> E["Per-Route Middleware<br/>(r.With)"]
    E --> F["Handler Method"]
    F --> G["Response"]
```

See the full architectural context in [Request Flow Through WAFFLE](../../core/architecture.md#request-flow-through-waffle).

All examples follow the standard WAFFLE file structure:

```
internal/app/bootstrap/routes.go          — BuildHandler (top-level router)
internal/app/features/<feature>/routes.go — feature routers
```

---

## Built-in Middleware from router.New

When you use `router.New(coreCfg, logger)`, WAFFLE automatically applies these middlewares:

| Middleware | Purpose |
|------------|---------|
| RequestID | Unique ID for each request (for tracing) |
| RealIP | Extract client IP from X-Forwarded-For headers |
| Recoverer | Panic recovery with stack trace logging |
| Compression | Response compression (if enabled via config) |
| Body size limits | Enforce MaxRequestBodyBytes from config |
| HTTP metrics | Prometheus request duration histograms |
| Request logging | Structured access logs |
| JSON 404/405 | Consistent error responses |

You don't need to add these manually — they're included when you create the router.

---

## 1. App-Wide Middleware (BuildHandler)

This applies middleware to **every request** in the entire application, in addition to the built-in middleware.

**File:** `internal/app/bootstrap/routes.go`

```go
import (
    "github.com/dalemusser/waffle/config"
    "github.com/dalemusser/waffle/router"
    "github.com/dalemusser/waffle/middleware"
)

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    // router.New includes standard middleware (logging, metrics, recovery, etc.)
    r := router.New(coreCfg, logger)

    // Add app-wide CORS (in addition to built-in middleware)
    r.Use(middleware.CORS(coreCfg))

    // Mount features here:
    r.Mount("/about", about.Routes())

    return r, nil
}
```

**Use this when:**
You want consistent behavior (CORS, custom auth, rate limiting) across the entire app beyond what `router.New` provides.

---

## 2. Feature-Level Middleware (inside a feature router)

Apply middleware only to a specific feature's routes.

**File:** `internal/app/features/profile/routes.go`

```go
package profile

import (
    "net/http"

    "github.com/go-chi/chi/v5"
)

func Routes(h *Handler) chi.Router {
    r := chi.NewRouter()

    // Feature-specific middleware
    r.Use(h.featureLogger)

    // Public root
    r.Get("/", h.PublicProfile)

    return r
}

func (h *Handler) featureLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h.Logger.Info("profile feature request", zap.String("path", r.URL.Path))
        next.ServeHTTP(w, r)
    })
}
```

**Use this when:**
You want a feature to behave differently from the rest of the app.

---

## 3. Auth Middleware Applied to a Sub-Group

This is one of the most common real-world patterns:

- `/profile` — public
- `/profile/edit` — requires authentication

```go
func Routes(h *Handler) chi.Router {
    r := chi.NewRouter()

    r.Get("/", h.PublicProfile)

    // Protected section
    r.Group(func(pr chi.Router) {
        pr.Use(auth.RequireAuth)

        pr.Get("/edit", h.EditProfile)
        pr.Post("/edit", h.SaveProfile)
    })

    return r
}
```

**Use this for:**
Features with a mix of public and protected pages.

---

## 4. Role-Based Middleware Inside a Feature

Nested groups allow fine-grained permission control.

```go
func Routes(h *Handler) chi.Router {
    r := chi.NewRouter()

    r.Get("/", h.PublicDashboard)

    // Authenticated section
    r.Group(func(pr chi.Router) {
        pr.Use(auth.RequireAuth)

        pr.Get("/settings", h.UserSettings)

        // Admin-only area
        pr.Group(func(ar chi.Router) {
            ar.Use(auth.RequireRole("admin"))
            ar.Get("/admin", h.AdminDashboard)
            ar.Get("/admin/stats", h.AdminStats)
        })
    })

    return r
}
```

**Use this when:**
A feature contains multiple privilege levels.

---

## 5. Custom Middleware Example (Before + After Behavior)

A minimal custom middleware showing the basic pattern:

```go
func RequestTimer(logger *zap.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)
            logger.Info("request completed",
                zap.String("path", r.URL.Path),
                zap.Duration("duration", time.Since(start)),
            )
        })
    }
}
```

Apply it at any level:

```go
r.Use(RequestTimer(logger))
```

**Use this when:**
You want to measure, wrap, restrict, transform, or observe requests.

---

## 6. Middleware on a Single Route

Chi allows middleware to be applied directly to a route.

```go
r.With(rateLimitMiddleware).Get("/expensive", h.ExpensiveHandler)
```

Or:

```go
r.With(middleware.CORS(coreCfg)).Post("/webhook", h.HandleWebhook)
```

**Use this when:**
Only one or two routes need special behavior.

---

## 7. Combining Middleware Patterns

Middleware can stack:

```go
r.Group(func(pr chi.Router) {
    pr.Use(auth.RequireAuth)
    pr.Use(rateLimitMiddleware)
    pr.Use(auditLog)

    pr.Get("/secure", h.SecureHandler)
})
```

Order matters:

1. General middlewares (logging, request IDs) — provided by `router.New`
2. CORS
3. Auth / permissions
4. Rate limiting
5. Feature-specific logic

---

## Summary

This document covered practical ways to use middleware in WAFFLE:

- Built-in middleware from `router.New` (logging, metrics, recovery, compression)
- App-wide middleware in BuildHandler
- Feature-wide middleware
- Auth + role-based subgroups
- Custom middleware
- Per-route middleware
- Mixed and layered middleware

---

## See Also

- [Routing Reference](../../core/routing.md) — Chi router and WAFFLE middleware
- [Architecture Overview](../../core/architecture.md) — Request flow diagrams
- [Feature Structure Examples](./features.md) — Complete feature organization
- [Handler Patterns](./handlers.md) — Handler structure and wiring

