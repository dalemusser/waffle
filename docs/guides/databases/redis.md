# Redis in DBDeps  
*How to add a Redis client to your WAFFLE application's DBDeps and ConnectDB flow.*

This example shows the **minimum practical setup** for using Redis inside a WAFFLE app.  
It uses the popular `github.com/redis/go-redis/v9` client.

Use this pattern for:

- caching  
- rate limiting  
- session storage  
- ephemeral key/value data  

If you're integrating Redis for more advanced patterns, you can layer additional utilities on top of this setup.

---

# 📁 File: `internal/app/bootstrap/dbdeps.go`

DBDeps should hold long-lived backend dependencies.  
For Redis, this is typically a `*redis.Client`.

```go
package bootstrap

import "github.com/redis/go-redis/v9"

// DBDeps holds external backend dependencies.
// WAFFLE constructs one DBDeps at startup and passes it everywhere needed.
type DBDeps struct {
    Redis *redis.Client
}
```

---

# ⚙️ File: `internal/app/bootstrap/hooks.go`  
## Add Redis to ConnectDB

### Requirements:

```bash
go get github.com/redis/go-redis/v9
```

### Example ConnectDB implementation:

```go
import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
    "github.com/dalemusser/waffle/config"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // In real apps, put this in AppConfig or environment variables.
    redisAddr := "localhost:6379"

    client := redis.NewClient(&redis.Options{
        Addr:         redisAddr,
        DB:           0,              // use default DB
        PoolSize:     10,             // adjust for your needs
        MinIdleConns: 2,
    })

    // Test connection using PING
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := client.Ping(pingCtx).Err(); err != nil {
        return DBDeps{}, err
    }

    deps := DBDeps{
        Redis: client,
    }

    logger.Info("Redis connected", zap.String("addr", redisAddr))
    return deps, nil
}
```

---

# 💾 Using Redis in a Feature

Inside any feature, Redis is available through `deps.Redis`.

### Example: incrementing a counter

```go
package stats

import (
    "fmt"
    "net/http"

    "github.com/you/hello/internal/app/bootstrap"
    "github.com/redis/go-redis/v9"
)

type Handler struct {
    deps bootstrap.DBDeps
}

func NewHandler(deps bootstrap.DBDeps) *Handler {
    return &Handler{deps: deps}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    count, err := h.deps.Redis.Incr(r.Context(), "page_hits").Result()
    if err != nil && err != redis.Nil {
        http.Error(w, "redis error", http.StatusInternalServerError)
        return
    }

    _, _ = w.Write([]byte(fmt.Sprintf("Page hits: %d", count)))
}
```

You can mount this handler in your feature’s `Routes()` or `MountRoutes()` as usual.

---

# 🧹 Clean Shutdown

Redis clients should be closed during graceful shutdown.
WAFFLE provides the `Shutdown` hook for this purpose:

### File: `internal/app/bootstrap/shutdown.go`

```go
func Shutdown(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    if deps.Redis != nil {
        if err := deps.Redis.Close(); err != nil {
            logger.Warn("error closing Redis", zap.Error(err))
            return err
        }
        logger.Info("Redis connection closed")
    }
    return nil
}
```

---

# 🔧 Using WAFFLE Pantry Helper (Alternative)

WAFFLE provides convenience helpers in `pantry/db/redis`:

```go
import wredis "github.com/dalemusser/waffle/pantry/db/redis"

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    // Simple connection
    client, err := wredis.Connect("localhost:6379", coreCfg.DBConnectTimeout)

    // Or with password and DB selection
    client, err := wredis.ConnectWithPassword("localhost:6379", "password", 0, 10*time.Second)

    // Or with URL
    client, err := wredis.ConnectURL("redis://:password@localhost:6379/0", 10*time.Second)

    // Or with full options
    client, err := wredis.ConnectWithOptions(&wredis.Options{
        Addr:         "localhost:6379",
        Password:     "secret",
        DB:           0,
        PoolSize:     10,
        MinIdleConns: 2,
    }, 10*time.Second)

    if err != nil {
        return DBDeps{}, err
    }

    return DBDeps{Redis: client}, nil
}
```

The pantry also provides `ConnectCluster()` and `ConnectSentinel()` for Redis Cluster and Sentinel deployments.

---

# 🎯 Summary

To integrate Redis into WAFFLE:

1. Add a `*redis.Client` field to `DBDeps`  
2. Initialize the client inside `ConnectDB`  
3. Use Redis commands (e.g., `Get`, `Set`, `Incr`, `Del`) from your feature handlers  
4. Close the client during shutdown  

Common next steps:

- Add Redis-based rate limiting  
- Cache DB queries  
- Store session/user info  
- Track analytics counters  
