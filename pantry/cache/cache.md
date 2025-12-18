# cache

Caching for WAFFLE applications with in-memory and Redis backends.

## Overview

The `cache` package provides a unified caching interface with two implementations:
- **Memory** — In-memory cache with TTL and automatic cleanup
- **Redis** — Redis-backed cache for distributed applications

Also includes HTTP middleware for response caching.

## Import

```go
import "github.com/dalemusser/waffle/cache"
```

---

## Cache Interface

**Location:** `cache.go`

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Clear(ctx context.Context) error
    Close() error
}
```

All implementations satisfy this interface, allowing easy swapping between memory and Redis.

---

## Memory Cache

In-memory cache with TTL support and background cleanup.

### NewMemory

**Location:** `memory.go`

```go
func NewMemory() *Memory
func NewMemoryWithConfig(cfg MemoryConfig) *Memory
```

**MemoryConfig:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| CleanupInterval | time.Duration | 1 minute | How often to remove expired items |
| InitialCapacity | int | 100 | Initial map capacity |

### Basic Usage

```go
c := cache.NewMemory()
defer c.Close()

ctx := context.Background()

// Store value with 5 minute TTL
err := c.Set(ctx, "user:123", []byte(`{"name":"Alice"}`), 5*time.Minute)

// Retrieve value
data, err := c.Get(ctx, "user:123")
if errors.Is(err, cache.ErrNotFound) {
    // Key doesn't exist or expired
}

// Delete value
c.Delete(ctx, "user:123")

// Check existence
exists, _ := c.Exists(ctx, "user:123")

// Clear all entries
c.Clear(ctx)
```

### With Config

```go
c := cache.NewMemoryWithConfig(cache.MemoryConfig{
    CleanupInterval: 5 * time.Minute,
    InitialCapacity: 1000,
})
defer c.Close()
```

### Memory-Specific Methods

```go
c.Size()  // Number of items (including expired)
```

---

## Redis Cache

Redis-backed cache for distributed applications.

### Creating

**Location:** `redis.go`

```go
// With existing client
c := cache.NewRedis(redisClient)

// Simple connection
c, err := cache.Connect("localhost:6379", "", 0)

