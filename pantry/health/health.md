# health

Health check endpoints for load balancers, orchestrators, and monitoring systems.

## Overview

The `health` package provides configurable health check endpoints that report the status of your application and its dependencies. Use these for Kubernetes liveness/readiness probes, load balancer health checks, or monitoring systems.

## Import

```go
import "github.com/dalemusser/waffle/health"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Simple liveness probe (no checks)
    health.MountAt(r, "/live", nil, logger)

    // Readiness probe with dependency checks
    health.Mount(r, map[string]health.Check{
        "mongo": func(ctx context.Context) error {
            return db.MongoClient.Ping(ctx, readpref.Primary())
        },
        "redis": func(ctx context.Context) error {
            return db.Redis.Ping(ctx).Err()
        },
    }, logger)

    // ... other routes

    return r, nil
}
```

## API

### Check

**Location:** `health.go`

```go
type Check func(ctx context.Context) error
```

A health check function. Return `nil` if healthy, or an error describing the problem.

### Response

**Location:** `health.go`

```go
type Response struct {
    Status string            `json:"status"`          // "ok" or "error"
    Checks map[string]string `json:"checks,omitempty"` // Per-check results
}
```

### Handler

**Location:** `health.go`

```go
func Handler(checks map[string]Check, logger *zap.Logger) http.Handler
```

Returns an HTTP handler that runs all checks and returns a JSON response.

**Behavior:**
- No checks: Returns `{"status": "ok"}` with 200
- All checks pass: Returns `{"status": "ok", "checks": {...}}` with 200
- Any check fails: Returns `{"status": "error", "checks": {...}}` with 503

### Mount

**Location:** `health.go`

```go
func Mount(r chi.Router, checks map[string]Check, logger *zap.Logger)
```

Attaches a health check handler at `/health`.

### MountAt

**Location:** `health.go`

```go
func MountAt(r chi.Router, path string, checks map[string]Check, logger *zap.Logger)
```

Attaches a health check handler at a custom path.

## Response Examples

**Healthy (no checks):**
```json
{"status": "ok"}
```

**Healthy (with checks):**
```json
{
  "status": "ok",
  "checks": {
    "mongo": "ok",
    "redis": "ok"
  }
}
```

**Unhealthy:**
```json
{
  "status": "error",
  "checks": {
    "mongo": "ok",
    "redis": "error: connection refused"
  }
}
```

## Patterns

### Kubernetes Probes

```go
// Liveness: Is the process alive? (no dependency checks)
health.MountAt(r, "/live", nil, logger)

// Readiness: Can the service handle requests? (check dependencies)
health.MountAt(r, "/ready", map[string]health.Check{
    "mongo": func(ctx context.Context) error {
        return db.MongoClient.Ping(ctx, readpref.Primary())
    },
}, logger)
```

```yaml
# Kubernetes deployment
livenessProbe:
  httpGet:
    path: /live
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Load Balancer Health Check

```go
// Single endpoint for ALB/ELB health checks
health.Mount(r, map[string]health.Check{
    "db": func(ctx context.Context) error {
        return db.MongoClient.Ping(ctx, readpref.Primary())
    },
}, logger)
```

### Custom Check with Timeout

```go
health.Mount(r, map[string]health.Check{
    "external_api": func(ctx context.Context) error {
        // Create a shorter timeout for the health check
        ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
        defer cancel()

        req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/health", nil)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("status %d", resp.StatusCode)
        }
        return nil
    },
}, logger)
```

### Multiple Check Types

```go
checks := map[string]health.Check{
    // Database
    "mongo": func(ctx context.Context) error {
        return db.MongoClient.Ping(ctx, readpref.Primary())
    },

    // Cache
    "redis": func(ctx context.Context) error {
        return db.Redis.Ping(ctx).Err()
    },

    // Message queue
    "rabbitmq": func(ctx context.Context) error {
        // Check if channel is open
        if db.RabbitChannel.IsClosed() {
            return errors.New("channel closed")
        }
        return nil
    },

    // Disk space (example)
    "disk": func(ctx context.Context) error {
        var stat syscall.Statfs_t
        if err := syscall.Statfs("/data", &stat); err != nil {
            return err
        }
        freeGB := (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024 * 1024)
        if freeGB < 1 {
            return fmt.Errorf("low disk space: %dGB free", freeGB)
        }
        return nil
    },
}

health.Mount(r, checks, logger)
```

## See Also

- [metrics](../metrics/metrics.md) — Prometheus metrics
- [pprof](../pprof/pprof.md) — Profiling endpoints
