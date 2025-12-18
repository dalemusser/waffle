# CORS Examples
*How to enable and customize CORS behavior in your WAFFLE application.*

WAFFLE provides CORS middleware in the `middleware` package:

```go
import "github.com/dalemusser/waffle/middleware"
```

These examples show how to enable CORS at both the **feature level** and **application level**, and how to customize behavior.

---

## Available CORS Functions

| Function | Use Case |
|----------|----------|
| `CORSPermissive()` | Development/internal APIs — allows all origins |
| `CORS(opts)` | Custom settings via function parameters |
| `CORSFromConfig(coreCfg)` | Settings from config.toml |

---

# 1. Enable CORS in BuildHandler (App-Wide)

The most common pattern is to enable CORS for the entire WAFFLE application.

Open:

```
internal/app/bootstrap/routes.go
```

Modify your `BuildHandler` function:

```go
import (
    "github.com/dalemusser/waffle/middleware"
    "github.com/dalemusser/waffle/router"
)

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Option 1: Permissive CORS for development
    r.Use(middleware.CORSPermissive())

    // Option 2: CORS from config
    r.Use(middleware.CORSFromConfig(coreCfg))

    // Mount features below...
    r.Mount("/about", about.Routes())

    return r, nil
}
```

This applies CORS rules to every route in your service.

---

# 2. Feature-Level CORS

Sometimes you want **only one feature** to support cross-origin access.

Example: `/public-api` allows CORS but the rest of the app does not.

Feature file:

```
internal/app/features/publicapi/routes.go
```

```go
package publicapi

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/dalemusser/waffle/middleware"
)

func Routes() chi.Router {
    r := chi.NewRouter()

    // Enable CORS only for this feature
    r.Use(middleware.CORSPermissive())

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Public API"))
    })

    return r
}
```

Mount in BuildHandler:

```go
r.Mount("/public-api", publicapi.Routes())
```

This pattern is useful when exposing a dedicated public API endpoint.

---

# 3. Custom CORS Configuration

Use `middleware.CORS()` with `CORSOptions` for full control.

### Specific Allowed Origins

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins: []string{
        "https://example.com",
        "https://sub.example.com",
    },
}))
```

### Allow All Origins (development only)

```go
r.Use(middleware.CORSPermissive())
```

Or explicitly:

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins: []string{"*"},
}))
```

### Custom Methods and Headers

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins:   []string{"https://app.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Custom-Header"},
    ExposedHeaders:   []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           600, // 10 minutes
}))
```

### With Credentials

When `AllowCredentials` is true, you cannot use `"*"` for origins:

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins:   []string{"https://app.example.com"}, // Must be specific
    AllowCredentials: true,
}))
```

---

# 4. CORS From Configuration

For production deployments, define CORS in `config.toml`:

```toml
[cors]
enable_cors = true
cors_allowed_origins = ["https://app.example.com", "https://staging.example.com"]
cors_allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
cors_allowed_headers = ["Accept", "Authorization", "Content-Type"]
cors_exposed_headers = ["X-Request-ID"]
cors_allow_credentials = true
cors_max_age = 600
```

Then use:

```go
r.Use(middleware.CORSFromConfig(coreCfg))
```

If `enable_cors` is false, this returns a no-op middleware.

---

# 5. Mixing CORS With Auth or Other Middleware

CORS works alongside authentication or logging middleware.

Example:

```go
r.Group(func(pr chi.Router) {
    pr.Use(middleware.CORSPermissive())
    pr.Use(auth.RequireAuth)

    pr.Get("/secure", secureHandler)
})
```

**Order matters:**

- **CORS should generally come first**
- Then auth / other middlewares

This ensures preflight OPTIONS requests are handled before auth checks.

---

# 6. CORSOptions Reference

```go
type CORSOptions struct {
    // AllowedOrigins is a list of origins a cross-domain request can be executed from.
    // Use ["*"] to allow any origin, or specify exact origins.
    AllowedOrigins []string

    // AllowedMethods is a list of methods the client is allowed to use.
    // Default: ["GET", "POST", "OPTIONS"]
    AllowedMethods []string

    // AllowedHeaders is a list of headers the client is allowed to use.
    // Default: ["Accept", "Authorization", "Content-Type"]
    AllowedHeaders []string

    // ExposedHeaders is a list of headers that are safe to expose to the client.
    ExposedHeaders []string

    // AllowCredentials indicates whether the request can include credentials.
    // Cannot be true when AllowedOrigins is ["*"].
    AllowCredentials bool

    // MaxAge indicates how long (in seconds) preflight results can be cached.
    // Default: 300 (5 minutes). Maximum: 604800 (1 week).
    MaxAge int
}
```

### Validation

The `CORS()` function validates options at construction time and **panics** if:

- `AllowCredentials` is true and `AllowedOrigins` contains `"*"` (browsers reject this)
- `MaxAge` exceeds 604800 seconds (1 week)

This ensures misconfigurations are caught at startup, not at runtime.

### Security Considerations

When `AllowCredentials` is true, browsers will include cookies and authentication headers in cross-origin requests. This has significant security implications:

- **Never use credentials with wildcard origins** — Browsers reject this, and WAFFLE enforces it
- **Only allow origins you trust** — Each allowed origin can make authenticated requests on behalf of your users
- **Consider SameSite cookies** — Use alongside CORS for defense in depth against CSRF
- **Prefer no credentials** — If you don't need cookies/auth, keep `AllowCredentials` false to reduce attack surface

---

# Summary

| Scenario | Recommended Approach |
|----------|---------------------|
| Development/testing | `CORSPermissive()` |
| Production with config | `CORSFromConfig(coreCfg)` |
| Specific origins in code | `CORS(CORSOptions{...})` |
| Single feature needs CORS | Apply middleware to that feature's router |

Use CORS selectively to expose only what your application intends to make public.
