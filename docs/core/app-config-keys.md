# App-Specific Configuration Keys

*How to register application configuration that works uniformly with config files, environment variables, and command-line flags.*

---

## Overview

WAFFLE provides a `LoadWithAppConfig` function that allows applications to register their own configuration keys. These keys work exactly like WAFFLE's core configuration:

- **Config files** — `config.toml`, `config.yaml`, `config.json`
- **Environment variables** — With your app's prefix (e.g., `MYAPP_SESSION_NAME`)
- **Command-line flags** — `--session_name=value`

This ensures a consistent configuration experience across all sources.

---

## Quick Example

```go
// Define your app's configuration keys
var appConfigKeys = []config.AppKey{
    {Name: "mongo_uri", Default: "mongodb://localhost:27017", Desc: "MongoDB connection URI"},
    {Name: "mongo_database", Default: "myapp", Desc: "MongoDB database name"},
    {Name: "session_name", Default: "myapp-session", Desc: "Session cookie name"},
    {Name: "session_key", Default: "change-me", Desc: "Session signing key"},
}

// Load configuration with your app prefix
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, appValues, err := config.LoadWithAppConfig(logger, "MYAPP", appConfigKeys)
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        MongoURI:      appValues.String("mongo_uri"),
        MongoDatabase: appValues.String("mongo_database"),
        SessionName:   appValues.String("session_name"),
        SessionKey:    appValues.String("session_key"),
    }

    return coreCfg, appCfg, nil
}
```

With this setup, users can configure `mongo_uri` via:

```bash
# Config file (config.toml)
mongo_uri = "mongodb://prod-server:27017"

# Environment variable
MYAPP_MONGO_URI=mongodb://prod-server:27017

# Command-line flag
./myapp --mongo_uri=mongodb://prod-server:27017
```

---

## The AppKey Type

Each configuration key is defined using the `AppKey` struct:

```go
type AppKey struct {
    Name    string  // Key name (e.g., "session_name")
    Default any     // Default value if not set elsewhere
    Desc    string  // Description for --help output
}
```

### Supported Default Types

| Type | Example | Notes |
|------|---------|-------|
| `string` | `"localhost"` | Most common |
| `int` | `8080` | Integer values |
| `int64` | `int64(1000000)` | Large integers |
| `bool` | `true` | Boolean flags |
| `[]string` | `[]string{"a", "b"}` | String slices (JSON array on CLI) |

---

## LoadWithAppConfig Function

```go
func LoadWithAppConfig(
    logger *zap.Logger,
    appEnvPrefix string,
    appKeys []AppKey,
) (*CoreConfig, AppConfigValues, error)
```

**Parameters:**

- `logger` — Zap logger for startup messages
- `appEnvPrefix` — Prefix for environment variables (e.g., `"MYAPP"` → `MYAPP_*`)
- `appKeys` — Slice of AppKey definitions

**Returns:**

- `*CoreConfig` — WAFFLE's core configuration (same as `config.Load()`)
- `AppConfigValues` — Map of loaded app configuration values
- `error` — Any loading errors

---

## AppConfigValues Helper Methods

The returned `AppConfigValues` map provides typed accessor methods:

```go
// Get a string value (returns "" if not found or wrong type)
value := appValues.String("mongo_uri")

// Get an int value (returns 0 if not found or wrong type)
port := appValues.Int("port")

// Get an int64 value (returns 0 if not found or wrong type)
maxSize := appValues.Int64("max_size")

// Get a bool value (returns false if not found or wrong type)
enabled := appValues.Bool("feature_enabled")

// Get a string slice (returns nil if not found or wrong type)
hosts := appValues.StringSlice("allowed_hosts")
```

---

## Configuration Precedence

App configuration follows the same precedence as WAFFLE core config (lowest → highest):

1. **Default values** — From `AppKey.Default`
2. **Config files** — `config.toml`, `config.yaml`, `config.json`
3. **Environment variables** — `{PREFIX}_{KEY_NAME}`
4. **Command-line flags** — `--{key_name}=value`

