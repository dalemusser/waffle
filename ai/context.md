# WAFFLE Framework Context

## Project Overview

**WAFFLE** (Web Application Framework for Flavorful Layered Engineering) is a Go web application framework designed for building production-ready web services. It provides a modular, layered architecture with sensible defaults and a rich library of reusable components.

**Repository:** `github.com/dalemusser/waffle`
**Go Version:** 1.24+
**Status:** In development

## Core Philosophy

- **Modular:** Each component (routing, logging, persistence) is independent. Use what you need.
- **Simple:** Clarity over cleverness. Easy to understand, fast to build.
- **Scalable:** Works for single users or production loads.
- **Layered:** Clean separation of concerns.
- **Composable:** Components work together seamlessly.

## Project Structure

```
waffle/
├── app/                    # Application scaffolding utilities
├── cmd/
│   ├── makewaffle/         # CLI for scaffolding new WAFFLE projects
│   └── wafflectl/          # Control CLI for WAFFLE services
├── config/                 # Configuration loading (viper + pflag + env)
├── docs/                   # Documentation
│   ├── core/               # Core concepts documentation
│   ├── flavors/            # Deployment flavor guides
│   ├── guides/             # How-to guides
│   ├── pantry/             # Pantry package documentation
│   ├── philosophy/         # Design philosophy
│   └── reference/          # API reference
├── httputil/               # HTTP utility functions
├── internal/
│   └── wafflegen/          # Code generation utilities
├── logging/                # Structured logging (zap-based)
├── metrics/                # Prometheus metrics
├── middleware/             # HTTP middleware (compression, size limits, etc.)
├── pantry/                 # Reusable component library (see below)
├── router/                 # Chi router wrapper with standard middleware
├── server/                 # HTTP server with graceful shutdown, TLS, Let's Encrypt
└── windowsservice/         # Windows service support
```

## The Pantry

The `pantry/` directory contains reusable packages for common functionality:

### Authentication & Authorization
- `auth/jwt/` - JWT token generation, validation, refresh, middleware
- `auth/oauth2/` - OAuth2 providers (Google, GitHub, Microsoft, Apple, Okta, Discord, LinkedIn, and education providers: Clever, ClassLink, Canvas, Schoology, etc.)
- `auth/apikey/` - API key authentication

### Databases
- `db/mongo/` - MongoDB connection and utilities
- `db/postgres/` - PostgreSQL (pgx)
- `db/mysql/` - MySQL
- `db/sqlite/` - SQLite
- `db/redis/` - Redis

### MongoDB Utilities
- `mongo/` - Cursor helpers, keyset pagination, validation, error handling

### Caching & Sessions
- `cache/` - Cache interface with memory and Redis implementations
- `session/` - Session management with memory and Redis stores

### Cryptography
- `crypto/` - Password hashing (argon2), encryption, random ID generation

### Communication
- `email/` - Email sending (go-mail)
- `websocket/` - WebSocket hub and connection management
- `sse/` - Server-Sent Events

### Background Processing
- `jobs/` - Background job pool with worker management
- `mq/rabbitmq/` - RabbitMQ message queue
- `mq/sqs/` - AWS SQS

### HTTP Utilities
- `httpnav/` - Back button and navigation state
- `fileserver/` - Static file serving
- `pagination/` - Keyset and offset pagination
- `ratelimit/` - Rate limiting
- `requestid/` - Request ID propagation and logging
- `retry/` - HTTP client retry with circuit breaker
- `timeout/` - Context timeouts for HTTP operations

### Templates
- `templates/` - Go template engine with adapter, custom functions, view rendering

### Error Handling
- `errors/` - Structured errors with HTTP status mapping

### Push Notifications
- `apns/` - Apple Push Notification Service
- `fcm/` - Firebase Cloud Messaging
- `notify/` - Unified notification interface

### Other Utilities
- `geo/` - Geolocation (IP to location, timezone)
- `i18n/` - Internationalization
- `pdf/` - PDF generation
- `export/` - Data export utilities
- `search/` - Search functionality
- `storage/` - File storage (local, S3, Azure Blob, GCS)
- `text/` - Text utilities (case folding)
- `validate/` - Input validation
- `version/` - Version information
- `webhook/` - Webhook handling
- `audit/` - Audit logging
- `feature/` - Feature flags
- `urlutil/` - URL utilities
- `testing/` - Test helpers and HTTP recorder
- `query/` - Query building utilities

## Configuration System

WAFFLE uses a layered configuration system with precedence (highest wins):
1. Explicit CLI flags
2. Environment variables (WAFFLE_* prefix)
3. Config files (config.yaml, config.json, config.toml)
4. Defaults

### Core Configuration (CoreConfig)

```go
type CoreConfig struct {
    Env      string        // "dev" | "prod"
    LogLevel string        // debug, info, warn, error

    HTTP HTTPConfig        // Ports, timeouts
    TLS  TLSConfig         // Certs, Let's Encrypt
    CORS CORSConfig        // CORS settings

    DBConnectTimeout    time.Duration
    IndexBootTimeout    time.Duration
    MaxRequestBodyBytes int64
    EnableCompression   bool
    CompressionLevel    int
}
```

