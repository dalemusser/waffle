

# WAFFLE Quickstart Guide  
*Web Application Framework for Flavorful Layered Engineering*  
*Build deliciously structured Go web services—fast.*

---

# 🍽️ Introduction

WAFFLE is a Go-based framework that gives you a clean, layered, extensible foundation for building production‑grade web applications. It handles the boilerplate—lifecycle, configuration, logging, metrics, health, graceful shutdown—while you focus on your app’s features.

This guide walks you through creating your **first WAFFLE-powered service** from scratch.

---

# 🧇 1. Install WAFFLE

In your terminal:

```bash
go get github.com/dalemusser/waffle
```

You now have all WAFFLE core and toolkit modules available for import.

---

# 🏗️ 2. Create a New WAFFLE Service

Create a directory for your new service:

```bash
mkdir myservice
cd myservice
go mod init github.com/you/myservice
```

---

# 📦 3. Create the WAFFLE App Skeleton

Your project should use this structure:

```
cmd/myservice/main.go
internal/
  app/
    bootstrap/
      appconfig.go
      dbdeps.go
      hooks.go
    features/
    policy/
    store/
  domain/
    models/
```

You can create the folders now:

```bash
mkdir -p cmd/myservice
mkdir -p internal/app/bootstrap
mkdir -p internal/app/features
mkdir -p internal/app/policy
mkdir -p internal/app/store
mkdir -p internal/domain/models
```

---

# ⚙️ 4. Define Your App Config

Create: `internal/app/bootstrap/appconfig.go`

```go
package bootstrap

type AppConfig struct {
    Message string
}
```

---

# 🗄️ 5. Define Your DB Dependencies (Optional)

If your app doesn't use a database yet, create a minimal placeholder:

`internal/app/bootstrap/dbdeps.go`

```go
package bootstrap

type DBDeps struct{}
```

---

# 🔌 6. Implement WAFFLE Hooks

Create: `internal/app/bootstrap/hooks.go`

```go
package bootstrap

import (
    "context"
    "net/http"

    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/config"
    "go.uber.org/zap"
    "github.com/go-chi/chi/v5"
)

func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load()
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        Message: "Hello from WAFFLE!",
    }

    return coreCfg, appCfg, nil
}

func ConnectDB(ctx context.Context, core *config.CoreConfig, cfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    return DBDeps{}, nil
}

func EnsureSchema(ctx context.Context, core *config.CoreConfig, cfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    return nil
}

func BuildHandler(core *config.CoreConfig, cfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(cfg.Message))
    })

    return r, nil
}

var Hooks = app.Hooks[AppConfig, DBDeps]{
    Name:         "myservice",
    LoadConfig:   LoadConfig,
    ConnectDB:    ConnectDB,
    EnsureSchema: EnsureSchema,
    BuildHandler: BuildHandler,
}
```

---

# 🚀 7. Create Your WAFFLE Entry Point

Create `cmd/myservice/main.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/dalemusser/waffle/app"
    "github.com/you/myservice/internal/app/bootstrap"
)

func main() {
    if err := app.Run(context.Background(), bootstrap.Hooks); err != nil {
        log.Fatal(err)
    }
}
```

---

# ▶️ 8. Run Your WAFFLE App

```bash
go run ./cmd/myservice
```

Visit:

```
http://localhost:8080
```

You should see:

```
Hello from WAFFLE!
```

Congratulations — you’ve just built your first WAFFLE service.

---

# 🍯 9. Add Optional Toolkit Helpers

### Enable CORS

```go
import "github.com/dalemusser/waffle/toolkit/cors"

r.Use(cors.Default())
```

### Run as a Windows Service

```go
import "github.com/dalemusser/waffle/toolkit/windowsservice"
```

---

# 🧱 10. Add Real Features

Inside `internal/app/features`, create route groups such as:

```
users/
  routes.go
  handler.go
games/
  routes.go
  handler.go
```

And mount them in `BuildHandler`.

---

# 💡 11. WAFFLE Best Practices

- Keep domain logic in `internal/domain/models`.
- Keep persistence logic in `internal/app/store`.
- Use WAFFLE middleware + config patterns consistently.
- Put reusable helpers into WAFFLE Toolkit—not your app.
- Keep your app small and flavorful. No soggy abstractions.

---

# 🧇 12. What to Build Next

Once the basics work, you can explore:

- Health checks via `waffle/health`
- Production HTTPS via WAFFLE server config
- Mongo, Postgres, or Redis DB layers
- Authentication middleware
- Frontend templates
- Metrics dashboards
- Logging integrations via zap

---

# 🎉 You Are Now a WAFFLE Chef

You now know how to:

- Scaffold a WAFFLE app  
- Define configs & dependencies  
- Build routes  
- Run and extend a service  
- Use toolkit helpers  

From here, you can build full Strata-powered applications like:

- **StrataHub**
- **StrataLog**
- **StrataSave**

Just add your features—and WAFFLE handles the rest.

Bon appé‑tech! 🧇🚀