# db

Database connection utilities for WAFFLE applications.

## Overview

The `db` package provides connection helpers for databases with timeout-bounded connections and connectivity verification.

## Packages

| Package | Database | Description |
|---------|----------|-------------|
| [mongo](mongo/mongo.md) | MongoDB/DocumentDB | Document database with replica set support |
| [postgres](postgres/postgres.md) | PostgreSQL | Connection pooling via pgx |
| [mysql](mysql/mysql.md) | MySQL/MariaDB | Connection pooling via database/sql |
| [sqlite](sqlite/sqlite.md) | SQLite | Embedded database with WAL mode |
| [redis](redis/redis.md) | Redis | In-memory data store with cluster/sentinel support |

## Import

```go
import "github.com/dalemusser/waffle/db/mongo"
import "github.com/dalemusser/waffle/db/postgres"
import "github.com/dalemusser/waffle/db/mysql"
import "github.com/dalemusser/waffle/db/sqlite"
import "github.com/dalemusser/waffle/db/redis"
```

## Common Patterns

All database packages follow the same patterns:

- **Timeout-bounded connections** — All `Connect` functions accept a `timeout` parameter
- **Connectivity verification** — Connections are pinged before being returned
- **WAFFLE integration** — Use in `ConnectDB` and `Shutdown` hooks
- **Health checks** — Compatible with the health package (where applicable)

## Timeout Configuration

Use `core.DBConnectTimeout` from WAFFLE's core config, which defaults to 10 seconds:

- Config file: `db_connect_timeout: "30s"`
- Environment: `WAFFLE_DB_CONNECT_TIMEOUT=30s`
- Flag: `--db_connect_timeout=30s`

## Oracle Database

WAFFLE does not include a built-in Oracle package due to external dependencies (Oracle Instant Client, CGO). See the [Oracle integration guide](oracle.md) for using Oracle with WAFFLE via the godror driver.

## See Also

- [app](../app/app.md) — Application lifecycle hooks
- [config](../config/config.md) — Core configuration including `DBConnectTimeout`
- [pantry/mongo](../pantry/mongo/mongo.md) — MongoDB query utilities
