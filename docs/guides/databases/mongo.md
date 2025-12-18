# MongoDB in DBDeps  
*How to add a MongoDB client to your WAFFLE application's DBDeps and ConnectDB flow.*

This example shows the **minimum practical code** needed to integrate MongoDB into a WAFFLE service.

It assumes you already understand the WAFFLE startup lifecycle and want a focused recipe for adding a Mongo client.

---

# 📁 File: `internal/app/bootstrap/dbdeps.go`

Your DBDeps struct should hold long‑lived dependencies.  
For MongoDB, that means a *Mongo client* and optionally a *database reference*.

```go
package bootstrap

import "go.mongodb.org/mongo-driver/mongo"

// DBDeps holds connections to external backends.
// WAFFLE creates one instance of DBDeps and passes it to your handlers.
type DBDeps struct {
    MongoClient *mongo.Client
    MongoDB     *mongo.Database
}
```

Nothing else happens here — DBDeps is just a struct.

---

# ⚙️ File: `internal/app/bootstrap/hooks.go`  
## Add MongoDB to ConnectDB

MongoDB setup belongs inside `ConnectDB`, because this is the phase where WAFFLE expects you to initialize external systems.

### Requirements:

```bash
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/mongo/options
```

### Example ConnectDB implementation:

```go
import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // You can place your connection string in AppConfig or env vars.
    mongoURI := "mongodb://localhost:27017"
    dbName := "waffle_app"

    // Create client options
    clientOpts := options.Client().ApplyURI(mongoURI)

    // Connect to MongoDB
    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        return DBDeps{}, err
    }

    // Ping to verify
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := client.Ping(pingCtx, nil); err != nil {
        return DBDeps{}, err
    }

    // Package into DBDeps
    deps := DBDeps{
        MongoClient: client,
        MongoDB:     client.Database(dbName),
    }

    logger.Info("MongoDB connected", zap.String("db", dbName))
    return deps, nil
}
```

---

# 💾 Using MongoDB in a Feature

Inside a feature handler:

```go
package items

import (
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
    collection := h.deps.MongoDB.Collection("items")

    // Example: Count documents
    count, _ := collection.CountDocuments(r.Context(), bson.D{})
    w.Write([]byte(fmt.Sprintf("Items in DB: %d", count)))
}
```

Then register your handler in your Routes or MountRoutes function.

---

# 🧹 Clean Shutdown (Recommended)

Mongo clients should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.MongoClient != nil {
        if err := deps.MongoClient.Disconnect(ctx); err != nil {
            logger.Error("failed to disconnect MongoDB", zap.Error(err))
            return err
        }
        logger.Info("MongoDB disconnected")
    }
    return nil
}
```

The `Shutdown` function is called after the HTTP server stops accepting new requests.

---

# 🔧 Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides a convenience helper in `pantry/db/mongo`:

```go
import "github.com/dalemusser/waffle/pantry/db/mongo"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    client, err := mongo.Connect(appCfg.MongoURI, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{
        MongoClient: client,
        MongoDB:     client.Database(appCfg.DBName),
    }, nil
}
```

This handles timeout and ping verification automatically.

---

# 🎯 Summary

To integrate MongoDB into WAFFLE:

1. Add fields to `DBDeps`  
2. Implement Mongo connection code in `ConnectDB`  
3. Use `deps.MongoDB.Collection("name")` in your handlers  
4. Optionally disconnect cleanly during shutdown  

The WAFFLE lifecycle makes database wiring predictable and clean.

