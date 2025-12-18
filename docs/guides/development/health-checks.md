# Health Checks

*Unified health checking for WAFFLE services.*

---

## Overview

WAFFLE includes a health framework for monitoring service health and supporting load balancer health checks.

---

## Basic Setup

```go
import "github.com/dalemusser/waffle/health"

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, dbDeps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    checks := map[string]health.Check{
        "postgres": func(ctx context.Context) error {
            return dbDeps.Postgres.Ping(ctx)
        },
        "redis": func(ctx context.Context) error {
            return dbDeps.Redis.Ping(ctx).Err()
        },
    }

    health.Mount(r, checks, logger)

    // ... rest of routes
    return r, nil
}
```

---

## Response Behavior

| Condition | Response |
|-----------|----------|
| All checks pass | `200 OK` |
| Any check fails | `503 Service Unavailable` |

Response body includes individual check status:

```json
{
  "status": "healthy",
  "checks": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```

Or when unhealthy:

```json
{
  "status": "unhealthy",
  "checks": {
    "postgres": "ok",
    "redis": "connection refused"
  }
}
```

---

## HEAD Support

The health endpoint supports HEAD requests for load balancers that only check status codes:

```
HEAD /health HTTP/1.1
```

Returns `200 OK` or `503 Service Unavailable` with no body.

---

## Common Checks

### Database

```go
"postgres": func(ctx context.Context) error {
    return pool.Ping(ctx)
},
```

### Redis

```go
"redis": func(ctx context.Context) error {
    return client.Ping(ctx).Err()
},
```

### MongoDB

```go
"mongo": func(ctx context.Context) error {
    return client.Ping(ctx, nil)
},
```

### External API

```go
"payment-api": func(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/health", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("status %d", resp.StatusCode)
    }
    return nil
},
```

---

## Load Balancer Configuration

### AWS ALB

```yaml
HealthCheckPath: /health
HealthCheckProtocol: HTTP
HealthyThresholdCount: 2
UnhealthyThresholdCount: 3
HealthCheckIntervalSeconds: 30
```

### Kubernetes

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
  periodSeconds: 10
```

---

## See Also

- [Operational Endpoints](../../core/operational-endpoints.md) — Health, metrics, and pprof endpoints
- [Lifecycle](./lifecycle.md) — Service startup and shutdown

---

[← Back to Development Guide](./README.md)
