# sqlite

SQLite connection utilities for WAFFLE applications.

## Overview

The `sqlite` package provides connection helpers for SQLite with sensible defaults for web applications: WAL mode, foreign keys, busy timeout, and page cache configuration.

## Import

```go
import "github.com/dalemusser/waffle/db/sqlite"
```

---

## Connect

**Location:** `sqlite.go`

```go
func Connect(path string, timeout time.Duration) (*sql.DB, error)
```

Opens a SQLite database with sensible defaults for web applications: WAL mode enabled, foreign keys enforced, 5-second busy timeout, and 64MB cache. Returns a `*sql.DB` compatible with Go's standard database/sql interface.

**Path formats:**

```
./data.db                         # Relative path
/var/lib/myapp/data.db            # Absolute path
:memory:                          # In-memory (single connection)
file::memory:?cache=shared        # Shared in-memory
```

**Example:**

```go
db, err := sqlite.Connect("./data.db", 10*time.Second)
if err != nil {
    return err
}
defer db.Close()
```

---

## ConnectWithOptions

**Location:** `sqlite.go`

```go
func ConnectWithOptions(path string, opts Options, timeout time.Duration) (*sql.DB, error)
```

Opens a SQLite database with custom options.

**Example:**

```go
opts := sqlite.DefaultOptions()
opts.CacheSize = -128000  // 128MB
opts.BusyTimeout = 10000  // 10 seconds

db, err := sqlite.ConnectWithOptions("./data.db", opts, 10*time.Second)
```

---

## Options

**Location:** `sqlite.go`

```go
type Options struct {
    WALMode         bool          // Enable Write-Ahead Logging (default: true)
    ForeignKeys     bool          // Enable foreign key constraints (default: true)
    BusyTimeout     int           // Lock wait timeout in ms (default: 5000)
    CacheSize       int           // Page cache in KiB (negative) or pages (default: -64000)
    Synchronous     string        // "OFF", "NORMAL", "FULL", "EXTRA" (default: "NORMAL")
    JournalMode     string        // Override WAL mode if set
    MaxOpenConns    int           // Max connections (default: 1)
    MaxIdleConns    int           // Idle connections (default: 1)
    ConnMaxLifetime time.Duration // Connection lifetime (default: 0)
}
```

---

## DefaultOptions

**Location:** `sqlite.go`

```go
func DefaultOptions() Options
```

Returns sensible defaults for web applications:

| Setting | Default | Description |
|---------|---------|-------------|
| `WALMode` | true | Better concurrent read performance |
| `ForeignKeys` | true | Enforce referential integrity |
| `BusyTimeout` | 5000 | Wait 5s when database is locked |
| `CacheSize` | -64000 | 64MB page cache |
| `Synchronous` | "NORMAL" | Good balance for WAL mode |
| `MaxOpenConns` | 1 | Avoids lock contention |

---

## ReadOnlyOptions

**Location:** `sqlite.go`

```go
func ReadOnlyOptions() Options
```

Returns options optimized for read-only access with multiple concurrent readers.

---

## InMemoryOptions

**Location:** `sqlite.go`

```go
func InMemoryOptions() Options
```

Returns options for in-memory databases with no durability requirements.

---

## InMemory

**Location:** `sqlite.go`

```go
func InMemory(timeout time.Duration) (*sql.DB, error)
```

Opens a shared in-memory SQLite database. Multiple connections can access the same data. Data is lost when all connections close.

**Example:**

```go
db, err := sqlite.InMemory(5*time.Second)
if err != nil {
    return err
}
defer db.Close()
```

---

## HealthCheck

**Location:** `sqlite.go`

```go
func HealthCheck(db *sql.DB) func(ctx context.Context) error
```

Returns a health check function compatible with the health package.

**Example:**

```go
health.Mount(r, map[string]health.Check{
    "sqlite": sqlite.HealthCheck(db),
}, logger)
```

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := sqlite.Connect(appCfg.SQLitePath, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("sqlite: %w", err)
    }

    logger.Info("connected to SQLite", zap.String("path", appCfg.SQLitePath))

    return DBDeps{DB: db}, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.DB != nil {
        if err := db.DB.Close(); err != nil {
            return fmt.Errorf("sqlite close: %w", err)
        }
        logger.Info("closed SQLite database")
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    SQLitePath string `conf:"sqlite_path"`
}
```

```bash
# Environment variables
SQLITE_PATH=./data/app.db
```

---

## When to Use SQLite

SQLite is excellent for:
- **Development and testing** — Zero configuration, no external services
- **Single-server deployments** — No network latency, very fast
- **Embedded applications** — Self-contained, no dependencies
- **Read-heavy workloads** — WAL mode handles concurrent reads well
- **Small to medium datasets** — Up to ~1TB comfortably

Consider PostgreSQL or MySQL when you need:
- Multiple application servers writing simultaneously
- Complex replication or clustering
- Very high write throughput

---

## WAL Mode

Write-Ahead Logging (WAL) mode is enabled by default and provides:
- **Concurrent readers** — Readers don't block writers, writers don't block readers
- **Better performance** — Writes are sequential, reads can continue during writes
- **Crash safety** — Database remains consistent after unexpected shutdown

Creates two additional files: `data.db-wal` and `data.db-shm`

---

## See Also

- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
