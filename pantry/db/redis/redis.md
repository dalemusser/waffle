# redis

Redis connection utilities for WAFFLE applications.

## Overview

The `redis` package provides connection helpers for Redis with support for standalone, cluster, and sentinel deployments with timeout-bounded connections and connectivity verification.

## Import

```go
import "github.com/dalemusser/waffle/db/redis"
```

---

## Connect

**Location:** `redis.go`

```go
func Connect(addr string, timeout time.Duration) (*redis.Client, error)
```

Opens a Redis connection using the given address (host:port). Pings to verify connectivity before returning.

**Example:**

```go
client, err := redis.Connect("localhost:6379", 10*time.Second)
if err != nil {
    return err
}
defer client.Close()
```

---

## ConnectWithPassword

**Location:** `redis.go`

```go
func ConnectWithPassword(addr, password string, db int, timeout time.Duration) (*redis.Client, error)
```

Opens a Redis connection with authentication and database selection.

**Example:**

```go
client, err := redis.ConnectWithPassword("localhost:6379", "secret", 0, 10*time.Second)
```

---

## ConnectWithOptions

**Location:** `redis.go`

```go
func ConnectWithOptions(opts *redis.Options, timeout time.Duration) (*redis.Client, error)
```

Opens a Redis connection with full configuration control.

**Example:**

```go
client, err := redis.ConnectWithOptions(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "secret",
    DB:           0,
    PoolSize:     10,
    MinIdleConns: 2,
}, core.DBConnectTimeout)
```

---

## ConnectURL

**Location:** `redis.go`

```go
func ConnectURL(url string, timeout time.Duration) (*redis.Client, error)
```

Opens a Redis connection using a URL.

**URL formats:**

```
redis://localhost:6379
redis://:password@localhost:6379/0
rediss://localhost:6379  (TLS)
```

**Example:**

```go
client, err := redis.ConnectURL("redis://:secret@localhost:6379/0", 10*time.Second)
```

---

## ConnectCluster

**Location:** `redis.go`

```go
func ConnectCluster(addrs []string, password string, timeout time.Duration) (*redis.ClusterClient, error)
```

Opens a Redis Cluster connection for horizontal scaling.

**Example:**

```go
client, err := redis.ConnectCluster([]string{
    "node1:6379",
    "node2:6379",
    "node3:6379",
}, "", core.DBConnectTimeout)
```

---

## ConnectSentinel

**Location:** `redis.go`

```go
func ConnectSentinel(masterName string, sentinelAddrs []string, password string, timeout time.Duration) (*redis.Client, error)
```

Opens a Redis connection via Sentinel for high availability with automatic failover.

**Example:**

```go
client, err := redis.ConnectSentinel("mymaster", []string{
    "sentinel1:26379",
    "sentinel2:26379",
    "sentinel3:26379",
}, "", core.DBConnectTimeout)
```

---

## HealthCheck

**Location:** `redis.go`

```go
func HealthCheck(client *redis.Client) func(ctx context.Context) error
```

Returns a health check function compatible with the health package.

**Example:**

```go
health.Mount(r, map[string]health.Check{
    "redis": redis.HealthCheck(redisClient),
}, logger)
```

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    redisClient, err := redis.ConnectWithPassword(
        appCfg.RedisAddr,
        appCfg.RedisPassword,
        0,
        core.DBConnectTimeout,
    )
    if err != nil {
        return DBDeps{}, fmt.Errorf("redis: %w", err)
    }

    logger.Info("connected to Redis")

    return DBDeps{Redis: redisClient}, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.Redis != nil {
        if err := db.Redis.Close(); err != nil {
            return fmt.Errorf("redis close: %w", err)
        }
        logger.Info("disconnected from Redis")
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    RedisAddr     string `conf:"redis_addr"`
    RedisPassword string `conf:"redis_password"`
}
```

```bash
# Environment variables
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=secret
```

---

## Common Options

When using `ConnectWithOptions`, common settings include:

| Setting | Description | Default |
|---------|-------------|---------|
| `Addr` | Redis server address | localhost:6379 |
| `Password` | Authentication password | "" |
| `DB` | Database number (0-15) | 0 |
| `PoolSize` | Maximum connections | 10 x GOMAXPROCS |
| `MinIdleConns` | Minimum idle connections | 0 |
| `MaxRetries` | Max retries before giving up | 3 |
| `DialTimeout` | Timeout for establishing connection | 5s |
| `ReadTimeout` | Timeout for read operations | 3s |
| `WriteTimeout` | Timeout for write operations | ReadTimeout |

---

## Basic Usage

```go
ctx := context.Background()

// String operations
err := client.Set(ctx, "key", "value", time.Hour).Err()
val, err := client.Get(ctx, "key").Result()

// Hash operations
err := client.HSet(ctx, "user:1", "name", "Alice", "email", "alice@example.com").Err()
name, err := client.HGet(ctx, "user:1", "name").Result()

// List operations
err := client.LPush(ctx, "queue", "task1", "task2").Err()
task, err := client.RPop(ctx, "queue").Result()

// Set operations
err := client.SAdd(ctx, "tags", "go", "redis", "waffle").Err()
tags, err := client.SMembers(ctx, "tags").Result()

// Pub/Sub
pubsub := client.Subscribe(ctx, "channel")
ch := pubsub.Channel()
for msg := range ch {
    fmt.Println(msg.Payload)
}
```

---

## See Also

- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
