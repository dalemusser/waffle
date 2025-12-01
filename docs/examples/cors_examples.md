

# CORS Examples  
*How to enable and customize CORS behavior in your WAFFLE application.*

WAFFLE provides a simple CORS middleware in:

```go
import "github.com/dalemusser/waffle/toolkit/cors"
```

These examples show how to enable CORS at both the **feature level** and **application level**, and how to customize default behavior.

---

# 🧩 1. Enable CORS in BuildHandler (App‑Wide)

The most common pattern is to enable CORS for the entire WAFFLE application.

Open:

```
internal/app/bootstrap/hooks.go
```

Modify your `BuildHandler` function:

```go
import "github.com/dalemusser/waffle/toolkit/cors"

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Enable CORS globally
    r.Use(cors.Default())

    // Mount features below...
    r.Mount("/about", about.Routes())
    // r.Mount("/items", items.Routes())
    // ...

    return r, nil
}
```

This applies CORS rules to every route in your service.

---

# 🧩 2. Feature‑Level CORS

Sometimes you want **only one feature** to support cross‑origin access.

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
    "github.com/dalemusser/waffle/toolkit/cors"
)

func Routes() chi.Router {
    r := chi.NewRouter()

    // Enable CORS only for this feature
    r.Use(cors.Default())

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

# 🧩 3. Custom CORS Configuration

You can customize CORS settings instead of using `cors.Default()`.

### Allowed Origins Example

```go
cfg := cors.Config{
    AllowedOrigins: []string{
        "https://example.com",
        "https://sub.example.com",
    },
}

r.Use(cors.New(cfg))
```

### Allow All Origins (not recommended for production)

```go
r.Use(cors.New(cors.Config{
    AllowedOrigins: []string{"*"},
}))
```

### Custom Allowed Methods

```go
r.Use(cors.New(cors.Config{
    AllowedMethods: []string{"GET", "POST", "PUT"},
}))
```

---

# 🧩 4. Mixing CORS With Auth or Other Middleware

CORS works fine alongside authentication or logging middleware.

Example:

```go
r.Group(func(pr chi.Router) {
    pr.Use(cors.Default())
    pr.Use(auth.RequireAuth)

    pr.Get("/secure", secureHandler)
})
```

Order matters:

- **CORS should generally come first**
- Then auth / other middlewares

---

# 🧠 Summary

CORS can be applied in WAFFLE at multiple levels:

- **App‑wide** (in BuildHandler)
- **Feature‑level** (inside a feature’s local router)
- **Custom rules** (configure allowed origins, methods, headers)
- **Combined with auth** (just ensure CORS comes before auth)

Use CORS selectively to expose only what your application intends to make public.

If you'd like, we can also create:
- Preflight debugging examples  
- CORS + JSON API patterns  
- CORS setups for frontend SPAs served from another domain  