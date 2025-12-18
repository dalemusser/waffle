# version

Build and version information endpoint.

## Overview

The `version` package provides a `/version` endpoint that exposes build information as JSON. This is useful for deployment verification, debugging, and monitoring. Version info is set at build time using Go's `-ldflags` mechanism.

## Import

```go
import "github.com/dalemusser/waffle/version"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Mount version endpoint
    version.Mount(r)

    // ... other routes

    return r, nil
}
```

Build with version info:

```bash
go build -ldflags "-X github.com/dalemusser/waffle/version.Version=1.2.3 \
                   -X github.com/dalemusser/waffle/version.Commit=$(git rev-parse --short HEAD) \
                   -X github.com/dalemusser/waffle/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## API

### Variables

**Location:** `version.go`

```go
var (
    Version   = "dev"     // Semantic version (e.g., "1.2.3")
    Commit    = "unknown" // Git commit SHA
    BuildTime = "unknown" // Build timestamp (RFC3339)
)
```

Set these at build time using `-ldflags`:

```bash
-X github.com/dalemusser/waffle/version.Version=1.2.3
-X github.com/dalemusser/waffle/version.Commit=abc123
-X github.com/dalemusser/waffle/version.BuildTime=2024-01-15T10:30:00Z
```

### Info

**Location:** `version.go`

```go
type Info struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    BuildTime string `json:"build_time"`
    GoVersion string `json:"go_version"`
    OS        string `json:"os"`
    Arch      string `json:"arch"`
}
```

Contains version and build information. Includes runtime details (Go version, OS, architecture) that are detected automatically.

### Get

**Location:** `version.go`

```go
func Get() Info
```

Returns the current version info. Useful for logging at startup or embedding in other responses.

**Example:**

```go
func main() {
    logger.Info("starting application", zap.String("version", version.Get().Version))
}
```

### Handler

**Location:** `version.go`

```go
func Handler() http.Handler
```

Returns an HTTP handler that responds with version info as JSON.

### Mount

**Location:** `version.go`

```go
func Mount(r chi.Router)
```

Attaches a `/version` route to the router.

### MountAt

**Location:** `version.go`

```go
func MountAt(r chi.Router, path string)
```

Attaches a version route at a custom path.

**Example:**

```go
version.MountAt(r, "/api/v1/version")
```

### String

**Location:** `version.go`

```go
func String() string
```

Returns a human-readable version string for logging or CLI output.

**Example:**

```go
fmt.Println("MyApp", version.String())
// Output: MyApp 1.2.3 (abc123, built 2024-01-15T10:30:00Z)
```

## Response Format

```json
{
  "version": "1.2.3",
  "commit": "abc123def",
  "build_time": "2024-01-15T10:30:00Z",
  "go_version": "go1.22.0",
  "os": "linux",
  "arch": "amd64"
}
```

## Patterns

### Makefile Integration

```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT  ?= $(shell git rev-parse --short HEAD)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -ldflags "\
    -X github.com/dalemusser/waffle/version.Version=$(VERSION) \
    -X github.com/dalemusser/waffle/version.Commit=$(COMMIT) \
    -X github.com/dalemusser/waffle/version.BuildTime=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o bin/myapp ./cmd/myapp
```

### GitHub Actions

```yaml
- name: Build
  run: |
    VERSION=${GITHUB_REF_NAME:-dev}
    COMMIT=${GITHUB_SHA::7}
    BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    go build -ldflags "\
      -X github.com/dalemusser/waffle/version.Version=$VERSION \
      -X github.com/dalemusser/waffle/version.Commit=$COMMIT \
      -X github.com/dalemusser/waffle/version.BuildTime=$BUILD_TIME" \
      -o myapp ./cmd/myapp
```

### Goreleaser

```yaml
# .goreleaser.yaml
builds:
  - main: ./cmd/myapp
    ldflags:
      - -X github.com/dalemusser/waffle/version.Version={{.Version}}
      - -X github.com/dalemusser/waffle/version.Commit={{.ShortCommit}}
      - -X github.com/dalemusser/waffle/version.BuildTime={{.Date}}
```

### Log Version at Startup

```go
func main() {
    logger := logging.MustBuildLogger(cfg.LogLevel, cfg.Env)

    info := version.Get()
    logger.Info("application starting",
        zap.String("version", info.Version),
        zap.String("commit", info.Commit),
        zap.String("go_version", info.GoVersion),
    )
}
```

### CLI Version Flag

```go
func main() {
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Println("myapp", version.String())
        os.Exit(0)
    }

    // Continue with normal startup...
}
```

### Protected Version Endpoint

```go
// Public version (safe to expose)
version.Mount(r)

// Or protect it if you prefer
r.Group(func(r chi.Router) {
    r.Use(apikey.Require(appCfg.AdminKey, apikey.Options{Realm: "admin"}, logger))
    version.Mount(r)
})
```

### Include in Health Response

```go
type ExtendedHealth struct {
    Status  string       `json:"status"`
    Version version.Info `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    resp := ExtendedHealth{
        Status:  "ok",
        Version: version.Get(),
    }
    httputil.WriteJSON(w, http.StatusOK, resp)
}
```

## See Also

- [health](../health/health.md) — Health check endpoints
- [app](../app/app.md) — Application lifecycle

