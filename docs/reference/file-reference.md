# WAFFLE Framework - Complete File Documentation

This document provides comprehensive documentation for every file in the WAFFLE (Web Application Framework for Flavorful Layered Engineering) project.

**Module:** `github.com/dalemusser/waffle`
**Go Version:** 1.24.0 (toolchain 1.24.1)
**License:** MIT
**Author:** Dale Musser

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Directory Structure](#directory-structure)
3. [Root Files](#root-files)
4. [Core Packages](#core-packages)
   - [app/](#appappgo---application-lifecycle)
   - [config/](#config---configuration-management)
   - [server/](#serverservergo---http-server)
   - [router/](#routerroutergo---router-setup)
   - [logging/](#logging---structured-logging)
   - [metrics/](#metricsmetricsgo---prometheus-metrics)
   - [health/](#healthhealthgo---health-checks)
   - [httputil/](#httputiljsongo---http-utilities)
   - [middleware/](#middleware---http-middleware)
   - [auth/](#auth---authentication)
   - [pprof/](#pprofpprofgo---profiling)
   - [db/](#db---database-utilities)
   - [templates/](#templates---template-engine)
   - [pantry/](#pantry---additional-utilities)
   - [windowsservice/](#windowsservice---windows-service-support)
5. [CLI Tools](#cli-tools)
6. [Internal Packages](#internal-packages)
7. [Dependencies](#dependencies)

---

## Project Overview

WAFFLE is a modern Go web framework designed to help developers build scalable, well-structured web applications. It emphasizes:

- **Clean Architecture** - Separation of concerns with clear package boundaries
- **Composable Middleware** - Flexible request processing pipeline
- **Configuration Flexibility** - Multiple sources (flags, env, files) with clear precedence
- **Observability** - Built-in structured logging, metrics, and health checks
- **Developer Experience** - CLI scaffolding and sensible defaults

---

## Directory Structure

```
waffle/
├── app/                        # Application lifecycle management
├── auth/                       # Authentication modules
│   └── apikey/                 # API key authentication
├── cmd/                        # CLI entry points
│   ├── makewaffle/             # Project scaffolding CLI
│   └── wafflectl/              # Control CLI
├── config/                     # Configuration loading and validation
├── db/                         # Database utilities
│   └── mongo/                  # MongoDB connection helper
├── docs/                       # Documentation
├── health/                     # Health check endpoints
├── httputil/                   # HTTP helper utilities
├── internal/                   # Internal packages
│   └── wafflegen/              # Code generation logic
├── logging/                    # Structured logging
├── metrics/                    # Prometheus metrics
├── middleware/                 # HTTP middleware components
├── pprof/                      # Go profiling endpoints
├── router/                     # Router factory
├── server/                     # HTTP server implementation
├── templates/                  # HTML template engine
├── pantry/                     # Additional utilities
│   ├── apns/                   # Apple Push Notification Service
│   ├── audit/                  # Audit logging
│   ├── auth/                   # Authentication utilities
│   │   ├── apikey/             # API key authentication
│   │   ├── jwt/                # JWT tokens
│   │   └── oauth2/             # OAuth2 providers
│   ├── cache/                  # Caching (memory, Redis)
│   ├── crypto/                 # Encryption, hashing, passwords
│   ├── db/                     # Database connections
│   │   ├── mongo/              # MongoDB
│   │   ├── mysql/              # MySQL
│   │   ├── oracle/             # Oracle
│   │   ├── postgres/           # PostgreSQL
│   │   ├── redis/              # Redis
│   │   └── sqlite/             # SQLite
│   ├── email/                  # Email sending via SMTP
│   ├── errors/                 # Error handling utilities
│   ├── export/                 # Data export utilities
│   ├── fcm/                    # Firebase Cloud Messaging
│   ├── feature/                # Feature flags
│   ├── fileserver/             # Static file serving
│   ├── geo/                    # Geolocation utilities
│   │   ├── ip/                 # IP geolocation
│   │   └── tz/                 # Timezone utilities
│   ├── health/                 # Health check utilities
│   ├── httpnav/                # Navigation helpers
│   ├── i18n/                   # Internationalization
│   ├── jobs/                   # Background job processing
│   ├── mongo/                  # MongoDB utilities
│   ├── mq/                     # Message queues
│   │   ├── rabbitmq/           # RabbitMQ
│   │   └── sqs/                # AWS SQS
│   ├── notify/                 # Notification abstractions
│   ├── pagination/             # Keyset and offset pagination
│   ├── pdf/                    # PDF generation
│   ├── pprof/                  # Profiling utilities
│   ├── ratelimit/              # Rate limiting
│   ├── requestid/              # Request ID utilities
│   ├── retry/                  # Retry logic
│   ├── search/                 # Search indexing
│   ├── session/                # Session management
│   ├── sse/                    # Server-sent events
│   ├── storage/                # File storage (S3, local)
│   ├── templates/              # Template utilities
│   ├── testing/                # Test helpers
│   ├── text/                   # Text processing
│   ├── timeout/                # Context timeout utilities
│   ├── urlutil/                # URL utilities
│   ├── validate/               # Input validation
│   ├── version/                # Version utilities
│   ├── webhook/                # Webhook handling
│   └── websocket/              # WebSocket support
└── windowsservice/             # Windows Service support
```

---

## Root Files

### go.mod - Module Definition

**Location:** `/go.mod`

Defines the Go module and its dependencies.

| Field | Value |
|-------|-------|
| Module | `github.com/dalemusser/waffle` |
| Go Version | 1.24.0 |
| Toolchain | go1.24.1 |

**Primary Dependencies:**
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | v5.2.3 | HTTP router |
| `github.com/go-chi/cors` | v1.2.2 | CORS middleware |
| `github.com/joho/godotenv` | v1.5.1 | .env file loading |
| `github.com/kardianos/service` | v1.2.4 | Windows service support |
| `github.com/prometheus/client_golang` | v1.23.2 | Prometheus metrics |
| `github.com/spf13/pflag` | v1.0.10 | Command-line flags |
| `github.com/spf13/viper` | v1.21.0 | Configuration management |
| `go.mongodb.org/mongo-driver` | v1.17.6 | MongoDB driver |
| `go.uber.org/zap` | v1.27.1 | Structured logging |
| `golang.org/x/crypto` | v0.45.0 | ACME/TLS support |

---

### LICENSE - MIT License

**Location:** `/LICENSE`

The project is licensed under the MIT License, Copyright (c) 2025 Dale Musser.

---

### README.md - Project Introduction

**Location:** `/README.md`

Contains the WAFFLE manifesto, quick start instructions, and links to documentation.

---

## Core Packages

### app/app.go - Application Lifecycle

**Location:** `/app/app.go`
**Package:** `app`

Orchestrates the complete startup and shutdown sequence of a WAFFLE application.

#### Types

##### `Hooks[C, D any]`

Generic struct that defines integration points for applications to hook into WAFFLE's lifecycle.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Application name (for logging/diagnostics) |
| `LoadConfig` | `func(*zap.Logger) (*config.CoreConfig, C, error)` | Loads core and app-specific configuration |
| `ValidateConfig` | `func(*config.CoreConfig, C, *zap.Logger) error` | Optional: validates configuration |
| `ConnectDB` | `func(context.Context, *config.CoreConfig, C, *zap.Logger) (D, error)` | Connects to databases/backends |
| `EnsureSchema` | `func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error` | Optional: creates indexes/schema |
| `Startup` | `func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error` | Optional: one-time initialization |
| `BuildHandler` | `func(*config.CoreConfig, C, D, *zap.Logger) (http.Handler, error)` | Constructs the HTTP handler |
| `OnReady` | `func(*config.CoreConfig, C, D, *zap.Logger)` | Optional: called after server starts listening |
| `Shutdown` | `func(context.Context, *config.CoreConfig, C, D, *zap.Logger) error` | Optional: graceful cleanup |

Type parameters:
- `C` - Application-specific config type
- `D` - Database/dependencies bundle type

#### Functions

##### `Run[C, D any](ctx context.Context, hooks Hooks[C, D]) error`

Main entry point that executes the WAFFLE startup sequence:

1. Bootstrap logger for early startup
2. Load core + app config via `hooks.LoadConfig`
3. Validate config via `hooks.ValidateConfig` (if provided)
4. Build final logger based on config
5. Wire shutdown signals (SIGINT/SIGTERM) to context
6. Register default Prometheus metrics
7. Connect to databases via `hooks.ConnectDB`
8. Ensure schema/indexes via `hooks.EnsureSchema` (if provided)
9. Run startup hook via `hooks.Startup` (if provided)
10. Build HTTP handler via `hooks.BuildHandler`
11. Call `hooks.OnReady` (if provided) to signal readiness
12. Start HTTP(S) server and block until shutdown
13. Run shutdown hook via `hooks.Shutdown` (if provided)

**Example Usage:**
```go
func main() {
    if err := app.Run(context.Background(), bootstrap.Hooks); err != nil {
        log.Fatal(err)
    }
}
```

---

### config/ - Configuration Management

#### config/config.go - Core Configuration

**Location:** `/config/config.go`
**Package:** `config`

Handles configuration loading from multiple sources with proper precedence.

#### Types

##### `HTTPConfig`

Groups HTTP/HTTPS port, protocol, and server timeout settings.

| Field | Type | Mapstructure | Description |
|-------|------|--------------|-------------|
| `HTTPPort` | `int` | `http_port` | HTTP port (default: 8080) |
| `HTTPSPort` | `int` | `https_port` | HTTPS port (default: 443) |
| `UseHTTPS` | `bool` | `use_https` | Enable HTTPS |
| `ReadTimeout` | `time.Duration` | `read_timeout` | HTTP server read timeout (default: 15s) |
| `ReadHeaderTimeout` | `time.Duration` | `read_header_timeout` | HTTP server read header timeout (default: 10s) |
| `WriteTimeout` | `time.Duration` | `write_timeout` | HTTP server write timeout (default: 60s) |
| `IdleTimeout` | `time.Duration` | `idle_timeout` | HTTP server idle timeout (default: 120s) |
| `ShutdownTimeout` | `time.Duration` | `shutdown_timeout` | Graceful shutdown timeout (default: 15s) |

##### `TLSConfig`

Groups all TLS/ACME-related settings.

| Field | Type | Mapstructure | Description |
|-------|------|--------------|-------------|
| `CertFile` | `string` | `cert_file` | TLS certificate file (manual TLS) |
| `KeyFile` | `string` | `key_file` | TLS key file (manual TLS) |
| `UseLetsEncrypt` | `bool` | `use_lets_encrypt` | Use Let's Encrypt |
| `LetsEncryptEmail` | `string` | `lets_encrypt_email` | ACME account email |
| `LetsEncryptCacheDir` | `string` | `lets_encrypt_cache_dir` | ACME cache directory |
| `Domain` | `string` | `domain` | Domain for TLS/ACME |
| `LetsEncryptChallenge` | `string` | `lets_encrypt_challenge` | Challenge type: `http-01` or `dns-01` |
| `Route53HostedZoneID` | `string` | `route53_hosted_zone_id` | Route53 zone ID (for dns-01) |
| `ACMEDirectoryURL` | `string` | `acme_directory_url` | ACME directory URL (auto-detected based on env) |

##### `CORSConfig`

Groups all CORS behavior settings.

| Field | Type | Mapstructure | Description |
|-------|------|--------------|-------------|
| `EnableCORS` | `bool` | `enable_cors` | Enable CORS middleware |
| `CORSAllowedOrigins` | `[]string` | `cors_allowed_origins` | Allowed origins |
| `CORSAllowedMethods` | `[]string` | `cors_allowed_methods` | Allowed HTTP methods |
| `CORSAllowedHeaders` | `[]string` | `cors_allowed_headers` | Allowed headers |
| `CORSExposedHeaders` | `[]string` | `cors_exposed_headers` | Exposed headers |
| `CORSAllowCredentials` | `bool` | `cors_allow_credentials` | Allow credentials |
| `CORSMaxAge` | `int` | `cors_max_age` | Preflight cache duration (seconds) |

##### `CoreConfig`

Main configuration struct holding all WAFFLE-level settings.

| Field | Type | Description |
|-------|------|-------------|
| `Env` | `string` | Runtime environment: `dev` or `prod` |
| `LogLevel` | `string` | Log level: debug, info, warn, error |
| `HTTP` | `HTTPConfig` | HTTP/HTTPS settings (embedded) |
| `TLS` | `TLSConfig` | TLS/ACME settings (embedded) |
| `CORS` | `CORSConfig` | CORS settings (embedded) |
| `DBConnectTimeout` | `time.Duration` | Database connection timeout |
| `IndexBootTimeout` | `time.Duration` | Index creation timeout |
| `MaxRequestBodyBytes` | `int64` | Maximum request body size |
| `EnableCompression` | `bool` | Enable HTTP compression |
| `CompressionLevel` | `int` | Compression level 1-9 (default: 5) |

#### Functions

##### `Load(logger *zap.Logger) (*CoreConfig, error)`

Merges configuration from multiple sources with this precedence (highest wins):
1. Command-line flags (explicit only)
2. Environment variables (`WAFFLE_*` prefix)
3. Config files (`config.yaml`, `config.yml`, `config.json`, `config.toml`)
4. `.env` file (loaded via godotenv)
5. Built-in defaults

Also performs comprehensive validation for:
- TLS/ACME consistency
- Port validity (1-65535)
- CORS configuration sanity
- Positive timeout values

##### `(CoreConfig) Dump() string`

Returns a pretty-printed, redacted JSON string of the config for debugging.

#### Environment Variables

All settings can be configured via environment variables with the `WAFFLE_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `WAFFLE_ENV` | `dev` | Runtime environment |
| `WAFFLE_LOG_LEVEL` | `debug` | Log level |
| `WAFFLE_HTTP_PORT` | `8080` | HTTP port |
| `WAFFLE_HTTPS_PORT` | `443` | HTTPS port |
| `WAFFLE_USE_HTTPS` | `false` | Enable HTTPS |
| `WAFFLE_USE_LETS_ENCRYPT` | `false` | Use Let's Encrypt |
| `WAFFLE_DOMAIN` | `""` | Domain for TLS |
| `WAFFLE_LETS_ENCRYPT_EMAIL` | `""` | ACME email |
| `WAFFLE_LETS_ENCRYPT_CACHE_DIR` | `letsencrypt-cache` | ACME cache |
| `WAFFLE_LETS_ENCRYPT_CHALLENGE` | `http-01` | ACME challenge type |
| `WAFFLE_ROUTE53_HOSTED_ZONE_ID` | `""` | Route53 zone ID |
| `WAFFLE_CERT_FILE` | `""` | TLS cert file |
| `WAFFLE_KEY_FILE` | `""` | TLS key file |
| `WAFFLE_ENABLE_CORS` | `false` | Enable CORS |
| `WAFFLE_CORS_ALLOWED_ORIGINS` | `[]` | JSON array of origins |
| `WAFFLE_CORS_ALLOWED_METHODS` | `[]` | JSON array of methods |
| `WAFFLE_CORS_ALLOWED_HEADERS` | `[]` | JSON array of headers |
| `WAFFLE_CORS_EXPOSED_HEADERS` | `[]` | JSON array of headers |
| `WAFFLE_CORS_ALLOW_CREDENTIALS` | `false` | Allow credentials |
| `WAFFLE_CORS_MAX_AGE` | `0` | Preflight cache seconds |
| `WAFFLE_DB_CONNECT_TIMEOUT` | `10s` | DB connection timeout |
| `WAFFLE_INDEX_BOOT_TIMEOUT` | `120s` | Index creation timeout |
| `WAFFLE_MAX_REQUEST_BODY_BYTES` | `2097152` (2MB) | Max request body |
| `WAFFLE_ENABLE_COMPRESSION` | `true` | HTTP compression |
| `WAFFLE_COMPRESSION_LEVEL` | `5` | Compression level (1-9) |
| `WAFFLE_READ_TIMEOUT` | `15s` | HTTP server read timeout |
| `WAFFLE_READ_HEADER_TIMEOUT` | `10s` | HTTP server read header timeout |
| `WAFFLE_WRITE_TIMEOUT` | `60s` | HTTP server write timeout |
| `WAFFLE_IDLE_TIMEOUT` | `120s` | HTTP server idle timeout |
| `WAFFLE_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown timeout |
| `WAFFLE_ACME_DIRECTORY_URL` | `""` | ACME directory URL (auto-detected) |

---

#### config/duration.go - Duration Parsing

**Location:** `/config/duration.go`
**Package:** `config`

Provides flexible duration parsing from multiple input formats.

#### Functions

##### `parseDurationFlexible(raw interface{}, def time.Duration) (time.Duration, error)`

Parses duration values from various formats:

| Input Type | Example | Interpretation |
|------------|---------|----------------|
| `time.Duration` | `90 * time.Second` | Direct value |
| `string` | `"90s"`, `"2m"`, `"1h"` | Go duration format |
| `string` | `"120"` | Numeric seconds |
| `int`, `int32`, `int64` | `120` | Seconds |
| `float64` | `120.5` | Seconds (fractional) |

Returns the default value on empty/nil input; returns error on invalid strings.

---

### server/server.go - HTTP Server

**Location:** `/server/server.go`
**Package:** `server`

Implements HTTP/HTTPS server with TLS, Let's Encrypt, and graceful shutdown support.

#### Functions

##### `WithShutdownSignals(parent context.Context, logger *zap.Logger) (context.Context, context.CancelFunc)`

Returns a context that is canceled when the process receives `SIGINT` or `SIGTERM`. Use this to tie OS signals into context cancellation.

##### `ListenAndServeWithContext(ctx context.Context, cfg *config.CoreConfig, handler http.Handler, logger *zap.Logger) error`

Starts an HTTP or HTTPS server and blocks until the context is canceled or a terminal error occurs.

**Operating Modes:**

| Mode | Condition | Behavior |
|------|-----------|----------|
| HTTP only | `UseHTTPS=false` | Listens on HTTP port only |
| HTTPS (Let's Encrypt) | `UseHTTPS=true`, `UseLetsEncrypt=true` | Port 80 for ACME/redirect, port 443 for HTTPS |
| HTTPS (Manual TLS) | `UseHTTPS=true`, `UseLetsEncrypt=false` | Port 80 for redirect, port 443 for HTTPS |

**Server Timeouts (configurable via CoreConfig.HTTP):**
| Timeout | Config Key | Default |
|---------|-----------|---------|
| ReadTimeout | `read_timeout` | 15s |
| ReadHeaderTimeout | `read_header_timeout` | 10s |
| WriteTimeout | `write_timeout` | 60s |
| IdleTimeout | `idle_timeout` | 120s |
| ShutdownTimeout | `shutdown_timeout` | 15s |

#### Internal Functions

| Function | Description |
|----------|-------------|
| `servePrimary()` | Runs the primary server listener |
| `serveAuxiliary()` | Runs the auxiliary server (ACME/redirect) |
| `shutdownAux()` | Gracefully shuts down the auxiliary server |
| `httpRedirectHandler()` | Returns a handler that redirects HTTP to HTTPS |
| `waitForCert()` | Blocks until Let's Encrypt certificate is ready |

---

### router/router.go - Router Setup

**Location:** `/router/router.go`
**Package:** `router`

Creates a chi.Router pre-wired with WAFFLE's standard middleware stack.

#### Functions

##### `New(coreCfg *config.CoreConfig, logger *zap.Logger) chi.Router`

Creates a new router with the following middleware applied in order:

1. **RequestID** - Generates unique request correlation IDs
2. **RealIP** - Extracts real client IP from proxy headers
3. **Recoverer** - Recovers from panics, logs with stack trace, returns 500
4. **LimitBodySize** - Enforces `MaxRequestBodyBytes` limit
5. **HTTPMetrics** - Records request duration for Prometheus
6. **RequestLogger** - Logs request details (method, path, status, latency)
7. **NotFound/MethodNotAllowed** - JSON error handlers

**Note:** CORS middleware is not applied here; it should be added at the app level.

---

### logging/ - Structured Logging

#### logging/logging.go - Logger Initialization

**Location:** `/logging/logging.go`
**Package:** `logging`

Creates and configures zap loggers for different stages of application lifecycle.

#### Functions

##### `BootstrapLogger() *zap.Logger`

Returns a development-friendly logger for early startup (before config loads).

| Property | Value |
|----------|-------|
| Level | Info |
| Output | stderr |
| Time Format | ISO8601 |
| Mode | Development (colorized) |

##### `BuildLogger(level, env string) (*zap.Logger, error)`

Constructs the final logger based on configuration.

| Environment | Encoding | Features |
|-------------|----------|----------|
| `prod` | JSON | Production optimized |
| Other | Console | Colorized development output |

Both use RFC-3339 (ISO8601) timestamps.

##### `MustBuildLogger(level, env string) *zap.Logger`

Like `BuildLogger` but panics on failure. Intended for use in `main()`.

---

#### logging/requestmw.go - Request Logging Middleware

**Location:** `/logging/requestmw.go`
**Package:** `logging`

Logs HTTP requests with comprehensive information.

#### Functions

##### `RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler`

Returns middleware that logs each request with:

| Field | Description |
|-------|-------------|
| `method` | HTTP method |
| `path` | URL path |
| `host` | Request host |
| `scheme` | http or https |
| `proto` | HTTP protocol version |
| `status` | Response status code |
| `bytes` | Response body size |
| `remote_ip` | Client IP address |
| `user_agent` | User-Agent header |
| `referer` | Referer header |
| `latency` | Request duration |
| `request_id` | Correlation ID |

---

#### logging/recovermw.go - Panic Recovery Middleware

**Location:** `/logging/recovermw.go`
**Package:** `logging`

Recovers from panics and logs them with full stack traces.

#### Functions

##### `Recoverer(logger *zap.Logger) func(http.Handler) http.Handler`

Returns middleware that:
1. Recovers from panics
2. Logs panic value and full stack trace
3. Returns HTTP 500 "internal server error"

Logged fields: `panic_value`, `stacktrace`, `method`, `path`, `remote_ip`

---

### metrics/metrics.go - Prometheus Metrics

**Location:** `/metrics/metrics.go`
**Package:** `metrics`

Collects HTTP metrics for monitoring with Prometheus.

#### Metrics

##### `http_request_duration_seconds`

Histogram of HTTP request durations.

| Property | Value |
|----------|-------|
| Type | Histogram |
| Labels | `path`, `method`, `status` |
| Buckets | 0.01, 0.1, 0.3, 1.2, 5 seconds |

#### Functions

##### `RegisterDefault(logger *zap.Logger)`

Registers the default collectors with Prometheus:
- Go runtime metrics (goroutines, memory, GC)
- Process metrics (CPU, file descriptors)
- HTTP request duration histogram

Safe to call multiple times (handles `AlreadyRegisteredError`).

##### `HTTPMetrics(next http.Handler) http.Handler`

Middleware that records request duration into the histogram.

##### `Handler() http.Handler`

Returns the Prometheus metrics exposition handler. Mount at `/metrics`.

---

### health/health.go - Health Checks

**Location:** `/health/health.go`
**Package:** `health`

Provides liveness and readiness probe support for Kubernetes/load balancers.

#### Types

##### `Check`

```go
type Check func(ctx context.Context) error
```

Function type for health checks. Returns `nil` if healthy, error otherwise.

##### `Response`

```go
type Response struct {
    Status string            `json:"status"`
    Checks map[string]string `json:"checks,omitempty"`
}
```

JSON structure returned by health endpoints.

#### Functions

##### `Handler(checks map[string]Check, logger *zap.Logger) http.Handler`

Creates a health check handler.

**Response Scenarios:**

| Scenario | Status Code | Response |
|----------|-------------|----------|
| No checks defined | 200 | `{"status": "ok"}` |
| All checks pass | 200 | `{"status": "ok", "checks": {...}}` |
| Any check fails | 503 | `{"status": "error", "checks": {...}}` |

##### `Mount(r chi.Router, checks map[string]Check, logger *zap.Logger)`

Attaches the health handler at `/health`.

##### `MountAt(r chi.Router, path string, checks map[string]Check, logger *zap.Logger)`

Attaches the health handler at a custom path (e.g., `/ready`, `/live`).

**Example:**
```go
checks := map[string]health.Check{
    "db": func(ctx context.Context) error {
        return client.Ping(ctx, readpref.Primary())
    },
}
health.Mount(r, checks, logger)
```

---

### httputil/json.go - HTTP Utilities

**Location:** `/httputil/json.go`
**Package:** `httputil`

Standardized JSON response handling utilities.

#### Types

##### `ErrorResponse`

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}
```

Standard JSON error envelope.

#### Functions

##### `WriteJSON(w http.ResponseWriter, status int, v any)`

Writes a JSON response with the given status code. Sets `Content-Type: application/json`.

##### `JSONError(w http.ResponseWriter, status int, code, message string)`

Writes a structured JSON error with an error code and message.

##### `JSONErrorSimple(w http.ResponseWriter, status int, message string)`

Shorthand for errors where the message itself is the code (omits `message` field).

---

### middleware/ - HTTP Middleware

#### middleware/notfound.go - 404/405 Handlers

**Location:** `/middleware/notfound.go`
**Package:** `middleware`

JSON responses for not found and method not allowed errors.

#### Functions

##### `NotFoundHandler(logger *zap.Logger) http.HandlerFunc`

Returns a handler that:
1. Logs the 404 event
2. Returns JSON: `{"error": "not_found", "message": "The requested resource was not found"}`

##### `MethodNotAllowedHandler(logger *zap.Logger) http.HandlerFunc`

Returns a handler that:
1. Logs the 405 event
2. Returns JSON: `{"error": "method_not_allowed", "message": "..."}`

---

#### middleware/sizelimit.go - Request Size Limiting

**Location:** `/middleware/sizelimit.go`
**Package:** `middleware`

Enforces maximum request body size.

#### Functions

##### `LimitBodySize(maxBytes int64) func(http.Handler) http.Handler`

Returns middleware that limits request body size.

| maxBytes | Behavior |
|----------|----------|
| `<= 0` | No limit (identity middleware) |
| `> 0` | Uses `http.MaxBytesReader` to enforce limit |

Should be applied early in the middleware chain.

---

#### middleware/cors.go - CORS Support

**Location:** `/middleware/cors.go`
**Package:** `middleware`

Configurable CORS middleware from configuration.

#### Functions

##### `CORSFromConfig(coreCfg *config.CoreConfig) func(http.Handler) http.Handler`

Creates CORS middleware based on `CoreConfig.CORS` settings.

- Returns identity middleware if CORS is disabled
- Safe to unconditionally apply
- Maps all `CORSConfig` fields to `cors.Options`

---

#### middleware/contenttype.go - Content-Type Validation

**Location:** `/middleware/contenttype.go`
**Package:** `middleware`

Enforces JSON Content-Type for specific routes.

#### Functions

##### `RequireJSON() func(http.Handler) http.Handler`

Returns middleware that:
1. Checks for `application/json` or `*+json` Content-Type
2. Strips parameters (e.g., `; charset=utf-8`)
3. Returns 415 Unsupported Media Type if missing/wrong

---

### auth/ - Authentication

#### auth/apikey/apikey.go - API Key Authentication

**Location:** `/auth/apikey/apikey.go`
**Package:** `apikey`

Static API key authentication with multiple lookup methods.

#### Types

##### `Options`

```go
type Options struct {
    Realm      string  // WWW-Authenticate realm
    CookieName string  // Optional: enables cookie-based auth
}
```

#### Functions

##### `Require(expected string, opts Options, logger *zap.Logger) func(http.Handler) http.Handler`

Returns middleware that enforces API key validation.

**Key Lookup Order:**
1. `Authorization: Bearer <token>` header
2. `X-API-Key` header
3. `api_key` query parameter
4. Cookie (if `CookieName` configured)

**On Success:**
- Sets HttpOnly, Secure, SameSite=Lax cookie (if configured)
- Passes request to next handler

**On Failure:**
- Logs unauthorized attempt
- Sets `WWW-Authenticate: Bearer realm="..."` header
- Returns 401 Unauthorized

---

### pprof/pprof.go - Profiling

**Location:** `/pprof/pprof.go`
**Package:** `pprof`

Mounts Go's built-in pprof profiling endpoints.

#### Functions

##### `Mount(r chi.Router)`

Attaches pprof handlers under `/debug/pprof`:

| Path | Handler |
|------|---------|
| `/debug/pprof/` | Index page |
| `/debug/pprof/cmdline` | Command line |
| `/debug/pprof/profile` | CPU profile |
| `/debug/pprof/symbol` | Symbol lookup |
| `/debug/pprof/trace` | Execution trace |
| `/debug/pprof/{name}` | Named profiles (heap, goroutine, allocs, block, etc.) |

**Security Note:** Should be mounted behind authentication middleware.

**Example:**
```go
r.Group(func(r chi.Router) {
    r.Use(apikey.Require(adminKey, opts, logger))
    pprof.Mount(r)
})
```

---

### db/ - Database Utilities

#### db/mongo/mongo.go - MongoDB Connection

**Location:** `/db/mongo/mongo.go`
**Package:** `mongo`

Simplified MongoDB connection setup.

#### Functions

##### `Connect(uri string, timeout time.Duration) (*mongo.Client, error)`

Opens a MongoDB/DocumentDB connection:
1. Connects using the provided URI and timeout
2. Performs a Ping to verify connection
3. Returns the client (caller must call `Disconnect()` when done)

---

### templates/ - Template Engine

WAFFLE provides an advanced HTML template engine supporting shared layouts, per-page isolation, and HTMX integration.

#### templates/engine.go - Core Template Engine

**Location:** `/templates/engine.go`
**Package:** `templates`

Manages compiled templates with shared layouts and per-page clones.

#### Types

##### `Engine`

```go
type Engine struct {
    mu      sync.RWMutex
    funcs   template.FuncMap
    base    *template.Template            // compiled from "shared"
    byName  map[string]*template.Template // templateName -> compiled set
    devMode bool
    Logger  *zap.Logger
}
```

Thread-safe template engine with:
- Shared base template for common layout
- Per-page clones for isolation
- Template name collision prevention

#### Functions

##### `New(dev bool) *Engine`

Creates a new Engine. `dev=true` can enable hot-reload (future feature).

##### `(e *Engine) Boot(logger *zap.Logger) error`

Compiles all registered template Sets:
1. Parses "shared" set first (common layout)
2. For each feature set, creates one clone per page file
3. Rewrites non-target files' `define "content"` to unique names
4. Indexes only template names owned by target file

##### `(e *Engine) Render(w Writer, r Request, name string, data any) error`

Executes a full page template by name.

##### `(e *Engine) RenderSnippet(w Writer, name string, data any) error`

Executes a partial template by name (for HTMX fragments).

##### `(e *Engine) RenderContent(w Writer, entry string, data any) error`

Executes just the "content" block of a page template.

---

#### templates/adapter.go - Template Rendering API

**Location:** `/templates/adapter.go`
**Package:** `templates`

Package-level helper functions for template rendering.

#### Functions

##### `UseEngine(e *Engine, l *zap.Logger)`

Installs the engine and logger for package-level render functions.

##### `Render(w http.ResponseWriter, r *http.Request, name string, data any)`

Executes a full page with layout.

##### `RenderSnippet(w http.ResponseWriter, name string, data any)`

Renders a partial template (e.g., table fragment).

##### `RenderAutoMap(w http.ResponseWriter, r *http.Request, page string, targets map[string]string, data any)`

HTMX-aware rendering that:
1. Detects HTMX requests via `HX-Request` header
2. Checks `HX-Target` for target element
3. Maps targets to snippets via the `targets` map
4. Falls back to content block if target is "content"
5. Renders full page for non-HTMX requests

##### `RenderAuto(w http.ResponseWriter, r *http.Request, page, tableSnippet, targetID string, data any)`

Convenience wrapper for single-table swap scenarios.

---

#### templates/views.go - Template Set Registry

**Location:** `/templates/views.go`
**Package:** `templates`

Registry for template sets from feature modules.

#### Types

##### `Set`

```go
type Set struct {
    Name     string   // For logging/debugging
    FS       fs.FS    // Embedded filesystem
    Patterns []string // Glob patterns to load
}
```

#### Functions

##### `Register(s Set)`

Called from feature packages' `init()` to register a template set.

##### `All() []Set`

Returns all registered template sets.

##### `Reset()`

Clears the registry (for testing).

---

#### templates/funcs.go - Template Helpers

**Location:** `/templates/funcs.go`
**Package:** `templates`

Template functions available to all templates.

#### Functions

##### `Funcs() template.FuncMap`

Returns the function map:

| Function | Description | Example |
|----------|-------------|---------|
| `urlquery` | URL query encoding | `{{ "a b" \| urlquery }}` → `"a+b"` |
| `safeHTML` | Mark string as safe HTML | `{{ .HTML \| safeHTML }}` |
| `lower` | Lowercase string | `{{ "ABC" \| lower }}` → `"abc"` |
| `upper` | Uppercase string | `{{ "abc" \| upper }}` → `"ABC"` |
| `join` | Join string slice | `{{ .Slice \| join "," }}` |
| `printf` | String formatting | `{{ printf "%d items" .Count }}` |

---

### pantry/ - Additional Utilities

The WAFFLE pantry contains optional, reusable utilities.

#### pantry/email/email.go - Email Sending

**Location:** `/pantry/email/email.go`
**Package:** `email`

SMTP email sending using go-mail.

| Feature | Description |
|---------|-------------|
| `NewSender(cfg Config)` | Creates a new email sender |
| `Send(ctx, msg)` | Sends a fully customized message |
| `SendSimple(ctx, to, subject, body)` | Sends a plain text email |
| `SendHTML(ctx, to, subject, text, html)` | Sends an HTML email with plain text fallback |

---

#### pantry/fileserver/fileserver.go - Static File Serving

**Location:** `/pantry/fileserver/fileserver.go`
**Package:** `fileserver`

HTTP handler for serving static files with pre-compressed file support (gzip, Brotli).

| Function | Description |
|----------|-------------|
| `Handler(urlPrefix, rootDir)` | Returns a handler serving files from rootDir |
| `HandlerWithOptions(urlPrefix, rootDir, opts)` | Handler with cache control options |

---

#### pantry/mongo/ - MongoDB Utilities

**Location:** `/pantry/mongo/`
**Package:** `mongo`

MongoDB helper utilities including cursor iteration, keyset pagination, and error handling.

---

#### pantry/text/fold.go - Text Processing

**Location:** `/pantry/text/fold.go`
**Package:** `text`

Unicode-aware text folding for search and comparison.

| Function | Description |
|----------|-------------|
| `Fold(s)` | Lowercases and strips combining diacritics |
| `FoldTokens(s)` | Folds and splits on whitespace |
| `PrefixRange(q)` | Returns half-open range for prefix queries |

---

#### pantry/validate/email.go - Input Validation

**Location:** `/pantry/validate/email.go`
**Package:** `validate`

Input validation helpers.

| Function | Description |
|----------|-------------|
| `SimpleEmailValid(email)` | Basic email format validation |

---

### windowsservice/ - Windows Service Support

#### windowsservice/programwindows.go - Windows Service

**Location:** `/windowsservice/programwindows.go`
**Package:** `windowsservice`
**Build Tag:** `//go:build windows`

Wraps WAFFLE app to run as a Windows Service.

#### Types

##### `Program[C, D any]`

Implements `service.Service` interface for Windows SCM integration.

| Field | Type | Description |
|-------|------|-------------|
| `Hooks` | `app.Hooks[C, D]` | WAFFLE application hooks |
| `cancel` | `func()` | Context cancel function |

#### Methods

##### `(p *Program) Start(s service.Service) error`

Called by SCM when service starts. Runs the WAFFLE app in a goroutine.

##### `(p *Program) Stop(s service.Service) error`

Called by SCM when service stops. Cancels context for graceful shutdown.

---

#### windowsservice/programstub.go - Non-Windows Stub

**Location:** `/windowsservice/programstub.go`
**Package:** `windowsservice`
**Build Tag:** `//go:build !windows`

Placeholder for non-Windows platforms.

#### Variables

##### `ErrNotWindows`

```go
var ErrNotWindows = errors.New("windowsservice: not supported on this platform")
```

---

## CLI Tools

### cmd/makewaffle/main.go - Project Scaffolding CLI

**Location:** `/cmd/makewaffle/main.go`
**Package:** `main`

Entry point for the `makewaffle` CLI tool.

```go
func main() {
    os.Exit(wafflegen.Run("makewaffle", os.Args[1:]))
}
```

**Installation:**
```bash
go install github.com/dalemusser/waffle/cmd/makewaffle@latest
```

**Usage:**
```bash
makewaffle new <appname> --module <module-path> [--waffle-version <version>] [--go-version <version>]
```

---

### cmd/wafflectl/main.go - Control CLI

**Location:** `/cmd/wafflectl/main.go`
**Package:** `main`

Entry point for the `wafflectl` CLI tool (same functionality as makewaffle).

```go
func main() {
    os.Exit(wafflegen.Run("wafflectl", os.Args[1:]))
}
```

---

## Internal Packages

### internal/wafflegen/wafflegen.go - Code Generator

**Location:** `/internal/wafflegen/wafflegen.go`
**Package:** `wafflegen`

Code generator for creating new WAFFLE projects.

#### Functions

##### `Run(binName string, args []string) int`

Main entry point for CLI tools. Returns exit code.

**Commands:**
| Command | Description |
|---------|-------------|
| `new <appname>` | Create a new WAFFLE project |

##### `newCmd(binName string, args []string) int`

Handles the `new` command with flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--module` | (required) | Go module path |
| `--waffle-version` | `""` | WAFFLE version to require |
| `--go-version` | `1.21` | Go version for go.mod |

##### `scaffoldApp(appName, module, waffleVersion, goVersion string) error`

Creates the project structure:

```
<appname>/
├── go.mod
├── cmd/<appname>/
│   └── main.go
└── internal/
    ├── app/
    │   ├── bootstrap/
    │   │   ├── hooks.go
    │   │   ├── config.go
    │   │   ├── appconfig.go
    │   │   ├── db.go
    │   │   ├── dbdeps.go
    │   │   ├── startup.go
    │   │   ├── routes.go
    │   │   └── shutdown.go
    │   ├── features/
    │   │   └── README.md
    │   ├── resources/
    │   │   └── README.md
    │   ├── system/
    │   │   └── README.md
    │   ├── store/
    │   │   └── README.md
    │   └── policy/
    │       └── README.md
    └── domain/
        └── models/
            └── README.md
```

##### `validateAppName(name string) error`

Validates app name:
- Only letters, digits, underscore allowed
- Cannot start with digit
- Cannot be empty

---

## Dependencies

### Primary Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/go-chi/chi/v5` | Lightweight, idiomatic HTTP router |
| `github.com/go-chi/cors` | CORS middleware for chi |
| `github.com/joho/godotenv` | Loads environment variables from .env |
| `github.com/kardianos/service` | Cross-platform service manager |
| `github.com/prometheus/client_golang` | Prometheus instrumentation library |
| `github.com/spf13/pflag` | POSIX-compliant command-line flags |
| `github.com/spf13/viper` | Complete configuration solution |
| `go.mongodb.org/mongo-driver` | Official MongoDB Go driver |
| `go.uber.org/zap` | Fast, structured, leveled logging |
| `golang.org/x/crypto` | Cryptographic packages (ACME, TLS) |

### Transitive Dependencies

Notable indirect dependencies include:
- `github.com/fsnotify/fsnotify` - File system notifications
- `github.com/pelletier/go-toml/v2` - TOML parsing
- `github.com/prometheus/client_model` - Prometheus data model
- `github.com/spf13/afero` - Filesystem abstraction
- `go.yaml.in/yaml/v3` - YAML parsing

---

## Quick Reference

### Creating a New Project

```bash
# Install CLI
go install github.com/dalemusser/waffle/cmd/makewaffle@latest

# Create project
makewaffle new myapp --module github.com/user/myapp

# Run
cd myapp
go mod tidy
go run ./cmd/myapp
```

### Implementing Hooks

```go
var Hooks = app.Hooks[AppConfig, DBDeps]{
    Name:           "myapp",
    LoadConfig:     LoadConfig,
    ValidateConfig: ValidateConfig,  // optional
    ConnectDB:      ConnectDB,
    EnsureSchema:   EnsureSchema,    // optional
    Startup:        Startup,         // optional
    BuildHandler:   BuildHandler,
    OnReady:        OnReady,         // optional
    Shutdown:       Shutdown,        // optional
}
```

### Configuration via Environment

```bash
export WAFFLE_ENV=prod
export WAFFLE_LOG_LEVEL=info
export WAFFLE_HTTP_PORT=8080
export WAFFLE_SHUTDOWN_TIMEOUT=30s
export WAFFLE_ENABLE_CORS=true
export WAFFLE_CORS_ALLOWED_ORIGINS='["https://example.com"]'
```

### Adding Health Checks

```go
checks := map[string]health.Check{
    "db": func(ctx context.Context) error {
        return db.Ping(ctx)
    },
    "redis": func(ctx context.Context) error {
        return redis.Ping(ctx).Err()
    },
}
health.Mount(r, checks, logger)
```

### Protecting Routes

```go
r.Group(func(r chi.Router) {
    r.Use(apikey.Require(adminKey, apikey.Options{
        Realm: "admin",
        CookieName: "admin_token",
    }, logger))
    pprof.Mount(r)
    r.Get("/admin", adminHandler)
})
```

---

## See Also

- [Operational Endpoints Guide](../core/operational-endpoints.md) — Health, metrics, and pprof
- [Routes & Middleware Guide](../core/routing.md) — Routing patterns
- [Development Guide](../guides/development/README.md) — Framework overview
- [WAFFLE Quickstart Guide](../guides/getting-started/quickstart.md) — Quick overview
- [makewaffle CLI Guide](../guides/getting-started/makewaffle.md) — Scaffold new applications

---

*This is generated reference documentation for the WAFFLE framework.*