### App-Specific Configuration

Applications define their own config keys:

```go
appKeys := []config.AppKey{
    {Name: "mongo_uri", Default: "mongodb://localhost:27017", Desc: "MongoDB URI"},
    {Name: "session_name", Default: "app-session", Desc: "Session cookie name"},
}
coreCfg, appCfg, err := config.LoadWithAppConfig(logger, "MYAPP", appKeys)
mongoURI := appCfg.String("mongo_uri")
```

Environment variables use app prefix: `MYAPP_MONGO_URI`

## Router

The router package provides a pre-configured Chi router with standard middleware:

```go
r := router.New(coreCfg, logger)
// Includes: RequestID, RealIP, Recoverer, Compression, BodySizeLimit,
//           Metrics, RequestLogger, NotFound/MethodNotAllowed handlers
```

## Server

The server package provides HTTP/HTTPS server with:
- Graceful shutdown
- Let's Encrypt auto-TLS (HTTP-01 and DNS-01 challenges)
- Manual TLS certificate support
- Configurable timeouts

## CLI Tools

### makewaffle

Scaffolds new WAFFLE projects:

```bash
makewaffle new myapp --module github.com/you/myapp
```

### wafflectl

Control CLI for WAFFLE services (health checks, version info, etc.)

## Dependencies

Key external dependencies:
- `github.com/go-chi/chi/v5` - HTTP router
- `go.uber.org/zap` - Structured logging
- `github.com/spf13/viper` - Configuration
- `github.com/spf13/pflag` - CLI flags
- `go.mongodb.org/mongo-driver` - MongoDB driver
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `golang.org/x/oauth2` - OAuth2
- `github.com/aws/aws-sdk-go-v2` - AWS SDK (S3, SQS, CloudFront, Route53)
- `github.com/prometheus/client_golang` - Metrics
- `github.com/coder/websocket` - WebSocket
- `github.com/wneessen/go-mail` - Email

## Patterns & Conventions

### Package Structure
- Each pantry package is self-contained with its own types and interfaces
- Packages expose constructors like `New(...)` or `NewWithConfig(...)`
- Configuration is passed explicitly, not through globals

### Error Handling
- Use `pantry/errors` for structured errors with HTTP status
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`

### Logging
- Use structured logging via zap
- Include request IDs for traceability
- Log at appropriate levels (debug for dev, info+ for prod)

### Testing
- Use `pantry/testing` for test helpers
- HTTP tests use httptest.ResponseRecorder pattern

## Related Project

**StrataHub** (`stratahub/`) uses WAFFLE as its framework. See `stratahub/ai/context.md` for StrataHub-specific context.

## Documentation

Full documentation is in the `docs/` directory. Read `docs/waffle-docs.md` for the complete index.

### Read at Session Start

When starting work on WAFFLE itself, read:
- `docs/core/architecture.md` - System design and visual diagrams
- `docs/guides/patterns/` - Handler, routing, middleware, and feature patterns

### Read When Working On Specific Areas

**Configuration:**
- `docs/core/configuration.md` - CoreConfig, Viper, environment variables
- `docs/core/app-config-keys.md` - Adding app-specific config keys
- `docs/guides/patterns/appconfig.md` - Configuration patterns

**Routing & Handlers:**
- `docs/core/routing.md` - Chi router, middleware, route patterns
- `docs/guides/patterns/handlers.md` - Handler structure patterns
- `docs/guides/patterns/routes.md` - Routing patterns
- `docs/guides/patterns/middleware.md` - Middleware patterns

**Databases:**
- `docs/guides/databases/mongo.md` - MongoDB setup and usage
- `docs/guides/databases/postgres.md` - PostgreSQL setup
- `docs/guides/databases/redis.md` - Redis setup

**Authentication:**
- `docs/guides/auth/oauth2.md` - OAuth2 integration
- `docs/guides/auth/providers/` - Provider-specific guides (Google, GitHub, Clever, etc.)

**Templates & UI:**
- `docs/flavors/server-html-htmx/` - HTMX + Tailwind (recommended approach)
- `docs/flavors/server-html-htmx/templates.md` - Go template patterns
- `docs/flavors/server-html-htmx/htmx.md` - HTMX integration

**File Serving:**
- `docs/guides/fileserving/static-files.md` - Embedded static files
- `docs/guides/fileserving/static-files-filesystem.md` - Filesystem with pre-compression

**Deployment:**
- `docs/guides/deployment/windows-service.md` - Windows service setup
- `docs/core/operational-endpoints.md` - Health, metrics, pprof endpoints

**Understanding Design Decisions:**
- `docs/philosophy/` - Why WAFFLE exists and design principles
- `docs/philosophy/why-htmx-tailwind.md` - Recommended UI approach rationale

### Pantry Package Documentation

When using or modifying pantry packages:
- `docs/pantry/README.md` - Pantry overview by category
- `docs/pantry/alphabetical-index.md` - Quick lookup by package name
