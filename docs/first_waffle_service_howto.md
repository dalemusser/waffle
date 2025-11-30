# How to Write Your First WAFFLE Service  
*A hands-on, step-by-step guide to creating your first WAFFLE-powered Go web application.*

---

# 🧇 Introduction

WAFFLE — the **Web Application Framework for Flavorful Layered Engineering** — gives you a clean, structured backbone for modern Go web applications.

In this guide, you’ll build your first WAFFLE service from scratch. No fluff, no detours — just the essential steps to get you productive fast.

---

# 🏁 1. Create a New Project

Choose a name for your app and create a new folder:

```bash
mkdir hello_waffle
cd hello_waffle
go mod init github.com/you/hello_waffle
```

---

# 🧇 2. Install WAFFLE

Pull in the WAFFLE framework:

```bash
go get github.com/dalemusser/waffle
```

---

# 🗂️ 3. Create the WAFFLE Directory Structure

WAFFLE encourages a clean separation of concerns.

Run:

```bash
mkdir -p cmd/hello
mkdir -p internal/app/bootstrap
mkdir -p internal/app/features
mkdir -p internal/app/store
mkdir -p internal/app/policy
mkdir -p internal/domain/models
```

Your structure now looks like this:

```
cmd/hello/
internal/
  app/
    bootstrap/
    features/
    store/
    policy/
  domain/
    models/
```

---

# ⚙️ 4. Define Your Application Configuration

Create:

`internal/app/bootstrap/appconfig.go`

```go
package bootstrap

type AppConfig struct {
    Greeting string
}
```

This struct holds application-specific configuration.  
WAFFLE automatically loads **core** configuration; you define **your app’s** config as needed.

---

# 🗄️ 5. Define Your Database Dependencies (Optional)

If your app doesn’t use a database yet, use a placeholder:

`internal/app/bootstrap/dbdeps.go`

```go
package bootstrap

type DBDeps struct{}
```

Later, DBDeps may include:

- *Mongo clients*
- *SQL connections*
- *Redis pools*
- *AWS clients*
- etc.

---

# 🔌 6. Implement Your WAFFLE Hooks

Create:

`internal/app/bootstrap/hooks.go`

```go
package bootstrap

import (
    "context"
    "net/http"

    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/config"
    "github.com/go-chi/chi/v5"
    "go.uber.org/zap"
)

func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load()
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        Greeting: "Hello from WAFFLE!",
    }

    return coreCfg, appCfg, nil
}

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    return DBDeps{}, nil
}

func EnsureSchema(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    return nil
}

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(appCfg.Greeting))
    })

    return r, nil
}

var Hooks = app.Hooks[AppConfig, DBDeps]{
    Name:         "hello-waffle",
    LoadConfig:   LoadConfig,
    ConnectDB:    ConnectDB,
    EnsureSchema: EnsureSchema,
    BuildHandler: BuildHandler,
}
```

This file declares:

- How config loads  
- How DB connections are created  
- How schema is prepared (optional)  
- How routes and middleware are assembled  

These hooks are the glue between your application and the WAFFLE lifecycle.

---

# 🚀 7. Create the WAFFLE Entry Point

Create:

`cmd/hello/main.go`

```go
package main

import (
    "context"
    "log"

    "github.com/dalemusser/waffle/app"
    "github.com/you/hello_waffle/internal/app/bootstrap"
)

func main() {
    if err := app.Run(context.Background(), bootstrap.Hooks); err != nil {
        log.Fatal(err)
    }
}
```

This is all your `main.go` needs.  
WAFFLE takes care of the lifecycle, logging, config, metrics, server startup, and graceful shutdown.

---

# ▶️ 8. Run Your First WAFFLE App

Start your service:

```bash
go run ./cmd/hello
```

Visit:

```
http://localhost:8080
```

You should see:

```
Hello from WAFFLE!
```

---

# 🧩 9. Add Routes (Your First Feature)

Create a feature directory:

```bash
mkdir internal/app/features/about
```

Add a handler:

`internal/app/features/about/about.go`

```go
package about

import "net/http"

func Handler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("WAFFLE apps are deliciously layered."))
}
```

Add routes:

`internal/app/features/about/routes.go`

```go
package about

import "github.com/go-chi/chi/v5"

func Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", Handler)
    return r
}
```

Mount feature routes in `BuildHandler`:

```go
r.Mount("/about", about.Routes())
```

---

# 🧰 10. Use the WAFFLE Toolkit (Optional Enhancements)

### CORS Support

```go
import "github.com/dalemusser/waffle/toolkit/cors"
r.Use(cors.Default())
```

### Windows Service Support

```go
import "github.com/dalemusser/waffle/toolkit/windowsservice"
```

Build Windows-only `main_windows.go` using the WAFFLE adapter.

---

# 🧱 11. Adding Real Functionality

You now have everything you need to build multilayered real apps:

- Add Mongo or Postgres in `ConnectDB`  
- Add schemas in `EnsureSchema`  
- Add templates or JSON APIs in features  
- Add rate limiting or middleware  
- Add authentication  
- Add scheduled tasks (triggered outside WAFFLE)  

WAFFLE is the structure. Your service is the flavor.

---

# 🎉 Congratulations!

You've built your first WAFFLE service.  
You’ve learned how to:

- Scaffold a WAFFLE project  
- Load configuration  
- Connect databases  
- Build routes  
- Use toolkit helpers  
- Run a real WAFFLE application  
- Extend with additional features  

From here, you’re ready to build complete WAFFLE-based systems such as:

- **StrataHub**  
- **StrataLog**  
- **StrataSave**  

Go build something delicious. 🧇🚀
