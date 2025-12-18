# Project Structure

*How WAFFLE itself is organized, and how to structure your applications.*

---

## WAFFLE Repository Structure

```
waffle/
  app/                    # Application lifecycle (app.Run, Hooks)
  cmd/
    makewaffle/           # CLI for scaffolding new projects
  config/                 # CoreConfig and configuration loading
  docs/                   # Documentation
  httputil/               # HTTP utility functions
  internal/               # Internal implementation details
  logging/                # Structured logging (zap-based)
  metrics/                # Prometheus metrics
  middleware/             # Core middleware (CORS, compression, etc.)
  router/                 # Chi router with WAFFLE middleware
  server/                 # HTTP/HTTPS server with graceful shutdown
  windowsservice/         # Windows SCM adapter
  pantry/
    auth/                 # Authentication (JWT, OAuth2, API keys)
    cache/                # Caching (memory, Redis)
    crypto/               # Encryption, hashing, passwords
    db/                   # Database connections (postgres, mysql, etc.)
    email/                # SMTP email sending
    errors/               # Error handling utilities
    fileserver/           # Static file serving
    httpnav/              # Navigation helpers
    i18n/                 # Internationalization
    jobs/                 # Background job processing
    mongo/                # MongoDB utilities
    mq/                   # Message queues (RabbitMQ, SQS)
    pagination/           # Keyset and offset pagination
    session/              # Session management
    templates/            # Template engine
    text/                 # Text processing
    validate/             # Input validation
    websocket/            # WebSocket support
    ...                   # Plus many more integrations
```

---

## Application Structure

A standard WAFFLE-based service follows this layout:

```
myservice/
  cmd/
    myservice/
      main.go               # Entry point: calls app.Run(hooks)

  internal/
    app/
      bootstrap/
        hooks.go            # Wires lifecycle functions into app.Hooks
        config.go           # LoadConfig and ValidateConfig
        appconfig.go        # AppConfig struct definition
        db.go               # ConnectDB and EnsureSchema
        dbdeps.go           # DBDeps struct definition
        startup.go          # Startup hook
        routes.go           # BuildHandler (HTTP routing)
        shutdown.go         # Shutdown hook

      features/             # Domain features
        users/
          handler.go
          routes.go
          models.go
        products/
          handler.go
          routes.go
          models.go

      resources/            # Shared resources (templates, static files)
      system/               # System-level handlers (health, metrics)
      store/                # Data access layer
      policy/               # Business rules and authorization

    domain/
      models/               # Domain models shared across features
```

---

## Key Directories

### `cmd/`

Application entry points. Each subdirectory is a separate binary:

```go
// cmd/myservice/main.go
func main() {
    hooks := bootstrap.NewHooks()
    app.Run(context.Background(), hooks)
}
```

### `internal/app/bootstrap/`

Lifecycle wiring. This is where WAFFLE hooks connect to your application:

| File | Purpose |
|------|---------|
| `hooks.go` | Creates and returns `app.Hooks` struct |
| `config.go` | `LoadConfig` and `ValidateConfig` implementations |
| `appconfig.go` | AppConfig struct definition |
| `db.go` | `ConnectDB` and `EnsureSchema` implementations |
| `dbdeps.go` | DBDeps struct definition |
| `routes.go` | `BuildHandler` implementation |
| `startup.go` | `Startup` hook (optional) |
| `shutdown.go` | `Shutdown` hook (optional) |

### `internal/app/features/`

Domain features, each with its own handler and routes:

```
features/
  users/
    handler.go      # UserHandler struct and methods
    routes.go       # UserRoutes(h) chi.Router
    models.go       # User-specific types
```

### `internal/domain/models/`

Shared domain models used across features.

---

## Import Paths

Services import WAFFLE with:

```go
// Core
"github.com/dalemusser/waffle/app"
"github.com/dalemusser/waffle/config"
"github.com/dalemusser/waffle/router"
"github.com/dalemusser/waffle/middleware"

// Pantry
"github.com/dalemusser/waffle/pantry/db/postgres"
"github.com/dalemusser/waffle/pantry/auth/jwt"
"github.com/dalemusser/waffle/pantry/session"
```

---

## Creating a New Project

Use the `makewaffle` CLI:

```bash
# Install the CLI (once)
go install github.com/dalemusser/waffle/cmd/makewaffle@latest

# Create a new project
makewaffle new myservice --module github.com/you/myservice

# Enter the project and download dependencies
cd myservice
go mod tidy

# Run the app
go run ./cmd/myservice
```

This generates a complete project structure with all bootstrap files, ready to customize.

### Manual Setup

If you prefer to set up manually:

1. Create new repo
2. `go mod init github.com/you/myservice`
3. `go get github.com/dalemusser/waffle`
4. Create the bootstrap files
5. Implement routing in `BuildHandler`
6. Add pantry helpers as needed
7. Run with `app.Run`

---

## See Also

- [makewaffle CLI Guide](../getting-started/makewaffle.md) — Full CLI documentation
- [First Service Tutorial](../getting-started/first-service.md) — Step-by-step guide
- [Feature Structure Examples](../patterns/features.md) — Complete feature organization

---

[← Back to Development Guide](./README.md)
