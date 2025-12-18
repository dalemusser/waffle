# DBDeps Examples

This directory contains examples for configuring database dependencies (DBDeps) in WAFFLE applications.

---

## Database Drivers

| Database | Description | File |
|----------|-------------|------|
| [PostgreSQL](./postgres.md) | Basic `*sql.DB` with pgx driver | `postgres.md` |
| [PostgreSQL (pgxpool)](./postgres-pgxpool.md) | Connection pooling with pgx | `postgres-pgxpool.md` |
| [MongoDB](./mongo.md) | MongoDB with official Go driver | `mongo.md` |
| [MySQL](./mysql.md) | MySQL with go-sql-driver | `mysql.md` |
| [SQLite](./sqlite.md) | SQLite with go-sqlite3 | `sqlite.md` |
| [Redis](./redis.md) | Redis with go-redis | `redis.md` |

## Usage Patterns

| Document | Description |
|----------|-------------|
| [Usage Examples](./usage-examples.md) | Common patterns for using DBDeps in handlers |

---

## Overview

DBDeps (Database Dependencies) is WAFFLE's pattern for managing database connections. It provides:

- **Centralized connection management** — All database connections in one struct
- **Lifecycle management** — Connections created in `ConnectDB`, cleaned up in `Shutdown`
- **Dependency injection** — DBDeps passed to handlers via `BuildHandler`

### Basic Pattern

```go
// internal/app/bootstrap/dbdeps.go
type DBDeps struct {
    DB    *sql.DB
    Redis *redis.Client
}

// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Initialize connections
    db, err := sql.Open("postgres", appCfg.DatabaseURL)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{DB: db}, nil
}

// internal/app/bootstrap/shutdown.go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.DB != nil {
        return deps.DB.Close()
    }
    return nil
}
```

### WAFFLE Pantry Helpers

WAFFLE provides convenience helpers in `pantry/db/` for common databases:

```go
import (
    "github.com/dalemusser/waffle/pantry/db/postgres"
    "github.com/dalemusser/waffle/pantry/db/mongo"
    "github.com/dalemusser/waffle/pantry/db/mysql"
    "github.com/dalemusser/waffle/pantry/db/sqlite"
    "github.com/dalemusser/waffle/pantry/db/redis"
)

// Each provides a Connect() function with timeout and ping
pool, err := postgres.ConnectPool(connString, 10*time.Second)
client, err := mongo.Connect(uri, 10*time.Second)
db, err := mysql.Connect(dsn, 10*time.Second)
db, err := sqlite.Connect("./app.db", 10*time.Second)
redisClient, err := redis.Connect("localhost:6379", 10*time.Second)
```

---

[← Back to Examples](../README.md)
