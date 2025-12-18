# Postgres in DBDeps  
*How to add a PostgreSQL connection to your WAFFLE application's DBDeps and ConnectDB flow.*

This example shows the **minimum practical code** needed to integrate Postgres into a WAFFLE service.

It assumes you already understand the WAFFLE startup lifecycle and want a focused recipe for adding a Postgres database.

---

# 📁 File: `internal/app/bootstrap/dbdeps.go`

Your DBDeps struct should hold long-lived dependencies.  
For Postgres, that’s usually a `*sql.DB` connection pool.

```go
package bootstrap

import "database/sql"

// DBDeps holds connections to external backends.
// WAFFLE creates one instance of DBDeps and passes it to your handlers.
type DBDeps struct {
    Postgres *sql.DB
}
```

Nothing else happens here — DBDeps is just a struct that will be filled in by `ConnectDB`.

---

# ⚙️ File: `internal/app/bootstrap/hooks.go`  
## Add Postgres to ConnectDB

Postgres setup belongs inside `ConnectDB`, because this is where WAFFLE expects you to initialize external systems.

We’ll use Go’s `database/sql` package with the `pgx` driver.

### Requirements:

```bash
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/stdlib
```

### Example ConnectDB implementation:

```go
import (
    "context"
    "database/sql"
    "time"

    "github.com/jackc/pgx/v5/stdlib"
    "go.uber.org/zap"
    "github.com/dalemusser/waffle/config"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // You can place your connection string in AppConfig or env vars.
    dsn := "postgres://user:password@localhost:5432/waffle_app?sslmode=disable"

    // Register the pgx driver if needed (often done in init)
    sql.Register("pgx", &stdlib.Driver{})

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return DBDeps{}, err
    }

    // Set connection pool settings as appropriate for your app.
    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(30 * time.Minute)

    // Ping to verify the connection.
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := db.PingContext(pingCtx); err != nil {
        return DBDeps{}, err
    }

    deps := DBDeps{
        Postgres: db,
    }

    logger.Info("Postgres connected", zap.String("dsn", dsn))
    return deps, nil
}
```

> 💡 **Tip:** In a real app, avoid putting credentials directly in source code.  
> Use environment variables or a configuration system (e.g., store DSN pieces in `AppConfig`).

---

# 💾 Using Postgres in a Feature

Inside a feature handler, you can use `deps.Postgres` like any `*sql.DB`:

```go
package users

import (
    "database/sql"
    "fmt"
    "net/http"

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
    err := h.deps.Postgres.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil && err != sql.ErrNoRows {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Users in DB: %d", count)))
}
```

Then register your handler in your feature’s `Routes()` or `MountRoutes()` function, as usual.

---

# 🧹 Clean Shutdown (Recommended)

`*sql.DB` pools should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.Postgres != nil {
        if err := deps.Postgres.Close(); err != nil {
            logger.Warn("error closing Postgres", zap.Error(err))
            return err
        }
        logger.Info("Postgres connection closed")
    }
    return nil
}
```

The key idea is: **open the DB in ConnectDB, close it in Shutdown.**

---

# 🔧 Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides convenience helpers in `pantry/db/postgres`:

```go
import "github.com/dalemusser/waffle/pantry/db/postgres"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // For a single connection (simple use cases)
    conn, err := postgres.Connect(appCfg.DatabaseURL, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{Postgres: conn}, nil
}

// For production with connection pooling (recommended):
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.DatabaseURL, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{PostgresPool: pool}, nil
}
```

See [Postgres with pgxpool](./postgres-pgxpool.md) for more details on pool configuration.

---

# 🎯 Summary

To integrate Postgres into WAFFLE:

1. Add a `*sql.DB` field to `DBDeps`  
2. Implement Postgres connection logic in `ConnectDB` using `database/sql` + `pgx`  
3. Use `deps.Postgres.Query*` / `Exec*` in your handlers  
4. Close the DB pool during shutdown  

The WAFFLE lifecycle keeps this wiring predictable, explicit, and testable.

For an example using `pgxpool.Pool` directly, see  
[Postgres DBDeps with pgxpool](./postgres-pgxpool.md).
