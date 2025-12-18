# config

Configuration loading with layered precedence from defaults, files, environment variables, and command-line flags.

## Overview

The `config` package handles WAFFLE's core configuration — the settings shared by all WAFFLE applications like HTTP ports, TLS, CORS, logging, and timeouts. It uses a layered approach where each source can override the previous:

```
defaults → config file → environment variables → command-line flags
```

Application-specific configuration (database URIs, API keys, etc.) is handled separately in your app's `LoadConfig` hook, typically by reading additional environment variables or config sections.

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
| Environment | High | `WAFFLE_HTTP_PORT=8080` |
| Flags | Highest | `--http_port=8080` |

Environment variables are prefixed with `WAFFLE_` and use underscores (e.g., `WAFFLE_USE_HTTPS=true`).

## API

### Load

**Location:** `config.go`

```go
func Load(logger *zap.Logger) (*CoreConfig, error)
```

Loads and validates the core configuration from all sources. Returns an error if validation fails.

**Example:**

```go
func loadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load(logger)
    if err != nil {
        return nil, AppConfig{}, err
    }

    // Load app-specific config from environment
    appCfg := AppConfig{
        MongoURI: os.Getenv("MONGO_URI"),
        MongoDB:  os.Getenv("MONGO_DB"),
    }

    return coreCfg, appCfg, nil
}
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

## Configuration Reference

### Runtime

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `env` | `WAFFLE_ENV` | `"dev"` | Runtime environment |
| `log_level` | `WAFFLE_LOG_LEVEL` | `"debug"` | Log level |

### HTTP

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `http_port` | `WAFFLE_HTTP_PORT` | `8080` | HTTP port |
| `https_port` | `WAFFLE_HTTPS_PORT` | `443` | HTTPS port |
| `use_https` | `WAFFLE_USE_HTTPS` | `false` | Enable HTTPS |
| `max_request_body_bytes` | `WAFFLE_MAX_REQUEST_BODY_BYTES` | `2097152` (2MB) | Max request body size |
| `enable_compression` | `WAFFLE_ENABLE_COMPRESSION` | `true` | Enable gzip compression |

### TLS / Let's Encrypt

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `cert_file` | `WAFFLE_CERT_FILE` | `""` | TLS certificate file (manual) |
| `key_file` | `WAFFLE_KEY_FILE` | `""` | TLS key file (manual) |
| `use_lets_encrypt` | `WAFFLE_USE_LETS_ENCRYPT` | `false` | Use ACME/Let's Encrypt |
| `lets_encrypt_email` | `WAFFLE_LETS_ENCRYPT_EMAIL` | `""` | ACME account email |
| `lets_encrypt_cache_dir` | `WAFFLE_LETS_ENCRYPT_CACHE_DIR` | `"letsencrypt-cache"` | ACME cache directory |
| `domain` | `WAFFLE_DOMAIN` | `""` | Domain for TLS |
| `lets_encrypt_challenge` | `WAFFLE_LETS_ENCRYPT_CHALLENGE` | `"http-01"` | ACME challenge type |
| `route53_hosted_zone_id` | `WAFFLE_ROUTE53_HOSTED_ZONE_ID` | `""` | Route 53 zone (for dns-01) |

### CORS

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `enable_cors` | `WAFFLE_ENABLE_CORS` | `false` | Enable CORS middleware |
| `cors_allowed_origins` | `WAFFLE_CORS_ALLOWED_ORIGINS` | `[]` | JSON array of origins |
| `cors_allowed_methods` | `WAFFLE_CORS_ALLOWED_METHODS` | `[]` | JSON array of methods |
| `cors_allowed_headers` | `WAFFLE_CORS_ALLOWED_HEADERS` | `[]` | JSON array of headers |
| `cors_exposed_headers` | `WAFFLE_CORS_EXPOSED_HEADERS` | `[]` | JSON array of exposed headers |
| `cors_allow_credentials` | `WAFFLE_CORS_ALLOW_CREDENTIALS` | `false` | Allow credentials |
| `cors_max_age` | `WAFFLE_CORS_MAX_AGE` | `0` | Preflight cache seconds |

### Timeouts

| Key | Env Var | Default | Description |
|-----|---------|---------|-------------|
| `db_connect_timeout` | `WAFFLE_DB_CONNECT_TIMEOUT` | `"10s"` | DB connection timeout |
| `index_boot_timeout` | `WAFFLE_INDEX_BOOT_TIMEOUT` | `"120s"` | Schema/index creation timeout |

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

```bash
# .env or shell environment
WAFFLE_ENV=prod
WAFFLE_LOG_LEVEL=info
WAFFLE_USE_HTTPS=true
WAFFLE_USE_LETS_ENCRYPT=true
WAFFLE_LETS_ENCRYPT_EMAIL=admin@example.com
WAFFLE_DOMAIN=example.com
WAFFLE_CORS_ALLOWED_ORIGINS='["https://app.example.com"]'
```

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
