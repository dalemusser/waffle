# server

HTTP/HTTPS server with graceful shutdown and automatic TLS.

## Overview

The `server` package provides production-ready HTTP server functionality including graceful shutdown on OS signals, automatic TLS via Let's Encrypt, manual TLS certificates, and HTTP-to-HTTPS redirection. It handles the complexity of running secure web servers so you can focus on your application logic.

## Import

```go
import "github.com/dalemusser/waffle/server"
```

## Quick Start

```go
func main() {
    cfg, _ := config.Load()
    logger := logging.MustBuildLogger(cfg.Core.LogLevel, cfg.Core.Env)

    handler := buildHandler(cfg, logger)

    // Create context that cancels on SIGINT/SIGTERM
    ctx, cancel := server.WithShutdownSignals(context.Background(), logger)
    defer cancel()

    // Start server (blocks until shutdown)
    if err := server.ListenAndServeWithContext(ctx, cfg.Core, handler, logger); err != nil {
        logger.Fatal("server error", zap.Error(err))
    }
}
```

## API

### WithShutdownSignals

**Location:** `server.go`

```go
func WithShutdownSignals(parent context.Context, logger *zap.Logger) (context.Context, context.CancelFunc)
```

Returns a context that is canceled when the process receives SIGINT (Ctrl+C) or SIGTERM. Use this as the parent context for `ListenAndServeWithContext` to enable graceful shutdown.

**Example:**

```go
ctx, cancel := server.WithShutdownSignals(context.Background(), logger)
defer cancel()

// ctx is canceled when shutdown signal received
```

### ListenAndServeWithContext

**Location:** `server.go`

```go
func ListenAndServeWithContext(
    ctx context.Context,
    cfg *config.CoreConfig,
    handler http.Handler,
    logger *zap.Logger,
) error
```

Starts an HTTP or HTTPS server and blocks until the context is canceled or a terminal error occurs. Automatically selects the serving mode based on configuration.

**Server timeouts (hardcoded for security):**

| Timeout | Value | Purpose |
|---------|-------|---------|
| ReadTimeout | 15s | Max time to read entire request |
| ReadHeaderTimeout | 10s | Max time to read request headers |
| WriteTimeout | 60s | Max time to write response |
| IdleTimeout | 120s | Max time for keep-alive connections |

**Serving modes:**

| Mode | Config | Behavior |
|------|--------|----------|
| HTTP only | `UseHTTPS: false` | Plain HTTP on `HTTPPort` |
| Let's Encrypt | `UseHTTPS: true`, `UseLetsEncrypt: true` | HTTPS on `HTTPSPort`, ACME + redirect on port 80 |
| Manual TLS | `UseHTTPS: true`, `UseLetsEncrypt: false` | HTTPS on `HTTPSPort`, redirect on port 80 |

**Returns:**
- `nil` on graceful shutdown
- Error if server fails to start or encounters a terminal error

## Serving Modes

### HTTP Only

For development or when TLS is terminated by a load balancer/proxy.

```yaml
# config.yaml
http:
  http_port: 8080
  use_https: false
```

```go
// Starts HTTP server on :8080
server.ListenAndServeWithContext(ctx, cfg.Core, handler, logger)
```

### HTTPS with Let's Encrypt

Automatic certificate provisioning and renewal via ACME http-01 challenge.

```yaml
# config.yaml
http:
  use_https: true
  https_port: 443
tls:
  use_lets_encrypt: true
  domain: "myapp.example.com"
  lets_encrypt_email: "admin@example.com"
  lets_encrypt_cache_dir: "/var/cache/letsencrypt"
```

**Behavior:**
- Port 443: HTTPS with auto-provisioned certificates
- Port 80: ACME challenge handler + HTTP→HTTPS redirect
- Certificates cached in `lets_encrypt_cache_dir`
- Pre-warms certificate before accepting HTTPS connections

**Requirements:**
- Domain must resolve to your server
- Ports 80 and 443 must be accessible from the internet
- Write access to cache directory

### HTTPS with Manual Certificates

Use your own certificates (e.g., from a CA, self-signed, or corporate PKI).

```yaml
# config.yaml
http:
  use_https: true
  https_port: 443
tls:
  use_lets_encrypt: false
  cert_file: "/etc/ssl/certs/myapp.crt"
  key_file: "/etc/ssl/private/myapp.key"
```

**Behavior:**
- Port 443: HTTPS with provided certificates
- Port 80: HTTP→HTTPS redirect
- TLS 1.2 minimum version enforced

## Patterns

### Standard Main Function

```go
func main() {
    // Bootstrap logger for early startup
    bootLog := logging.BootstrapLogger()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        bootLog.Fatal("config load failed", zap.Error(err))
    }

    // Build configured logger
    logger := logging.MustBuildLogger(cfg.Core.LogLevel, cfg.Core.Env)
    defer logger.Sync()

    // Build HTTP handler
    handler, err := buildHandler(cfg.Core, cfg.App, cfg.DB, logger)
    if err != nil {
        logger.Fatal("failed to build handler", zap.Error(err))
    }

    // Setup graceful shutdown
    ctx, cancel := server.WithShutdownSignals(context.Background(), logger)
    defer cancel()

    // Start server
    logger.Info("starting server",
        zap.Int("http_port", cfg.Core.HTTP.HTTPPort),
        zap.Bool("https", cfg.Core.HTTP.UseHTTPS),
    )

    if err := server.ListenAndServeWithContext(ctx, cfg.Core, handler, logger); err != nil {
        logger.Fatal("server error", zap.Error(err))
    }

    logger.Info("server stopped")
}
```

### With app.Run

The `app` package wraps this pattern for you:

```go
func main() {
    app.Run(app.Hooks{
        BuildHandler: buildHandler,
    })
}
```

See [app](../app/app.md) for the full lifecycle.

### Behind a Load Balancer

When TLS is terminated at the load balancer:

```yaml
# config.yaml
http:
  http_port: 8080
  use_https: false
```

The load balancer handles HTTPS and forwards plain HTTP to your app. Use `X-Forwarded-Proto` header (handled by Chi's RealIP middleware) to detect the original protocol.

### Development with Self-Signed Certificates

```bash
# Generate self-signed cert for development
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost"
```

```yaml
# config.yaml
http:
  use_https: true
  https_port: 8443
tls:
  cert_file: "./cert.pem"
  key_file: "./key.pem"
```

### Kubernetes Deployment

```yaml
# Kubernetes typically terminates TLS at ingress
http:
  http_port: 8080
  use_https: false

# Health check endpoints for probes
# (configured in your handler)
```

```yaml
# kubernetes deployment.yaml
spec:
  containers:
    - name: myapp
      ports:
        - containerPort: 8080
      livenessProbe:
        httpGet:
          path: /live
          port: 8080
      readinessProbe:
        httpGet:
          path: /ready
          port: 8080
```

## Graceful Shutdown

When a shutdown signal is received:

1. Context is canceled
2. Server stops accepting new connections
3. In-flight requests have 15 seconds to complete
4. Auxiliary server (port 80) is shut down
5. Primary server is shut down
6. Function returns `nil`

This allows zero-downtime deployments when combined with load balancer health checks.

## See Also

- [app](../app/app.md) — Application lifecycle wrapper
- [config](../config/config.md) — HTTP and TLS configuration
- [health](../health/health.md) — Health check endpoints for load balancers

