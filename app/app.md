# app

Application lifecycle management and the entry point for WAFFLE services.

## Overview

The `app` package provides the `Run` function and `Hooks` struct that orchestrate the entire lifecycle of a WAFFLE application — from loading configuration through graceful shutdown. This is the foundation that ties all other WAFFLE packages together.

Rather than inheriting from a base class or implementing an interface with many methods, WAFFLE uses a hooks-based approach: you provide functions for the parts you need, and `Run` calls them in the correct order with proper error handling and logging.

## Import

```go
import "github.com/dalemusser/waffle/app"
```

## Lifecycle

When you call `app.Run()`, WAFFLE executes these steps in order:

1. **Bootstrap logger** — Create a minimal logger for early startup messages
2. **Load config** — Call your `LoadConfig` hook to get core + app config
3. **Validate config** — Call your `ValidateConfig` hook (optional)
4. **Build logger** — Create the production logger based on config
5. **Register metrics** — Set up default Go/process/HTTP metrics
6. **Connect DB** — Call your `ConnectDB` hook to establish backend connections
7. **Ensure schema** — Call your `EnsureSchema` hook (optional)
8. **Wire signals** — Set up SIGINT/SIGTERM handling for graceful shutdown
9. **Startup** — Call your `Startup` hook for any final initialization (optional)
10. **Build handler** — Call your `BuildHandler` hook to create the HTTP handler
11. **Serve** — Start the HTTP server and block until shutdown signal
12. **Shutdown** — Call your `Shutdown` hook to clean up resources (optional)

If any required step fails, WAFFLE logs the error and exits. Optional hooks (marked with "optional" above) are skipped if nil.

## API

### Hooks

**Location:** `app.go`

```go
type Hooks[C any, D any] struct {
    Name           string
    LoadConfig     func(logger *zap.Logger) (*config.CoreConfig, C, error)
    ValidateConfig func(core *config.CoreConfig, appCfg C, logger *zap.Logger) error
    ConnectDB      func(ctx context.Context, core *config.CoreConfig, appCfg C, logger *zap.Logger) (D, error)
    EnsureSchema   func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error
    Startup        func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error
    BuildHandler   func(core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) (http.Handler, error)
    Shutdown       func(ctx context.Context, core *config.CoreConfig, appCfg C, db D, logger *zap.Logger) error
}
```

The `Hooks` struct uses Go generics:
- `C` — Your application's config type (e.g., `AppConfig`)
- `D` — Your application's database/dependencies type (e.g., `DBDeps`)

| Hook | Required | Purpose |
|------|----------|---------|
| `Name` | Yes | Application name for logging |
| `LoadConfig` | Yes | Load and parse configuration |
| `ValidateConfig` | No | Validate config before connecting to backends |
| `ConnectDB` | Yes | Connect to databases and external services |
| `EnsureSchema` | No | Create indexes, run migrations |
| `Startup` | No | One-time initialization after DB is ready |
| `BuildHandler` | Yes | Construct the HTTP handler (router + middleware + routes) |
| `Shutdown` | No | Clean up resources on exit |

### Run

**Location:** `app.go`

```go
func Run[C any, D any](ctx context.Context, hooks Hooks[C, D]) error
```

Executes the WAFFLE lifecycle with the provided hooks. Blocks until the server shuts down. Returns any error from the server or shutdown hook.

## Example

```go
// cmd/myapp/main.go
package main

import (
    "context"
    "github.com/dalemusser/waffle/app"
)

func main() {
    ctx := context.Background()

    if err := app.Run(ctx, app.Hooks[AppConfig, DBDeps]{
        Name:         "myapp",
        LoadConfig:   loadConfig,
        ConnectDB:    connectDB,
        BuildHandler: buildHandler,
        Shutdown:     shutdown,
    }); err != nil {
        // Error already logged by WAFFLE
        os.Exit(1)
    }
}
```

### LoadConfig Hook

```go
func loadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load()
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        SMTPHost: os.Getenv("SMTP_HOST"),
        // ... load app-specific config
    }

    return coreCfg, appCfg, nil
}
```

### ConnectDB Hook

```go
func connectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    client, err := mongo.Connect(ctx, appCfg.MongoURI)
    if err != nil {
        return DBDeps{}, fmt.Errorf("mongo connect: %w", err)
    }

    return DBDeps{
        Mongo:  client,
        DB:     client.Database(appCfg.MongoDB),
        Logger: logger,
    }, nil
}
```

### BuildHandler Hook

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(core, logger)

    // Mount features
    users.Mount(r, db)
    health.Mount(r)

    return r, nil
}
```

### Shutdown Hook

```go
func shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.Mongo != nil {
        if err := db.Mongo.Disconnect(ctx); err != nil {
            return fmt.Errorf("mongo disconnect: %w", err)
        }
    }
    return nil
}
```

## Design Philosophy

**Hooks over inheritance**: Instead of requiring you to embed a base struct or implement a large interface, WAFFLE uses a flat struct of function hooks. This makes it clear what each function does and lets you provide only what you need.

**Explicit over magic**: Each lifecycle step is visible in the `Run` function. There's no hidden initialization or implicit ordering — you can read `app.go` and understand exactly what happens.

**Fail fast**: If any required step fails, WAFFLE logs the error and exits immediately. This prevents cascading failures and makes debugging straightforward.

**Graceful shutdown**: WAFFLE handles SIGINT/SIGTERM, gives the server time to drain connections, and calls your Shutdown hook with a timeout context.

## See Also

- [config](../config/config.md) — Configuration loading
- [server](../server/server.md) — HTTP server lifecycle
- [logging](../logging/logging.md) — Logger setup
