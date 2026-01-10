# config

Configuration loading with layered precedence from defaults, files, environment variables, and command-line flags.

## Overview

The `config` package handles WAFFLE's core configuration — the settings shared by all WAFFLE applications like HTTP ports, TLS, CORS, logging, and timeouts. It uses a layered approach where each source can override the previous:

```
defaults → config file → environment variables → command-line flags
```

Application-specific configuration (database URIs, API keys, etc.) can be loaded alongside core config using `LoadWithAppConfig`, which provides a unified environment variable prefix for both core and app settings.

## Import

```go
import "github.com/dalemusser/waffle/config"
```

## Configuration Sources

| Source | Precedence | Example |
|--------|------------|---------|
| Defaults | Lowest | Built into WAFFLE |
| Config file | Low | `config.yaml`, `config.toml`, `config.json` |
| `.env` file | Medium | Loaded via godotenv |
| Environment | High | `MYAPP_HTTP_PORT=8080` |
| Flags | Highest | `--http_port=8080` |

Environment variables use a prefix that you specify when calling `LoadWithAppConfig`. For example, if your app uses prefix `"STRATA"`, then `STRATA_HTTP_PORT`, `STRATA_LOG_LEVEL`, etc. If no prefix is provided (or using `Load()`), the default `WAFFLE_` prefix is used for backward compatibility.

## API

### Load

**Location:** `config.go`

```go
func Load(logger *zap.Logger) (*CoreConfig, error)
```

Loads and validates the core configuration from all sources. Returns an error if validation fails.

**Example:**

```go
func loadConfig(logger *zap.Logger) (*config.CoreConfig, error) {
    coreCfg, err := config.Load(logger)
    if err != nil {
        return nil, err
    }
    return coreCfg, nil
}
```

### LoadWithAppConfig

**Location:** `config.go`

```go
func LoadWithAppConfig(logger *zap.Logger, appEnvPrefix string, appKeys []AppKey) (*CoreConfig, AppConfigValues, error)
```

Loads both core config and app-specific config with a unified environment variable prefix. The `appEnvPrefix` is used for **all** environment variables (both core and app), allowing apps to have a single, consistent prefix.

**Parameters:**
- `logger`: Zap logger for logging config loading messages
- `appEnvPrefix`: Environment variable prefix for all config (e.g., `"STRATA"` → `STRATA_HTTP_PORT`, `STRATA_MONGO_URI`)
- `appKeys`: Slice of `AppKey` defining app-specific configuration keys

If `appEnvPrefix` is empty, core config uses the default `"WAFFLE"` prefix for backward compatibility.

**Example:**

```go
// Define app-specific config keys
var appConfigKeys = []config.AppKey{
    {Name: "mongo_uri", Default: "mongodb://localhost:27017", Desc: "MongoDB connection URI"},
    {Name: "mongo_database", Default: "myapp", Desc: "MongoDB database name"},
    {Name: "session_key", Default: "", Desc: "Session signing key"},
}

func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    // Use "STRATA" prefix for ALL env vars (core + app)
    coreCfg, appValues, err := config.LoadWithAppConfig(logger, "STRATA", appConfigKeys)
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        MongoURI:      appValues.String("mongo_uri"),
        MongoDatabase: appValues.String("mongo_database"),
        SessionKey:    appValues.String("session_key"),
    }

    return coreCfg, appCfg, nil
}

// Now these environment variables all work:
// STRATA_HTTP_PORT=8080       (core config)
// STRATA_LOG_LEVEL=info       (core config)
// STRATA_MONGO_URI=mongodb://... (app config)
// STRATA_SESSION_KEY=secret   (app config)
```

### CoreConfig

**Location:** `config.go`

The main configuration struct containing all WAFFLE core settings.

```go
type CoreConfig struct {
    // Runtime
    Env      string // "dev" | "prod"
    LogLevel string // "debug", "info", "warn", "error"

    // Grouped config
    HTTP HTTPConfig
    TLS  TLSConfig
    CORS CORSConfig

    // Timeouts
    DBConnectTimeout time.Duration
    IndexBootTimeout time.Duration

    // HTTP behavior
    MaxRequestBodyBytes int64
    EnableCompression   bool
}
```

### HTTPConfig

**Location:** `config.go`

