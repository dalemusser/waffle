# postgres

PostgreSQL connection utilities for WAFFLE applications.

## Overview

The `postgres` package provides connection helpers for PostgreSQL using the pgx driver with timeout-bounded connections and connectivity verification.

## Import

```go
import "github.com/dalemusser/waffle/db/postgres"
```

---

## Connect

**Location:** `postgres.go`

```go
func Connect(connString string, timeout time.Duration) (*pgx.Conn, error)
```

Opens a single PostgreSQL connection. Pings to verify connectivity before returning. Use this for simple applications or scripts. For production web services, use `ConnectPool` instead.

**Example:**

```go
conn, err := postgres.Connect("postgres://localhost/mydb", 10*time.Second)
if err != nil {
    return err
}
defer conn.Close(context.Background())
```

---

## ConnectPool

**Location:** `postgres.go`

```go
func ConnectPool(connString string, timeout time.Duration) (*pgxpool.Pool, error)
```

Opens a PostgreSQL connection pool. This is the recommended approach for production applications. The pool manages connection lifecycle, handles reconnection, and provides connection reuse.

**Example:**

```go
func connectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.PostgresURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("postgres connect: %w", err)
    }

    return DBDeps{Pool: pool}, nil
}
```

---

## ConnectPoolWithConfig

**Location:** `postgres.go`

```go
func ConnectPoolWithConfig(config *pgxpool.Config, timeout time.Duration) (*pgxpool.Pool, error)
```

Opens a connection pool with custom configuration. Use when you need fine-grained control over pool settings.

**Example:**

```go
config, _ := postgres.ParseConfig(appCfg.PostgresURI)
config.MaxConns = 20
config.MinConns = 5
config.MaxConnLifetime = time.Hour

pool, err := postgres.ConnectPoolWithConfig(config, core.DBConnectTimeout)
```

---

## ParseConfig

**Location:** `postgres.go`

```go
func ParseConfig(connString string) (*pgxpool.Config, error)
```

Parses a connection string into a pool configuration for modification before connecting.

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.PostgresURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("postgres: %w", err)
    }

    logger.Info("connected to PostgreSQL")

    return DBDeps{Pool: pool}, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.Pool != nil {
        db.Pool.Close()
        logger.Info("disconnected from PostgreSQL")
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    PostgresURI string `conf:"postgres_uri"`
}
```

```bash
# Environment variables
POSTGRES_URI=postgres://user:pass@localhost:5432/mydb?sslmode=disable
```

---

## Connection String Formats

```
# URL format
postgres://user:pass@localhost:5432/dbname
postgres://user:pass@localhost:5432/dbname?sslmode=disable
postgres://user:pass@localhost:5432/dbname?pool_max_conns=10

# Keyword/value format
host=localhost port=5432 user=user password=pass dbname=dbname sslmode=disable
```

---

## Pool Configuration Options

When using `ConnectPoolWithConfig`, common settings include:

| Setting | Description | Default |
|---------|-------------|---------|
| `MaxConns` | Maximum connections in pool | 4 x CPU cores |
| `MinConns` | Minimum connections to keep open | 0 |
| `MaxConnLifetime` | Maximum time a connection can be reused | 1 hour |
| `MaxConnIdleTime` | Maximum time a connection can be idle | 30 minutes |
| `HealthCheckPeriod` | How often to check connection health | 1 minute |

---

## See Also

- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
