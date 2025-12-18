

# Postgres in DBDeps (pgxpool)  
*How to use pgxpool.Pool directly in your WAFFLE application's DBDeps and ConnectDB flow.*

This example is an **advanced variant** of the basic Postgres example.  
Instead of `database/sql` + `pgx`, it uses `pgxpool.Pool` directly.

Use this if you:

- Prefer pgx-native APIs  
- Want more control over Postgres behavior  
- Are already comfortable with pgx or plan to use advanced features

If you just want the simplest possible Postgres integration, see:  
**[Postgres in DBDeps (basic `*sql.DB`)](./postgres.md)**

---

# 📁 File: `internal/app/bootstrap/dbdeps.go`

Here, DBDeps holds a `*pgxpool.Pool` instead of `*sql.DB`:

```go
package bootstrap

import "github.com/jackc/pgx/v5/pgxpool"

// DBDeps holds connections to external backends.
// WAFFLE creates one instance of DBDeps and passes it to your handlers.
type DBDeps struct {
    PostgresPool *pgxpool.Pool
}
```

---

# ⚙️ File: `internal/app/bootstrap/hooks.go`  
## Add pgxpool to ConnectDB

### Requirements:

```bash
go get github.com/jackc/pgx/v5/pgxpool
```

### Example ConnectDB implementation:

```go
import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
    "github.com/dalemusser/waffle/config"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // In a real app, this would come from AppConfig or env variables.
    dsn := "postgres://user:password@localhost:5432/waffle_app?sslmode=disable"

    // Configure pool settings
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return DBDeps{}, err
    }

    // Example: tweak pool settings
    cfg.MaxConns = 10
    cfg.MinConns = 2
    cfg.MaxConnLifetime = 30 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return DBDeps{}, err
    }

    // Test the connection
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := pool.Ping(pingCtx); err != nil {
        pool.Close()
        return DBDeps{}, err
    }

    deps := DBDeps{
        PostgresPool: pool,
    }

    logger.Info("Postgres (pgxpool) connected")

    return deps, nil
}
```

---

# 💾 Using pgxpool in a Feature

Inside a feature handler, you can now use pgxpool’s native APIs:

```go
package users

import (
    "fmt"
    "net/http"

    "github.com/jackc/pgx/v5"
    "github.com/you/hello/internal/app/bootstrap"
)

type Handler struct {
    deps bootstrap.DBDeps
}

func NewHandler(deps bootstrap.DBDeps) *Handler {
    return &Handler{deps: deps}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var count int

    err := h.deps.PostgresPool.QueryRow(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil && err != pgx.ErrNoRows {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Users in DB: %d", count)))
}
```

Register this handler in your feature’s `Routes()` / `MountRoutes()` as usual.

---

# 🧹 Clean Shutdown

`pgxpool.Pool` should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.PostgresPool != nil {
        deps.PostgresPool.Close()
        logger.Info("Postgres pgxpool closed")
    }
    return nil
}
```

---

# 🔧 Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides convenience helpers in `pantry/db/postgres`:

```go
import "github.com/dalemusser/waffle/pantry/db/postgres"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Simple pool connection with timeout and ping
    pool, err := postgres.ConnectPool(appCfg.DatabaseURL, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{PostgresPool: pool}, nil
}

// Or with custom config:
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    config, err := postgres.ParseConfig(appCfg.DatabaseURL)
    if err != nil {
        return DBDeps{}, err
    }

    config.MaxConns = 20
    config.MinConns = 5
    config.MaxConnLifetime = time.Hour

    pool, err := postgres.ConnectPoolWithConfig(config, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{PostgresPool: pool}, nil
}
```

The pantry also provides `postgres.Connect()` for single connections when pooling isn't needed.

---

# 🎯 Summary

To integrate Postgres with pgxpool into WAFFLE:

1. Add a `*pgxpool.Pool` field to `DBDeps`  
2. Initialize the pool in `ConnectDB` using `pgxpool.ParseConfig` / `pgxpool.NewWithConfig`  
3. Use `pool.Query`, `QueryRow`, `Exec` in your feature handlers  
4. Close the pool cleanly during shutdown  

Use this pattern when you want pgx-native performance and features while keeping DB wiring aligned with WAFFLE’s lifecycle.

For simpler Postgres usage, stick with:  
**[Postgres in DBDeps (basic `*sql.DB`)](./postgres.md)**