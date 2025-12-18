# Logging

*Structured logging with zap in WAFFLE applications.*

---

## Overview

WAFFLE provides two loggers:

1. **Bootstrap Logger** — Minimal logger available before config loads
2. **Final Logger** — Environment- and level-aware based on config

Both use [zap](https://github.com/uber-go/zap), Uber's high-performance structured logging library.

---

## Logger Availability

Every hook receives the final logger:

```go
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error)
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error)
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, dbDeps DBDeps, logger *zap.Logger) (http.Handler, error)
// etc.
```

---

## Structured Logging

Use structured fields for queryable logs:

```go
logger.Info("user logged in",
    zap.String("user_id", userID),
    zap.String("ip", clientIP),
)

logger.Error("database query failed",
    zap.Error(err),
    zap.String("query", queryName),
    zap.Duration("elapsed", elapsed),
)
```

---

## Log Levels

| Level | Use Case |
|-------|----------|
| `Debug` | Detailed troubleshooting information |
| `Info` | General operational events |
| `Warn` | Something unexpected but not an error |
| `Error` | Operation failed, needs attention |

```go
logger.Debug("cache miss", zap.String("key", key))
logger.Info("server started", zap.Int("port", port))
logger.Warn("deprecated endpoint called", zap.String("path", path))
logger.Error("failed to send email", zap.Error(err))
```

---

## Environment-Aware Output

WAFFLE configures the logger based on environment:

| Environment | Format | Level |
|-------------|--------|-------|
| Development | Console (human-readable) | Based on config |
| Production | JSON (machine-parseable) | Based on config |

---

## Child Loggers

Create child loggers with consistent context:

```go
// In handler constructor
func NewUserHandler(deps DBDeps, logger *zap.Logger) *UserHandler {
    return &UserHandler{
        deps:   deps,
        logger: logger.Named("user"),
    }
}

// In methods
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    h.logger.Info("creating user") // logs as "user: creating user"
}
```

---

## Request Logging

WAFFLE's router middleware automatically logs requests:

```
INFO    request completed    {"method": "GET", "path": "/api/users", "status": 200, "duration": "12.5ms"}
```

---

## Metrics

WAFFLE automatically registers:

- Runtime metrics (goroutines, memory)
- Process metrics (CPU, file descriptors)
- HTTP metrics (request duration histograms)

These are available at the metrics endpoint without additional configuration.

---

## See Also

- [Lifecycle](./lifecycle.md) — Logger initialization
- [Configuration](./configuration.md) — LogLevel setting
- [Operational Endpoints](../../core/operational-endpoints.md) — Metrics endpoint

---

[← Back to Development Guide](./README.md)
