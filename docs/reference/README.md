# WAFFLE Reference

*Quick lookup tables and API documentation — minimal prose, maximum information density.*

---

## Purpose

Reference documentation is for developers who already understand WAFFLE and need to quickly look up specific information. It prioritizes:

- **Density** — Maximum information per scroll
- **Scannability** — Tables, code blocks, bullet points
- **Completeness** — Every option, every parameter
- **Accuracy** — Always up to date with the code

---

## Reference Documents

| Document | Contents |
|----------|----------|
| [**Configuration Variables**](./config-vars.md) | All CoreConfig variables, defaults, and constraints |
| [**File Reference**](./file-reference.md) | Complete documentation for every WAFFLE file |

---

## Quick Links

### CoreConfig Fields

```go
type CoreConfig struct {
    Env                 string        // "dev" or "prod"
    LogLevel            string        // debug, info, warn, error
    HTTP                HTTPConfig    // Ports, timeouts
    TLS                 TLSConfig     // HTTPS, Let's Encrypt
    CORS                CORSConfig    // Cross-origin settings
    DBConnectTimeout    time.Duration // 10s default
    IndexBootTimeout    time.Duration // 120s default
    MaxRequestBodyBytes int64         // 2MB default
    EnableCompression   bool          // true default
    CompressionLevel    int           // 1-9, default 5
}
```

### HTTPConfig Fields

```go
type HTTPConfig struct {
    HTTPPort          int           // 8080 default
    HTTPSPort         int           // 443 default
    UseHTTPS          bool          // false default
    ReadTimeout       time.Duration // 15s default
    ReadHeaderTimeout time.Duration // 10s default
    WriteTimeout      time.Duration // 60s default
    IdleTimeout       time.Duration // 120s default
    ShutdownTimeout   time.Duration // 15s default
}
```

### Hooks Struct

```go
type Hooks[C, D any] struct {
    Name           string
    LoadConfig     func(*zap.Logger) (*config.CoreConfig, C, error)
    ValidateConfig func(*config.CoreConfig, C, *zap.Logger) error      // optional
    ConnectDB      func(context.Context, *config.CoreConfig, C, *zap.Logger) (D, error)
    EnsureSchema   func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error // optional
    Startup        func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error // optional
    BuildHandler   func(*config.CoreConfig, C, D, *zap.Logger) (http.Handler, error)
    OnReady        func(*config.CoreConfig, C, D, *zap.Logger)         // optional
    Shutdown       func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error // optional
}
```

### Common Imports

```go
import (
    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/config"
    "github.com/dalemusser/waffle/router"
    "github.com/dalemusser/waffle/middleware"
    "github.com/dalemusser/waffle/health"
    "github.com/dalemusser/waffle/pantry/email"
    "github.com/dalemusser/waffle/pantry/validate"
    "github.com/dalemusser/waffle/pantry/db/postgres"
)
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WAFFLE_ENV` | `dev` | Runtime environment |
| `WAFFLE_LOG_LEVEL` | `debug` | Log level |
| `WAFFLE_HTTP_PORT` | `8080` | HTTP port |
| `WAFFLE_USE_HTTPS` | `false` | Enable HTTPS |
| `WAFFLE_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown timeout |
| `WAFFLE_ENABLE_COMPRESSION` | `true` | HTTP compression |

See [Configuration Variables](./config-vars.md) for the complete list.

---

## See Also

- [Core Documentation](../core/README.md) — Conceptual explanations
- [Guides](../guides/README.md) — Task-oriented how-tos
- [Pantry Documentation](../pantry/README.md) — Utility package documentation
- [Philosophy](../philosophy/README.md) — Why things are designed this way

---

[← Back to Documentation](../waffle-docs.md)
