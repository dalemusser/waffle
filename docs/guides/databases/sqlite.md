# SQLite in DBDeps
*How to add a SQLite database to your WAFFLE application's DBDeps and ConnectDB flow.*

SQLite is an embedded database that stores data in a single file. It's ideal for:

- Development and testing
- Single-instance deployments
- Applications with moderate write loads
- Edge deployments or embedded systems

This example shows the **minimum practical code** needed to integrate SQLite into a WAFFLE service.

---

# File: `internal/app/bootstrap/dbdeps.go`

Your DBDeps struct should hold long-lived dependencies.
For SQLite, that's a `*sql.DB` handle.

```go
package bootstrap

import "database/sql"

// DBDeps holds connections to external backends.
// WAFFLE creates one instance of DBDeps and passes it to your handlers.
type DBDeps struct {
    SQLite *sql.DB
}
```

---

# File: `internal/app/bootstrap/hooks.go`
## Add SQLite to ConnectDB

SQLite setup belongs inside `ConnectDB`, because this is where WAFFLE expects you to initialize external systems.

### Requirements:

```bash
go get github.com/mattn/go-sqlite3
```

> **Note:** `go-sqlite3` requires CGO. Ensure CGO is enabled in your build environment.

### Example ConnectDB implementation:

```go
import (
    "context"
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "go.uber.org/zap"
    "github.com/dalemusser/waffle/config"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Path to the database file
    dbPath := "./data/app.db"

    // Enable WAL mode and foreign keys via DSN parameters
    dsn := dbPath + "?_busy_timeout=5000&_foreign_keys=on"

    db, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return DBDeps{}, err
    }

    // SQLite performs best with limited connections due to file locking
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)

    // Ping to verify the connection
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := db.PingContext(pingCtx); err != nil {
        return DBDeps{}, err
    }

    // Enable WAL mode for better concurrency
    if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
        db.Close()
        return DBDeps{}, err
    }

    deps := DBDeps{
        SQLite: db,
    }

    logger.Info("SQLite connected", zap.String("path", dbPath))
    return deps, nil
}
```

---

# Using SQLite in a Feature

Inside a feature handler, you can use `deps.SQLite` like any `*sql.DB`:

```go
package items

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
    err := h.deps.SQLite.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM items").Scan(&count)
    if err != nil && err != sql.ErrNoRows {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Items in DB: %d", count)))
}
```

---

# Clean Shutdown (Recommended)

SQLite connections should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.SQLite != nil {
        if err := deps.SQLite.Close(); err != nil {
            logger.Warn("error closing SQLite", zap.Error(err))
            return err
        }
        logger.Info("SQLite connection closed")
    }
    return nil
}
```

---

# Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides convenience helpers in `pantry/db/sqlite` that handle WAL mode, foreign keys, and other pragmas automatically:

```go
import "github.com/dalemusser/waffle/pantry/db/sqlite"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Simple connection with sensible defaults (WAL mode, foreign keys, etc.)
    db, err := sqlite.Connect(appCfg.SQLitePath, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{SQLite: db}, nil
}

// Or with custom options:
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := sqlite.ConnectWithOptions(appCfg.SQLitePath, sqlite.Options{
        WALMode:      true,
        ForeignKeys:  true,
        BusyTimeout:  5000,  // 5 seconds
        CacheSize:    -64000, // 64MB
        Synchronous:  "NORMAL",
        MaxOpenConns: 1,
        MaxIdleConns: 1,
    }, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{SQLite: db}, nil
}

// For read-only access with multiple readers:
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := sqlite.ConnectWithOptions(appCfg.SQLitePath, sqlite.ReadOnlyOptions(), coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{SQLite: db}, nil
}

// For in-memory database (useful for testing):
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := sqlite.InMemory(coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{SQLite: db}, nil
}
```

### Available Options

| Option | Default | Description |
|--------|---------|-------------|
| `WALMode` | `true` | Enable Write-Ahead Logging for better concurrency |
| `ForeignKeys` | `true` | Enable foreign key constraint enforcement |
| `BusyTimeout` | `5000` | How long to wait when database is locked (ms) |
| `CacheSize` | `-64000` | Page cache size (negative = KiB, positive = pages) |
| `Synchronous` | `"NORMAL"` | Durability setting: OFF, NORMAL, FULL, EXTRA |
| `MaxOpenConns` | `1` | Maximum open connections |
| `MaxIdleConns` | `1` | Maximum idle connections |

### Preset Options

- `sqlite.DefaultOptions()` — Sensible defaults for web applications
- `sqlite.ReadOnlyOptions()` — Optimized for read-only access (4 readers)
- `sqlite.InMemoryOptions()` — For in-memory databases

---

# SQLite Best Practices

### WAL Mode

WAL (Write-Ahead Logging) mode provides better concurrency by allowing readers and writers to operate simultaneously. It's enabled by default in the pantry helper.

### Connection Pooling

Unlike other databases, SQLite benefits from *fewer* connections due to file-level locking. For most applications, `MaxOpenConns=1` is recommended to avoid "database is locked" errors.

### File Paths

```go
// Relative path (relative to working directory)
db, err := sqlite.Connect("./data/app.db", timeout)

// Absolute path
db, err := sqlite.Connect("/var/lib/myapp/data.db", timeout)

// In-memory (data lost on close)
db, err := sqlite.Connect(":memory:", timeout)

// Shared in-memory (multiple connections can access)
db, err := sqlite.Connect("file::memory:?cache=shared", timeout)
```

### Health Checks

The pantry provides a health check function:

```go
import "github.com/dalemusser/waffle/pantry/db/sqlite"

health.Mount(r, map[string]health.Check{
    "sqlite": sqlite.HealthCheck(deps.SQLite),
}, logger)
```

---

# Summary

To integrate SQLite into WAFFLE:

1. Add a `*sql.DB` field to `DBDeps`
2. Implement SQLite connection logic in `ConnectDB`
3. Use `deps.SQLite.Query*` / `Exec*` in your handlers
4. Close the connection during shutdown

SQLite is perfect for development, testing, and single-instance deployments. For multi-instance production deployments, consider PostgreSQL or MySQL.