```go
type HTTPConfig struct {
    HTTPPort  int  // Default: 8080
    HTTPSPort int  // Default: 443
    UseHTTPS  bool // Default: false
}
```

### TLSConfig

**Location:** `config.go`

```go
type TLSConfig struct {
    CertFile             string // Manual TLS cert path
    KeyFile              string // Manual TLS key path
    UseLetsEncrypt       bool   // Use automatic ACME
    LetsEncryptEmail     string // ACME account email
    LetsEncryptCacheDir  string // Default: "letsencrypt-cache"
    Domain               string // Domain for TLS/ACME
    LetsEncryptChallenge string // "http-01" or "dns-01"
    Route53HostedZoneID  string // For dns-01 with Route 53
}
```

### CORSConfig

**Location:** `config.go`

```go
type CORSConfig struct {
    EnableCORS           bool
    CORSAllowedOrigins   []string
    CORSAllowedMethods   []string
    CORSAllowedHeaders   []string
    CORSExposedHeaders   []string
    CORSAllowCredentials bool
    CORSMaxAge           int
}
```

### CoreConfig.Dump

**Location:** `config.go`

```go
func (c CoreConfig) Dump() string
```

Returns a pretty-printed JSON representation of the config for debugging. Sensitive fields are redacted.

### AppKey

**Location:** `appconfig.go`

```go
type AppKey struct {
    Name    string // Key name (e.g., "mongo_uri")
    Default any    // Default value (string, int, int64, bool, []string)
    Desc    string // Description for --help output
}
```

Defines an application-specific configuration key. Used with `LoadWithAppConfig`.

### AppConfigValues

**Location:** `appconfig.go`

```go
type AppConfigValues map[string]any
```

Holds loaded app configuration values. Provides typed accessor methods:

```go
appCfg.String("mongo_uri")              // Returns string or ""
appCfg.Int("port")                      // Returns int or 0
appCfg.Int64("max_size")                // Returns int64 or 0
appCfg.Bool("debug")                    // Returns bool or false
appCfg.StringSlice("allowed_hosts")     // Returns []string or nil
appCfg.Duration("timeout", 30*time.Second) // Returns duration or default
```

## Configuration Reference

Environment variable names use your app's prefix. For example, with prefix `"STRATA"`:
- `env` → `STRATA_ENV`
- `http_port` → `STRATA_HTTP_PORT`

If using `Load()` without an app prefix, the default `WAFFLE_` prefix is used.

### Runtime

| Key | Env Var Pattern | Default | Description |
|-----|-----------------|---------|-------------|
| `env` | `{PREFIX}_ENV` | `"dev"` | Runtime environment |
| `log_level` | `{PREFIX}_LOG_LEVEL` | `"debug"` | Log level |

### HTTP

| Key | Env Var Pattern | Default | Description |
|-----|-----------------|---------|-------------|
| `http_port` | `{PREFIX}_HTTP_PORT` | `8080` | HTTP port |
| `https_port` | `{PREFIX}_HTTPS_PORT` | `443` | HTTPS port |
| `use_https` | `{PREFIX}_USE_HTTPS` | `false` | Enable HTTPS |
| `max_request_body_bytes` | `{PREFIX}_MAX_REQUEST_BODY_BYTES` | `2097152` (2MB) | Max request body size |
| `enable_compression` | `{PREFIX}_ENABLE_COMPRESSION` | `true` | Enable gzip compression |

### TLS / Let's Encrypt

| Key | Env Var Pattern | Default | Description |
|-----|-----------------|---------|-------------|
| `cert_file` | `{PREFIX}_CERT_FILE` | `""` | TLS certificate file (manual) |
| `key_file` | `{PREFIX}_KEY_FILE` | `""` | TLS key file (manual) |
| `use_lets_encrypt` | `{PREFIX}_USE_LETS_ENCRYPT` | `false` | Use ACME/Let's Encrypt |
| `lets_encrypt_email` | `{PREFIX}_LETS_ENCRYPT_EMAIL` | `""` | ACME account email |
| `lets_encrypt_cache_dir` | `{PREFIX}_LETS_ENCRYPT_CACHE_DIR` | `"letsencrypt-cache"` | ACME cache directory |
| `domain` | `{PREFIX}_DOMAIN` | `""` | Domain for TLS |
| `lets_encrypt_challenge` | `{PREFIX}_LETS_ENCRYPT_CHALLENGE` | `"http-01"` | ACME challenge type |
| `route53_hosted_zone_id` | `{PREFIX}_ROUTE53_HOSTED_ZONE_ID` | `""` | Route 53 zone (for dns-01) |