This means command-line flags always win, allowing easy overrides during development or deployment.

---

## Environment Variable Naming

Environment variables are automatically derived from the key name:

1. The `appEnvPrefix` is prepended (e.g., `MYAPP`)
2. The key name is uppercased
3. Hyphens and dots become underscores

**Examples with prefix `MYAPP`:**

| Key Name | Environment Variable |
|----------|---------------------|
| `mongo_uri` | `MYAPP_MONGO_URI` |
| `session_name` | `MYAPP_SESSION_NAME` |
| `max_connections` | `MYAPP_MAX_CONNECTIONS` |

---

## Complete Example: StrataHub

Here's a real-world example from the StrataHub application:

### bootstrap/config.go

```go
package bootstrap

import (
    "fmt"

    "github.com/dalemusser/waffle/config"
    "go.uber.org/zap"
)

// App configuration keys — these work with config files, env vars, and CLI flags.
var appConfigKeys = []config.AppKey{
    {Name: "mongo_uri", Default: "mongodb://localhost:27017", Desc: "MongoDB connection URI"},
    {Name: "mongo_database", Default: "strata_hub", Desc: "MongoDB database name"},
    {Name: "session_key", Default: "dev-only-change-me", Desc: "Session signing key"},
    {Name: "session_name", Default: "stratahub-session", Desc: "Session cookie name"},
    {Name: "session_domain", Default: "", Desc: "Session cookie domain"},
}

func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, appValues, err := config.LoadWithAppConfig(logger, "STRATAHUB", appConfigKeys)
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        MongoURI:      appValues.String("mongo_uri"),
        MongoDatabase: appValues.String("mongo_database"),
        SessionKey:    appValues.String("session_key"),
        SessionName:   appValues.String("session_name"),
        SessionDomain: appValues.String("session_domain"),
    }

    return coreCfg, appCfg, nil
}
```

### bootstrap/appconfig.go

```go
package bootstrap

type AppConfig struct {
    MongoURI      string
    MongoDatabase string
    SessionKey    string
    SessionName   string
    SessionDomain string
}
```

### Configuration Methods

Users can configure StrataHub using any of these methods:

**config.toml:**
```toml
mongo_uri = "mongodb://prod-cluster:27017"
mongo_database = "stratahub_prod"
session_name = "stratahub"
session_domain = ".example.com"
```

**Environment variables:**
```bash
export STRATAHUB_MONGO_URI=mongodb://prod-cluster:27017
export STRATAHUB_MONGO_DATABASE=stratahub_prod
export STRATAHUB_SESSION_KEY=super-secret-key
```

**Command-line flags:**
```bash
./stratahub --mongo_uri=mongodb://localhost:27017 --mongo_database=test_db
```

---

## Security: Automatic Redaction

When logging loaded configuration, WAFFLE automatically redacts values for keys containing:

- `key`
- `secret`
- `password`
- `token`

This prevents accidental exposure of sensitive values in logs.

---

## Backwards Compatibility

The original `config.Load()` function continues to work unchanged:

```go
func Load(logger *zap.Logger) (*CoreConfig, error) {
    core, _, err := LoadWithAppConfig(logger, "", nil)
    return core, err
}
```

Existing applications don't need to change unless they want to use app-specific configuration keys.

---

## When to Use This

Use `LoadWithAppConfig` when you want:

- **Consistent configuration** across all sources (files, env, CLI)
- **Self-documenting flags** with `--help` support
- **Environment variable support** with your app's prefix
- **Type-safe defaults** enforced at registration time

For simpler cases where you only need environment variables, the patterns in [AppConfig Patterns](../guides/patterns/appconfig.md) may be sufficient.

---

## See Also

- [Configuration Overview](./configuration.md) — CoreConfig, AppConfig, and Viper basics
- [AppConfig Patterns](../guides/patterns/appconfig.md) — Manual configuration patterns
- [Architecture](./architecture.md) — How configuration fits in the lifecycle

---

[← Back to Core](./README.md) | [← Back to Documentation Index](../waffle-docs.md)
