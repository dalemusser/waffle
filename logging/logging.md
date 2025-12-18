# logging

Structured logging and HTTP middleware for request logging and panic recovery.

## Overview

The `logging` package provides zap-based structured logging configured for different environments, plus HTTP middleware for logging requests and recovering from panics. It handles the two-phase logging pattern: a bootstrap logger for early startup, then a configured logger once settings are loaded.

## Import

```go
import "github.com/dalemusser/waffle/logging"
```

## Quick Start

```go
func main() {
    // Early startup logging (before config is loaded)
    bootLog := logging.BootstrapLogger()
    bootLog.Info("starting application")

    // Load config...
    cfg, err := config.Load()
    if err != nil {
        bootLog.Fatal("config load failed", zap.Error(err))
    }

    // Build configured logger
    logger := logging.MustBuildLogger(cfg.LogLevel, cfg.Env)
    defer logger.Sync()

    // Use in router
    r := chi.NewRouter()
    r.Use(logging.RequestLogger(logger))
    r.Use(logging.Recoverer(logger))
}
```

## API

### BootstrapLogger

**Location:** `logging.go`

```go
func BootstrapLogger() *zap.Logger
```

Returns a development-friendly logger for early startup, before configuration is loaded. Logs to stderr at Info level with ISO8601 timestamps. Safe to call without any setup.

**Example:**

```go
func main() {
    logger := logging.BootstrapLogger()
    logger.Info("loading configuration...")

    // Now load config and build the real logger
}
```

### BuildLogger

**Location:** `logging.go`

```go
func BuildLogger(level, env string) (*zap.Logger, error)
```

Constructs the application logger based on log level and environment.

**Parameters:**
- `level` — Log level: "debug", "info", "warn", "error" (defaults to "info" if invalid)
- `env` — Environment: "prod" uses JSON encoding, anything else uses development format

**Behavior:**
- Production (`env == "prod"`): JSON encoder, suitable for log aggregation
- Development (any other value): Human-readable console format with colors
- All output goes to stderr
- Timestamps use ISO8601/RFC-3339 format

**Example:**

```go
logger, err := logging.BuildLogger("debug", "dev")
if err != nil {
    log.Fatal(err)
}
defer logger.Sync()
```

### MustBuildLogger

**Location:** `logging.go`

```go
func MustBuildLogger(level, env string) *zap.Logger
```

Convenience wrapper that calls `BuildLogger` and exits on failure. Use in `main()` where you want to fail fast.

**Example:**

```go
func main() {
    logger := logging.MustBuildLogger(cfg.LogLevel, cfg.Env)
    defer logger.Sync()
}
```

### RequestLogger

**Location:** `requestmw.go`

```go
func RequestLogger(logger *zap.Logger) func(next http.Handler) http.Handler
```

Returns middleware that logs every HTTP request with structured fields.

**Logged fields:**
| Field | Description |
|-------|-------------|
| `method` | HTTP method (GET, POST, etc.) |
| `path` | Request path |
| `host` | Request host |
| `scheme` | http or https |
| `proto` | HTTP protocol version |
| `status` | Response status code |
| `bytes` | Response body size |
| `remote_ip` | Client IP address |
| `user_agent` | Client user agent |
| `referer` | Referer header |
| `latency` | Request duration |
| `request_id` | Chi request ID (if middleware enabled) |

**Example:**

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)  // Chi's request ID middleware
r.Use(logging.RequestLogger(logger))
```

**Sample output (development):**

```
INFO  http_request  {"method": "GET", "path": "/api/users", "status": 200, "bytes": 1234, "latency": "12.5ms"}
```

### Recoverer

**Location:** `recovermw.go`

```go
func Recoverer(logger *zap.Logger) func(next http.Handler) http.Handler
```

Returns middleware that recovers from panics, logs the panic with a stack trace, and returns HTTP 500.

**Logged fields on panic:**
| Field | Description |
|-------|-------------|
| `panic_value` | The value passed to panic() |
| `stacktrace` | Full stack trace |
| `method` | HTTP method |
| `path` | Request path |
| `remote_ip` | Client IP |

**Example:**

```go
r := chi.NewRouter()
r.Use(logging.Recoverer(logger))  // Should be early in the chain
r.Use(logging.RequestLogger(logger))
```

## Patterns

### Standard Middleware Stack

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Recovery first (catches panics from all other middleware)
    r.Use(logging.Recoverer(logger))

    // Request ID for correlation
    r.Use(middleware.RequestID)

    // Request logging
    r.Use(logging.RequestLogger(logger))

    // ... other middleware and routes

    return r, nil
}
```

### Two-Phase Logging in main()

```go
func main() {
    // Phase 1: Bootstrap logger for early startup
    bootLog := logging.BootstrapLogger()
    bootLog.Info("application starting")

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        bootLog.Fatal("failed to load config", zap.Error(err))
    }

    // Phase 2: Configured logger for application runtime
    logger := logging.MustBuildLogger(cfg.Core.LogLevel, cfg.Core.Env)
    defer logger.Sync()

    logger.Info("configuration loaded",
        zap.String("env", cfg.Core.Env),
        zap.String("log_level", cfg.Core.LogLevel),
    )

    // Continue with app startup...
}
```

### Child Loggers with Context

```go
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "id")

    // Create a child logger with request context
    log := h.logger.With(
        zap.String("handler", "GetUser"),
        zap.String("user_id", userID),
    )

    log.Debug("fetching user")

    user, err := h.db.FindUser(r.Context(), userID)
    if err != nil {
        log.Error("user lookup failed", zap.Error(err))
        // ...
    }
}
```

### Log Levels by Environment

```go
// Development: verbose logging
logger := logging.MustBuildLogger("debug", "dev")

// Staging: moderate logging
logger := logging.MustBuildLogger("info", "staging")

// Production: errors and warnings only, JSON format
logger := logging.MustBuildLogger("warn", "prod")
```

## See Also

- [app](../app/app.md) — Application lifecycle
- [config](../config/config.md) — Configuration loading
- [middleware](../middleware/middleware.md) — Additional middleware

