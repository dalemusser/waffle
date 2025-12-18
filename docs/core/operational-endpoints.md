# WAFFLE Operational Endpoints Guide
*Health checks, metrics, and profiling for production-ready services.*

This guide covers WAFFLE's built-in packages for operational endpoints:

- **Health checks** — Verify your service and its dependencies are healthy
- **Metrics** — Expose Prometheus-compatible metrics for monitoring
- **pprof** — Enable Go's profiling tools for debugging performance

These are WAFFLE-provided packages, not custom middleware you need to implement.

---

# 📂 Where These Endpoints Are Mounted

All operational endpoints are mounted in your `BuildHandler` function:

```text
internal/app/bootstrap/routes.go
```

The generated `routes.go` includes commented examples showing how to enable each endpoint.

---

# ❤️ Health Checks

Health endpoints let load balancers and orchestrators (Kubernetes, ECS, etc.) verify your service is ready to receive traffic.

## Package

```go
import "github.com/dalemusser/waffle/pantry/health"
```

## Basic Usage

### Simple Liveness Probe

If you just need to confirm the process is running:

```go
// internal/app/bootstrap/routes.go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Simple liveness — returns {"status": "ok"}
    health.Mount(r, nil, logger)

    return r, nil
}
```

### With Dependency Checks

For readiness probes that verify your dependencies are healthy:

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Health checks with dependency verification
    checks := map[string]health.Check{
        "mongo": func(ctx context.Context) error {
            return deps.MongoClient.Ping(ctx, nil)
        },
        "redis": func(ctx context.Context) error {
            return deps.RedisClient.Ping(ctx).Err()
        },
    }
    health.Mount(r, checks, logger)

    return r, nil
}
```

## Response Format

**All checks pass (HTTP 200):**

```json
{
  "status": "ok",
  "checks": {
    "mongo": "ok",
    "redis": "ok"
  }
}
```

**Any check fails (HTTP 503):**

```json
{
  "status": "error",
  "checks": {
    "mongo": "ok",
    "redis": "error: connection refused"
  }
}
```

## Custom Path

Use `MountAt` to mount at a custom path:

```go
// Mount at /ready instead of /health
health.MountAt(r, "/ready", checks, logger)
```

## Kubernetes Example

In Kubernetes, configure your pod spec:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

# 📊 Metrics

WAFFLE integrates with Prometheus for metrics collection and exposure.

## Package

```go
import "github.com/dalemusser/waffle/metrics"
```

## How It Works

WAFFLE's `router.New()` automatically includes the `metrics.HTTPMetrics` middleware, which records request durations to a histogram. You only need to expose the `/metrics` endpoint for Prometheus to scrape.

## Exposing the Metrics Endpoint

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Expose metrics for Prometheus scraping
    r.Handle("/metrics", metrics.Handler())

    return r, nil
}
```

## Metrics Collected

WAFFLE automatically registers:

| Metric | Type | Description |
|---|---|---|
| `http_request_duration_seconds` | Histogram | Request duration labeled by path, method, status |
| Go runtime metrics | Various | Memory, goroutines, GC stats |
| Process metrics | Various | CPU, file descriptors, start time |

## Prometheus Configuration

Add your service to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'myservice'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

## Protecting the Metrics Endpoint

In production, you may want to restrict access:

```go
r.Group(func(r chi.Router) {
    r.Use(internalOnly)  // Your middleware to restrict access
    r.Handle("/metrics", metrics.Handler())
})
```

---

# 🔬 pprof (Profiling)

Go's pprof tools help diagnose performance issues like CPU bottlenecks, memory leaks, and goroutine problems.

## Package

```go
import "github.com/dalemusser/waffle/pantry/pprof"
```

## Basic Usage

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Mount pprof at /debug/pprof
    pprof.Mount(r)

    return r, nil
}
```

## Available Endpoints

| Endpoint | Description |
|---|---|
| `/debug/pprof/` | Index page with links to all profiles |
| `/debug/pprof/heap` | Heap memory allocation profile |
| `/debug/pprof/goroutine` | Stack traces of all goroutines |
| `/debug/pprof/allocs` | Memory allocation profile |
| `/debug/pprof/block` | Blocking profile (requires `runtime.SetBlockProfileRate`) |
| `/debug/pprof/profile` | CPU profile (30-second default) |
| `/debug/pprof/trace` | Execution trace |
| `/debug/pprof/cmdline` | Command line arguments |
| `/debug/pprof/symbol` | Symbol lookup |

## Environment-Based Enabling

Only enable in development or when explicitly configured:

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Only enable pprof in development
    if coreCfg.Env == "dev" {
        pprof.Mount(r)
    }

    return r, nil
}
```

Or use an app-specific flag:

```go
// In appconfig.go
type AppConfig struct {
    EnablePprof bool `conf:"enable_pprof" conf-default:"false"`
}

// In routes.go
if appCfg.EnablePprof {
    pprof.Mount(r)
}
```

## Protecting pprof in Production

If you need pprof in production, protect it with authentication:

```go
r.Group(func(r chi.Router) {
    r.Use(adminOnly)  // Your auth middleware
    pprof.Mount(r)
})
```

## Using pprof

### Capture a CPU Profile

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

### View Heap Profile

```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

### View All Goroutines

```bash
curl http://localhost:8080/debug/pprof/goroutine?debug=2
```

### Interactive Web UI

```bash
go tool pprof -http=:6060 http://localhost:8080/debug/pprof/heap
```

---

# 🧩 Complete Example

Here's a `BuildHandler` with all operational endpoints configured:

```go
package bootstrap

import (
    "context"
    "net/http"

    "github.com/dalemusser/waffle/config"
    "github.com/dalemusser/waffle/metrics"
    "github.com/dalemusser/waffle/pantry/health"
    "github.com/dalemusser/waffle/pantry/pprof"
    "github.com/dalemusser/waffle/router"
    "go.uber.org/zap"
)

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // ─────────────────────────────────────────────
    // Feature routes (your application logic)
    // ─────────────────────────────────────────────
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(appCfg.Greeting))
    })

    // Mount feature modules here...
    // r.Mount("/users", users.Routes(deps, logger))

    // ─────────────────────────────────────────────
    // Operational endpoints
    // ─────────────────────────────────────────────

    // Health checks — for load balancers and orchestrators
    checks := map[string]health.Check{
        "db": func(ctx context.Context) error {
            return deps.MongoClient.Ping(ctx, nil)
        },
    }
    health.Mount(r, checks, logger)

    // Metrics — for Prometheus scraping
    r.Handle("/metrics", metrics.Handler())

    // pprof — for debugging (development only)
    if coreCfg.Env == "dev" {
        pprof.Mount(r)
    }

    return r, nil
}
```

---

# 📋 Summary

| Package | Import | Mount Function | Default Path |
|---|---|---|---|
| Health | `github.com/dalemusser/waffle/pantry/health` | `health.Mount(r, checks, logger)` | `/health` |
| Metrics | `github.com/dalemusser/waffle/metrics` | `r.Handle("/metrics", metrics.Handler())` | `/metrics` |
| pprof | `github.com/dalemusser/waffle/pantry/pprof` | `pprof.Mount(r)` | `/debug/pprof/*` |

---

## See Also

- [Routes & Middleware Guide](./routing.md) — Feature routing patterns
- [Development Guide](../guides/development/README.md) — Full WAFFLE reference
- [WAFFLE Quickstart Guide](../guides/getting-started/quickstart.md) — Quick overview