### CORS

| Key | Env Var Pattern | Default | Description |
|-----|-----------------|---------|-------------|
| `enable_cors` | `{PREFIX}_ENABLE_CORS` | `false` | Enable CORS middleware |
| `cors_allowed_origins` | `{PREFIX}_CORS_ALLOWED_ORIGINS` | `[]` | JSON array of origins |
| `cors_allowed_methods` | `{PREFIX}_CORS_ALLOWED_METHODS` | `[]` | JSON array of methods |
| `cors_allowed_headers` | `{PREFIX}_CORS_ALLOWED_HEADERS` | `[]` | JSON array of headers |
| `cors_exposed_headers` | `{PREFIX}_CORS_EXPOSED_HEADERS` | `[]` | JSON array of exposed headers |
| `cors_allow_credentials` | `{PREFIX}_CORS_ALLOW_CREDENTIALS` | `false` | Allow credentials |
| `cors_max_age` | `{PREFIX}_CORS_MAX_AGE` | `0` | Preflight cache seconds |

### Timeouts

| Key | Env Var Pattern | Default | Description |
|-----|-----------------|---------|-------------|
| `db_connect_timeout` | `{PREFIX}_DB_CONNECT_TIMEOUT` | `"10s"` | DB connection timeout |
| `index_boot_timeout` | `{PREFIX}_INDEX_BOOT_TIMEOUT` | `"120s"` | Schema/index creation timeout |

## Config File Examples

### YAML

```yaml
# config.yaml
env: prod
log_level: info

http_port: 8080
use_https: true
https_port: 443

use_lets_encrypt: true
lets_encrypt_email: admin@example.com
domain: example.com

enable_cors: true
cors_allowed_origins:
  - https://app.example.com
  - https://admin.example.com
cors_allowed_methods:
  - GET
  - POST
  - PUT
  - DELETE
```

### TOML

```toml
# config.toml
env = "prod"
log_level = "info"

http_port = 8080
use_https = true
https_port = 443

use_lets_encrypt = true
lets_encrypt_email = "admin@example.com"
domain = "example.com"
```

### Environment Variables

With a unified prefix (e.g., `"STRATA"`), all env vars use the same prefix:

```bash
# .env or shell environment (using STRATA prefix)
STRATA_ENV=prod
STRATA_LOG_LEVEL=info
STRATA_HTTP_PORT=8080
STRATA_USE_HTTPS=true
STRATA_USE_LETS_ENCRYPT=true
STRATA_LETS_ENCRYPT_EMAIL=admin@example.com
STRATA_DOMAIN=example.com
STRATA_CORS_ALLOWED_ORIGINS='["https://app.example.com"]'

# App-specific config also uses the same prefix
STRATA_MONGO_URI=mongodb://localhost:27017
STRATA_SESSION_KEY=your-secret-key
```

If using `Load()` without an app prefix, use `WAFFLE_` prefix for backward compatibility.

## Validation

`Load()` validates the configuration and returns clear errors for:

- **TLS consistency**: `use_lets_encrypt` requires `use_https`; can't mix Let's Encrypt with manual cert files
- **Required fields**: Let's Encrypt requires `domain` and `lets_encrypt_email`
- **Port sanity**: Ports must be 1-65535; HTTP and HTTPS ports can't be equal
- **CORS requirements**: `enable_cors` requires `cors_allowed_origins` and `cors_allowed_methods`
- **CORS security**: Can't use `*` origin with credentials
- **Timeout validity**: Timeouts must be positive

## Duration Parsing

**Location:** `duration.go`

Timeout fields accept flexible formats:

```yaml
db_connect_timeout: "10s"      # Go duration string
db_connect_timeout: "2m"       # Minutes
db_connect_timeout: 30         # Numeric seconds
db_connect_timeout: "30"       # String seconds
```

## See Also

- [app](../app/app.md) — Application lifecycle (calls `LoadConfig` hook)
- [logging](../logging/logging.md) — Logger configuration
- [middleware](../middleware/middleware.md) — CORS middleware implementation
