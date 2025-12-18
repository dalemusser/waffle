# Configuration

*How WAFFLE applications load and validate configuration.*

---

## Overview

WAFFLE applications load two configuration structures:

- **CoreConfig** — Shared across all WAFFLE services
- **AppConfig** — Specific to the individual service

Both are loaded in the `LoadConfig` hook:

```go
LoadConfig(logger) (*config.CoreConfig, AppConfig, error)
```

---

## CoreConfig

CoreConfig contains fields common to all WAFFLE services:

| Field | Type | Description |
|-------|------|-------------|
| `Env` | string | Environment (development, staging, production) |
| `LogLevel` | string | Logging level (debug, info, warn, error) |
| `HTTPPort` | int | HTTP server port |
| `HTTPSPort` | int | HTTPS server port |
| `UseHTTPS` | bool | Enable HTTPS |
| `UseLetsEncrypt` | bool | Use automatic Let's Encrypt certificates |
| `ShutdownTimeout` | duration | Graceful shutdown timeout |
| `DBConnectTimeout` | duration | Database connection timeout |
| `IndexBootTimeout` | duration | Schema initialization timeout |

---

## AppConfig

AppConfig is defined by each application for its specific needs:

```go
type AppConfig struct {
    DatabaseURL     string
    APIKey          string
    FeatureFlags    map[string]bool
    // ... your fields
}
```

---

## Loading Configuration

A typical `LoadConfig` implementation:

```go
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg := &config.CoreConfig{
        Env:             getEnv("ENV", "development"),
        LogLevel:        getEnv("LOG_LEVEL", "info"),
        HTTPPort:        getEnvInt("HTTP_PORT", 8080),
        HTTPSPort:       getEnvInt("HTTPS_PORT", 443),
        UseHTTPS:        getEnvBool("USE_HTTPS", false),
        ShutdownTimeout: 30 * time.Second,
    }

    appCfg := AppConfig{
        DatabaseURL: getEnv("DATABASE_URL", ""),
        APIKey:      getEnv("API_KEY", ""),
    }

    return coreCfg, appCfg, nil
}
```

---

## Validation

Use the `ValidateConfig` hook to validate configuration after loading:

```go
func ValidateConfig(coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) error {
    if appCfg.DatabaseURL == "" {
        return errors.New("DATABASE_URL is required")
    }
    if coreCfg.Env == "production" && !coreCfg.UseHTTPS {
        logger.Warn("HTTPS is disabled in production")
    }
    return nil
}
```

This hook is optional — set to `nil` if validation isn't needed.

---

## Environment-Based Configuration

Common pattern for environment-specific defaults:

```go
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    env := getEnv("ENV", "development")

    coreCfg := &config.CoreConfig{
        Env: env,
    }

    switch env {
    case "production":
        coreCfg.LogLevel = "info"
        coreCfg.UseHTTPS = true
    case "staging":
        coreCfg.LogLevel = "debug"
        coreCfg.UseHTTPS = true
    default: // development
        coreCfg.LogLevel = "debug"
        coreCfg.UseHTTPS = false
    }

    // ... rest of config
    return coreCfg, appCfg, nil
}
```

---

## See Also

- [Lifecycle](./lifecycle.md) — How LoadConfig fits in the lifecycle
- [Core Configuration Reference](../../reference/config-reference.md) — Full CoreConfig field reference

---

[← Back to Development Guide](./README.md)
