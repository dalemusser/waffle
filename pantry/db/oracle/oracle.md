# oracle

Oracle Database integration guide for WAFFLE applications.

## Overview

WAFFLE does not include a built-in Oracle package due to the external dependencies required (Oracle Instant Client, CGO compilation). However, Oracle Database integrates cleanly with WAFFLE using the [godror](https://github.com/godror/godror) driver and Go's standard `database/sql` interface.

## Prerequisites

1. **Oracle Instant Client** — Download from [Oracle](https://www.oracle.com/database/technologies/instant-client.html) and add to your library path
2. **CGO enabled** — `CGO_ENABLED=1` (default on most systems)
3. **godror driver** — `go get github.com/godror/godror`

## Basic Connection

```go
import (
    "context"
    "database/sql"
    "time"

    _ "github.com/godror/godror"
)

func connectOracle(dsn string, timeout time.Duration) (*sql.DB, error) {
    db, err := sql.Open("godror", dsn)
    if err != nil {
        return nil, err
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, err
    }

    return db, nil
}
```

## DSN Formats

```
# Simple format
user/password@host:port/service_name

# With options
user/password@(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=host)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=service)))

# Easy Connect Plus (Oracle 19c+)
user/password@tcps://host:port/service_name?wallet_location=/path/to/wallet
```

## Connection Pool Settings

```go
func connectOracleWithPool(dsn string, timeout time.Duration) (*sql.DB, error) {
    db, err := sql.Open("godror", dsn)
    if err != nil {
        return nil, err
    }

    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(5 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, err
    }

    return db, nil
}
```

## WAFFLE Integration

```go
// internal/app/bootstrap/db.go
import (
    "context"
    "database/sql"
    "fmt"

    _ "github.com/godror/godror"
    "github.com/dalemusser/waffle/config"
    "go.uber.org/zap"
)

type DBDeps struct {
    DB *sql.DB
}

func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := sql.Open("godror", appCfg.OracleDSN)
    if err != nil {
        return DBDeps{}, fmt.Errorf("oracle open: %w", err)
    }

    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Verify connection with timeout
    pingCtx, cancel := context.WithTimeout(ctx, core.DBConnectTimeout)
    defer cancel()

    if err := db.PingContext(pingCtx); err != nil {
        db.Close()
        return DBDeps{}, fmt.Errorf("oracle ping: %w", err)
    }

    logger.Info("connected to Oracle")

    return DBDeps{DB: db}, nil
}

func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.DB != nil {
        if err := db.DB.Close(); err != nil {
            return fmt.Errorf("oracle close: %w", err)
        }
        logger.Info("disconnected from Oracle")
    }
    return nil
}
```

## Health Check

```go
func OracleHealthCheck(db *sql.DB) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        return db.PingContext(ctx)
    }
}

// Usage with health package
health.Mount(r, map[string]health.Check{
    "oracle": OracleHealthCheck(db),
}, logger)
```

## Configuration

```go
type AppConfig struct {
    OracleDSN string `conf:"oracle_dsn"`
}
```

```bash
# Environment variables
ORACLE_DSN=user/password@localhost:1521/ORCL
```

## Build Considerations

Oracle requires CGO and the Oracle Instant Client. Your build environment needs:

```bash
# macOS (Intel)
export DYLD_LIBRARY_PATH=/path/to/instantclient_19_8

# macOS (Apple Silicon) - requires x86_64 client via Rosetta or cross-compile
export DYLD_LIBRARY_PATH=/path/to/instantclient_19_8

# Linux
export LD_LIBRARY_PATH=/opt/oracle/instantclient_19_8

# Build
CGO_ENABLED=1 go build ./...
```

For containerized deployments, include Oracle Instant Client in your Docker image:

```dockerfile
FROM oraclelinux:8-slim

RUN dnf install -y oracle-instantclient-release-el8 && \
    dnf install -y oracle-instantclient-basic oracle-instantclient-devel

COPY myapp /app/myapp
CMD ["/app/myapp"]
```

## Why No Built-in Package?

WAFFLE's other database packages (PostgreSQL, MySQL, SQLite, Redis) use pure-Go drivers that compile anywhere without external dependencies. Oracle's godror driver requires:

1. **Oracle Instant Client** — A C library that must be installed separately
2. **CGO** — Cross-compilation becomes complex
3. **Licensing** — Oracle Instant Client has its own license terms

By not including a `db/oracle` package, WAFFLE remains lightweight and easy to build. The pattern shown above integrates Oracle seamlessly using standard Go database/sql patterns.

## See Also

- [db](../db.md) — Database package index
- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
