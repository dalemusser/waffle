# Route Examples  
*Practical patterns for defining feature routes in a WAFFLE application.*

These examples show **concrete, minimal code** for adding routes inside a feature
using WAFFLE’s recommended **subrouter + Mount** model.

All examples follow this file structure:

```
internal/app/features/<feature>/routes.go
internal/app/bootstrap/hooks.go   (BuildHandler)
```

---

# 🧭 1. Basic Route

A feature with a single route:

**File:** `internal/app/features/about/routes.go`

```go
package about

import (
    "net/http"
    "github.com/go-chi/chi/v5"
)

func Routes() chi.Router {
    r := chi.NewRouter()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("About page"))
    })

    return r
}
```

**Mount in `BuildHandler`:**

```go
r.Mount("/about", about.Routes())
```

**Result URL:** `/about`

---

# 🧩 2. Multiple Routes in a Feature

**File:** `internal/app/features/info/routes.go`

```go
package info

import (
    "net/http"
    "github.com/go-chi/chi/v5"
)

func Routes() chi.Router {
    r := chi.NewRouter()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Info Home"))
    })

    r.Get("/team", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Team page"))
    })

    r.Get("/contact", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Contact page"))
    })

    return r
}
```

**Mount:**

```go
r.Mount("/info", info.Routes())
```

**Result URLs:**

- `/info`
- `/info/team`
- `/info/contact`

---

# 🔢 3. Parameterized Routes

**File:** `internal/app/features/items/routes.go`

```go
package items

import (
    "fmt"
    "net/http"
    "github.com/go-chi/chi/v5"
)

func Routes() chi.Router {
    r := chi.NewRouter()

    r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
        id := chi.URLParam(r, "id")
        w.Write([]byte(fmt.Sprintf("Item ID: %s", id)))
    })

    return r
}
```

**Mount:**

```go
r.Mount("/items", items.Routes())
```

**Result URL:**  
`/items/123`

---

# 🧱 4. Route Groups (Feature-Level Organization)

Group related routes under a subpath:

```go
func Routes() chi.Router {
    r := chi.NewRouter()

    // Public root
    r.Get("/", PublicHandler)

    // Group under /settings/...
    r.Route("/settings", func(sr chi.Router) {
        sr.Get("/", SettingsHome)
        sr.Get("/privacy", PrivacySettings)
        sr.Get("/notifications", NotificationSettings)
    })

    return r
}
```

**Result URLs:**

- `/feature/settings`
- `/feature/settings/privacy`
- `/feature/settings/notifications`

(Assuming `r.Mount("/feature", Routes())`)

---

# 🔒 5. Mixing Public and Protected Routes

This pattern is common and useful.

**File:** `internal/app/features/profile/routes.go`

```go
func Routes() chi.Router {
    r := chi.NewRouter()

    // Public profile info
    r.Get("/", PublicProfile)

    // Authenticated-only section
    r.Group(func(pr chi.Router) {
        pr.Use(auth.RequireAuth)

        pr.Get("/edit", EditProfile)
        pr.Post("/edit", SaveProfile)
    })

    return r
}
```

**Mount:**

```go
r.Mount("/profile", profile.Routes())
```

**Result URLs:**

- `/profile` → public  
- `/profile/edit` → authenticated

---

# 🧩 6. Feature Returning Multiple Subrouters

You may want separate public/admin subsets.

**File:** `internal/app/features/dashboard/routes.go`

```go
func Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", DashboardHome)
    return r
}

func AdminRoutes() chi.Router {
    r := chi.NewRouter()
    r.Use(auth.RequireRole("admin"))
    r.Get("/", AdminDashboard)
    r.Get("/stats", AdminStats)
    return r
}
```

**Mount both:**

```go
r.Mount("/dashboard", dashboard.Routes())
r.Mount("/dashboard/admin", dashboard.AdminRoutes())
```

---

# 🌐 7. Route + Query Parameters Example

```go
r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query().Get("q")
    w.Write([]byte("Search query: " + q))
})
```

**Result:** `/feature/search?q=hello`

---

# 🧠 Summary

This document showed practical routing patterns:

- Single feature route  
- Multi-route features  
- Parameterized paths  
- Grouped routes  
- Public vs authenticated areas  
- Multiple subrouters  
- Query parameters  

These examples complement the higher-level  
**[Routes & Middleware Guide](../routes-and-middleware-guide.md)**.


