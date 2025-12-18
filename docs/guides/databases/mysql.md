# MySQL in DBDeps
*How to add a MySQL connection to your WAFFLE application's DBDeps and ConnectDB flow.*

This example shows the **minimum practical code** needed to integrate MySQL into a WAFFLE service.

It assumes you already understand the WAFFLE startup lifecycle and want a focused recipe for adding a MySQL database.

---

# File: `internal/app/bootstrap/dbdeps.go`

Your DBDeps struct should hold long-lived dependencies.
For MySQL, that's typically a `*sql.DB` connection pool.

```go
package bootstrap

import "database/sql"

// DBDeps holds connections to external backends.
// WAFFLE creates one instance of DBDeps and passes it to your handlers.
type DBDeps struct {
    MySQL *sql.DB
}
```

Nothing else happens here — DBDeps is just a struct that will be filled in by `ConnectDB`.

---

# File: `internal/app/bootstrap/hooks.go`
## Add MySQL to ConnectDB

MySQL setup belongs inside `ConnectDB`, because this is where WAFFLE expects you to initialize external systems.

### Requirements:

```bash
go get github.com/go-sql-driver/mysql
```

### Example ConnectDB implementation:

```go
import (
    "context"
    "database/sql"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "go.uber.org/zap"
    "github.com/dalemusser/waffle/config"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // DSN format: user:password@tcp(host:port)/dbname?parseTime=true
    dsn := "user:password@tcp(localhost:3306)/waffle_app?parseTime=true"

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return DBDeps{}, err
    }

    // Set connection pool settings as appropriate for your app.
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Ping to verify the connection.
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := db.PingContext(pingCtx); err != nil {
        return DBDeps{}, err
    }

    deps := DBDeps{
        MySQL: db,
    }

    logger.Info("MySQL connected")
    return deps, nil
}
```

> **Tip:** Always use `parseTime=true` in your DSN to properly scan `time.Time` values from MySQL `DATETIME` columns.

---

# Using MySQL in a Feature

Inside a feature handler, you can use `deps.MySQL` like any `*sql.DB`:

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
    err := h.deps.MySQL.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil && err != sql.ErrNoRows {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Users in DB: %d", count)))
}
```

Then register your handler in your feature's `Routes()` or `MountRoutes()` function, as usual.

---

# Clean Shutdown (Recommended)

`*sql.DB` pools should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.MySQL != nil {
        if err := deps.MySQL.Close(); err != nil {
            logger.Warn("error closing MySQL", zap.Error(err))
            return err
        }
        logger.Info("MySQL connection closed")
    }
    return nil
}
```

The key idea is: **open the DB in ConnectDB, close it in Shutdown.**

---

# Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides convenience helpers in `pantry/db/mysql`:

```go
import "github.com/dalemusser/waffle/pantry/db/mysql"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Simple connection with timeout and ping
    db, err := mysql.Connect(appCfg.MySQLDSN, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{MySQL: db}, nil
}

// Or with custom pool configuration:
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := mysql.ConnectWithConfig(appCfg.MySQLDSN, mysql.PoolConfig{
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
        ConnMaxIdleTime: 10 * time.Minute,
    }, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{MySQL: db}, nil
}

// Or use sensible defaults:
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := mysql.ConnectWithConfig(appCfg.MySQLDSN, mysql.DefaultPoolConfig(), coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{MySQL: db}, nil
}
```

---

# DSN Format

MySQL DSN (Data Source Name) format:

```
user:password@tcp(host:port)/dbname?param=value
```

Common parameters:

| Parameter | Description |
|-----------|-------------|
| `parseTime=true` | Parse `DATETIME` to `time.Time` (recommended) |
| `loc=Local` | Use local timezone for time values |
| `charset=utf8mb4` | Use UTF-8 character set with full Unicode support |
| `collation=utf8mb4_unicode_ci` | Use Unicode collation |
| `timeout=5s` | Connection timeout |
| `readTimeout=30s` | Read timeout |
| `writeTimeout=30s` | Write timeout |

Example with multiple parameters:

```go
dsn := "user:pass@tcp(localhost:3306)/mydb?parseTime=true&charset=utf8mb4&loc=Local"
```

---

# Summary

To integrate MySQL into WAFFLE:

1. Add a `*sql.DB` field to `DBDeps`
2. Implement MySQL connection logic in `ConnectDB` using `database/sql` + `go-sql-driver/mysql`
3. Use `deps.MySQL.Query*` / `Exec*` in your handlers
4. Close the DB pool during shutdown

The WAFFLE lifecycle keeps this wiring predictable, explicit, and testable.