// With full config
c, err := cache.NewRedisWithConfig(cache.RedisConfig{
    Address:      "localhost:6379",
    Password:     "secret",
    DB:           0,
    KeyPrefix:    "myapp:",
    PoolSize:     20,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

**RedisConfig:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Client | redis.UniversalClient | nil | Existing client (ignores other options) |
| Address | string | required | Redis server address |
| Password | string | "" | Authentication password |
| DB | int | 0 | Database number |
| KeyPrefix | string | "" | Prefix for all keys |
| PoolSize | int | 10 | Connection pool size |
| DialTimeout | time.Duration | 5s | Connection timeout |
| ReadTimeout | time.Duration | 3s | Read operation timeout |
| WriteTimeout | time.Duration | 3s | Write operation timeout |

### Basic Usage

```go
c, err := cache.Connect("localhost:6379", "", 0)
if err != nil {
    log.Fatal(err)
}
defer c.Close()

ctx := context.Background()

// Same interface as Memory
c.Set(ctx, "key", []byte("value"), time.Hour)
data, err := c.Get(ctx, "key")
c.Delete(ctx, "key")
```

### Redis-Specific Methods

```go
// Atomic set if not exists
set, err := c.SetNX(ctx, "lock:123", []byte("owner"), 10*time.Second)

// Get and replace
oldValue, err := c.GetSet(ctx, "counter", []byte("new"))

// Increment/decrement numeric values
newVal, err := c.Incr(ctx, "counter")
newVal, err := c.IncrBy(ctx, "counter", 5)
newVal, err := c.Decr(ctx, "counter")

// TTL management
ttl, err := c.TTL(ctx, "key")
c.Expire(ctx, "key", time.Hour)

// Access underlying client
client := c.Client()
```

---

## Helper Functions

### JSON Operations

**Location:** `cache.go`

```go
// Get and unmarshal JSON
user, err := cache.GetJSON[User](ctx, c, "user:123")

// Marshal and set JSON
err := cache.SetJSON(ctx, c, "user:123", user, time.Hour)
```

### Get or Compute

```go
// Get value or compute if missing
data, err := cache.GetOrSet(ctx, c, "expensive:key", time.Hour, func() ([]byte, error) {
    return fetchExpensiveData()
})

// JSON version
user, err := cache.GetOrSetJSON(ctx, c, "user:123", time.Hour, func() (*User, error) {
    return db.GetUser(ctx, 123)
})
```

---

## Batch Operations

Both Memory and Redis support batch operations for efficiency.

```go
// Get multiple keys
values, err := c.GetMulti(ctx, []string{"key1", "key2", "key3"})
for key, val := range values {
    // Process value
}

// Set multiple keys
items := map[string][]byte{
    "key1": []byte("value1"),
    "key2": []byte("value2"),
}
err := c.SetMulti(ctx, items, time.Hour)

// Delete multiple keys
err := c.DeleteMulti(ctx, []string{"key1", "key2", "key3"})
```

---

## HTTP Middleware

Cache HTTP responses automatically.

### Middleware

**Location:** `middleware.go`

```go
func Middleware(c Cache, cfg MiddlewareConfig) func(http.Handler) http.Handler
```

**MiddlewareConfig:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| TTL | time.Duration | 5 minutes | Cache duration |
| KeyFunc | func(*http.Request) string | DefaultKeyFunc | Key generation |
| KeyPrefix | string | "" | Prefix for cache keys |
| Skip | func(*http.Request) bool | nil | Skip caching condition |
| CacheErrors | bool | false | Cache non-2xx responses |

### Basic Usage

```go
c := cache.NewMemory()

r := chi.NewRouter()
r.Use(cache.Middleware(c, cache.MiddlewareConfig{
    TTL: 5 * time.Minute,
}))

r.Get("/api/products", listProducts)  // Responses cached for 5 minutes
```

### Skip Certain Routes

```go
r.Use(cache.Middleware(c, cache.MiddlewareConfig{
    TTL: 10 * time.Minute,
    Skip: func(r *http.Request) bool {
        // Don't cache authenticated requests
        return r.Header.Get("Authorization") != ""
    },
}))
```

### Key Functions

```go
// Default: method + path + query string
cache.DefaultKeyFunc  // "GET:/api/users?page=1"

// Hash for long URLs
cache.HashKeyFunc     // "a1b2c3d4..."

// Path only (ignores query string)
cache.PathKeyFunc     // "GET:/api/users"

// Custom key function
cfg := cache.MiddlewareConfig{
    KeyFunc: func(r *http.Request) string {
        // Include user ID in cache key
        userID := r.Header.Get("X-User-ID")
        return userID + ":" + r.URL.Path
    },
}
```

### Per-Route Caching

```go
r := chi.NewRouter()

// No caching for most routes
r.Get("/api/users/{id}", getUser)

// Cache specific routes
r.Group(func(r chi.Router) {
    r.Use(cache.Middleware(c, cache.MiddlewareConfig{
        TTL: time.Hour,
    }))
    r.Get("/api/products", listProducts)
    r.Get("/api/categories", listCategories)
})
```

---

## WAFFLE Integration

### Application Setup

```go
func main() {
    logger, _ := zap.NewProduction()

    // Choose cache based on environment
    var c cache.Cache
    if os.Getenv("REDIS_URL") != "" {
        var err error
        c, err = cache.Connect(os.Getenv("REDIS_URL"), "", 0)
        if err != nil {
            log.Fatal(err)
        }
    } else {
        c = cache.NewMemory()
    }

    app := waffle.New(waffle.Config{
        Logger: logger,
        Shutdown: func(ctx context.Context) error {
            return c.Close()
        },
    })

    // Make cache available to handlers
    app.Set("cache", c)

    setupRoutes(app.Router(), c)
    app.Run()
}
```

### In Handlers

```go
func getUserHandler(c cache.Cache) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := chi.URLParam(r, "id")
        cacheKey := "user:" + userID

        // Try cache first
        user, err := cache.GetOrSetJSON(r.Context(), c, cacheKey, 5*time.Minute, func() (*User, error) {
            return db.GetUser(r.Context(), userID)
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(user)
    }
}
```

### Cache Invalidation

```go
func updateUserHandler(c cache.Cache) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := chi.URLParam(r, "id")

        // Update database...

        // Invalidate cache
        c.Delete(r.Context(), "user:"+userID)
        c.Delete(r.Context(), "user:list") // Invalidate list too

        w.WriteHeader(http.StatusOK)
    }
}
```

### Rate Limit Counter (Redis)

```go
func rateLimitMiddleware(c *cache.Redis) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            key := "ratelimit:" + ip

            count, err := c.Incr(r.Context(), key)
            if err != nil {
                next.ServeHTTP(w, r)
                return
            }

            // Set expiry on first request
            if count == 1 {
                c.Expire(r.Context(), key, time.Minute)
            }

            if count > 100 {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Errors

```go
cache.ErrNotFound  // Key doesn't exist or expired
cache.ErrClosed    // Cache has been closed
```

---

## See Also

- [db/redis](../db/redis/redis.md) — Redis database operations
- [ratelimit](../ratelimit/ratelimit.md) — Rate limiting middleware
- [middleware](../middleware/middleware.md) — HTTP middleware
