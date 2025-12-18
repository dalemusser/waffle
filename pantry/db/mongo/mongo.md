# mongo

MongoDB/DocumentDB connection utilities for WAFFLE applications.

## Overview

The `mongo` package provides connection helpers for MongoDB and Amazon DocumentDB with timeout-bounded connections and connectivity verification.

## Import

```go
import "github.com/dalemusser/waffle/db/mongo"
```

---

## Connect

**Location:** `mongo.go`

```go
func Connect(uri string, timeout time.Duration) (*mongo.Client, error)
```

Opens a MongoDB or DocumentDB connection using the provided URI. Pings the primary to verify connectivity before returning. The caller is responsible for disconnecting.

**Parameters:**
- `uri` — MongoDB connection string (e.g., `mongodb://localhost:27017`)
- `timeout` — Maximum time to wait for connection and ping

**Example:**

```go
func connectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    client, err := mongo.Connect(appCfg.MongoURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("mongo connect: %w", err)
    }

    return DBDeps{
        MongoClient: client,
        DB:          client.Database(appCfg.MongoDB),
    }, nil
}
```

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    client, err := mongo.Connect(appCfg.MongoURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("mongo: %w", err)
    }

    logger.Info("connected to MongoDB",
        zap.String("database", appCfg.MongoDB),
    )

    return DBDeps{
        MongoClient: client,
        DB:          client.Database(appCfg.MongoDB),
    }, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.MongoClient != nil {
        if err := db.MongoClient.Disconnect(ctx); err != nil {
            return fmt.Errorf("mongo disconnect: %w", err)
        }
        logger.Info("disconnected from MongoDB")
    }
    return nil
}
```

---

## Configuration

Database URIs and names are typically stored in your app-specific config, not in WAFFLE's core config:

```go
// internal/app/bootstrap/appconfig.go
type AppConfig struct {
    MongoURI string `conf:"mongo_uri"`
    MongoDB  string `conf:"mongo_db"`
}
```

```bash
# Environment variables
MONGO_URI=mongodb://localhost:27017
MONGO_DB=myapp
```

---

## Timeout

The `timeout` parameter controls the maximum time for both the connection attempt and the initial ping. Use `core.DBConnectTimeout` from WAFFLE's core config, which defaults to 10 seconds and can be configured via:

- Config file: `db_connect_timeout: "30s"`
- Environment: `WAFFLE_DB_CONNECT_TIMEOUT=30s`
- Flag: `--db_connect_timeout=30s`

---

## db/mongo vs pantry/mongo

WAFFLE has two MongoDB-related packages:

| Package | Purpose |
|---------|---------|
| `db/mongo` | Connection management (Connect/Disconnect) |
| `pantry/mongo` | Query utilities (keyset pagination, cursor encoding, duplicate detection, URI validation) |

Use `db/mongo` in your `ConnectDB` hook to establish the connection. Use `pantry/mongo` in your repositories for query helpers.

```go
import (
    dbmongo "github.com/dalemusser/waffle/db/mongo"       // Connection
    "github.com/dalemusser/waffle/pantry/mongo"          // Query utilities
)
```

---

## See Also

- [pantry/mongo](../../pantry/mongo/mongo.md) — Query utilities (pagination, cursors, error detection)
- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
