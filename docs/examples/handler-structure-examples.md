# Handler Structure Examples  
*How to build real-world handlers that have DBDeps, config, and logging in WAFFLE.*

In smaller examples, we often show just one dependency (like `DBDeps`) on a handler.  
In real applications, your handlers usually need **multiple things**:

- database connections (DBDeps)
- app-specific config
- core framework config
- logging
- possibly other shared utilities

This document shows how to define a **composite handler** type for a feature,  
and how `BuildHandler` wires it together.

---

## 🧠 Roles: Who defines what?

To clear up a subtle but important distinction:

- A **feature package** (e.g., `internal/app/features/items`) **defines**:
  - the `Handler` struct for that feature
  - the `NewHandler(...)` constructor
  - the `Routes(h *Handler) chi.Router` function that binds URLs to handler methods

- The **bootstrap package** (`internal/app/bootstrap/hooks.go`) **constructs**:
  - concrete instances of those handlers
  - and passes them into `Routes(...)` inside `BuildHandler`

In other words:

- Feature defines the **shape** of what it needs (`Handler` + `NewHandler` + `Routes`)
- `BuildHandler` in bootstrap decides **when and how** to build it and mount it

---

## 🧱 1. Composite Handler Struct in a Feature

**File:** `internal/app/features/items/handler.go`

```go
package items

import (
    "github.com/dalemusser/waffle/config"
    "go.uber.org/zap"
    "github.com/you/hello/internal/app/bootstrap"
)

// Handler contains everything the items feature needs.
type Handler struct {
    DB      bootstrap.DBDeps
    Logger  *zap.Logger
    CoreCfg *config.CoreConfig
    AppCfg  bootstrap.AppConfig
}
```

Here, `Handler` is a **feature-owned type**.  
The feature decides which fields it needs (DB, logger, configs, etc.).

---

## 🧩 2. Handler Constructor: NewHandler

Still in `internal/app/features/items/handler.go`:

```go
func NewHandler(coreCfg *config.CoreConfig, appCfg bootstrap.AppConfig, deps bootstrap.DBDeps, logger *zap.Logger) *Handler {
    return &Handler{
        DB:      deps,
        Logger:  logger,
        CoreCfg: coreCfg,
        AppCfg:  appCfg,
    }
}
```

This is where the feature says:

> “If you want to construct my handler, these are the pieces I need.”

The feature **defines** the constructor, but does **not** decide when it’s called.

---

## 🔌 3. Handler Methods Using All Dependencies

Example methods on the `Handler` type:

```go
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
    collection := h.DB.MongoDB.Collection("items")
    h.Logger.Info("listing items", zap.String("route", "/items"))

    // ... query the DB, render JSON, etc. ...
}

func (h *Handler) CountItems(w http.ResponseWriter, r *http.Request) {
    var count int
    err := h.DB.Postgres.QueryRowContext(r.Context(),
        "SELECT COUNT(*) FROM items",
    ).Scan(&count)
    if err != nil {
        http.Error(w, "db error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Total items: %d", count)))
}
```

Handlers now have **direct access** to:

- `h.DB` (Mongo, Postgres, Redis, etc.)
- `h.Logger` (Zap logger)
- `h.CoreCfg` (core framework config)
- `h.AppCfg` (app-specific config)

No need to “go through Env” unless you prefer that style.

---

## 🧭 4. Routes Function That Accepts *Handler

**File:** `internal/app/features/items/routes.go`

```go
package items

import (
    "github.com/go-chi/chi/v5"
)

func Routes(h *Handler) chi.Router {
    r := chi.NewRouter()

    r.Get("/", h.ListItems)
    r.Get("/count", h.CountItems)

    return r
}
```

Note the difference from earlier minimal examples:

- Earlier: `func Routes() chi.Router` — handler appears “from nowhere”
- Here: `func Routes(h *Handler) chi.Router` — handler is **explicitly passed in**

The feature **defines** this signature and uses the passed-in handler.

---

## 🧩 5. BuildHandler Wiring: Where the Handler Is Constructed

Now we wire everything together in the bootstrap package.

**File:** `internal/app/bootstrap/hooks.go`

```go
package bootstrap

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/dalemusser/waffle/config"
    "go.uber.org/zap"

    "github.com/you/hello/internal/app/features/about"
    "github.com/you/hello/internal/app/features/items"
)

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Construct composite handler for the items feature
    itemsHandler := items.NewHandler(coreCfg, appCfg, deps, logger)

    // Mount items routes, passing the handler
    r.Mount("/items", items.Routes(itemsHandler))

    // A feature that doesn't need DBDeps can still use simple Routes()
    r.Mount("/about", about.Routes())

    return r, nil
}
```

Key points:

- `BuildHandler` lives in **bootstrap** (`internal/app/bootstrap/hooks.go`)
- It is created by **makewaffle** and **meant to be edited**
- This is where you:
  - call `NewHandler(...)`
  - pass the resulting handler into `Routes(h *Handler)`
  - decide which URL prefixes features live under

---

## 🔄 6. Variations: Passing Env vs Individual Fields

If you prefer a slightly different shape, you can wrap multiple things in an Env struct:

```go
type Env struct {
    CoreCfg *config.CoreConfig
    AppCfg  AppConfig
    DB      DBDeps
    Logger  *zap.Logger
}
```

Then your handler might hold `Env` instead of separate fields:

```go
type Handler struct {
    Env Env
}
```

And:

```go
func NewHandler(env Env) *Handler {
    return &Handler{Env: env}
}
```

From the feature’s perspective, both patterns are fine:

- `Handler` with multiple fields: `h.DB`, `h.Logger`, `h.CoreCfg`, `h.AppCfg`
- `Handler` with a single `Env` field: `h.Env.DB`, `h.Env.Logger`, etc.

Choose the style that best matches how you think about your app.

---

## 🧠 Summary

This document ties together the missing “whole picture”:

- **Features define**:
  - The `Handler` struct (what they need)
  - `NewHandler(...)` (how to construct it)
  - `Routes(h *Handler)` (how to bind URLs to methods)

- **Bootstrap’s BuildHandler (in `internal/app/bootstrap/hooks.go`) constructs**:
  - Concrete handler instances per feature
  - Mounts feature routers with those handlers

This gives you the same comfort you had in StrataHub:

> “In a feature, I can assume my handler has what I need: DB, config, logger, etc.”

From here, you can:

- Reuse this pattern across all features
- Expand `Handler` as your app’s needs grow
- Keep the wiring centralized and explicit in `BuildHandler`
