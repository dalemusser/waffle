# pprof

Go runtime profiling endpoints for performance analysis.

## Overview

The `pprof` package mounts the standard Go profiling handlers on a Chi router. Use these endpoints to analyze CPU usage, memory allocation, goroutine behavior, and more. Profiles can be viewed in a browser or analyzed with `go tool pprof`.

## Import

```go
import "github.com/dalemusser/waffle/pprof"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Mount pprof with API key protection
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.DebugKey, apikey.Options{
            Realm:      "debug",
            CookieName: "debug_auth", // Enable browser access
        }, logger))
        pprof.Mount(r)
    })

    // ... other routes

    return r, nil
}
```

## API

### Mount

**Location:** `pprof.go`

```go
func Mount(r chi.Router)
```

Attaches the standard Go pprof handlers under `/debug/pprof`. Apply authentication middleware before calling Mount if protection is needed.

**Mounted endpoints:**

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | Index page listing all profiles |
| `/debug/pprof/cmdline` | Command line arguments |
| `/debug/pprof/profile` | CPU profile (30s default) |
| `/debug/pprof/symbol` | Symbol lookup for addresses |
| `/debug/pprof/trace` | Execution trace |
| `/debug/pprof/heap` | Heap memory profile |
| `/debug/pprof/goroutine` | Goroutine stack traces |
| `/debug/pprof/allocs` | Memory allocation profile |
| `/debug/pprof/block` | Blocking profile |
| `/debug/pprof/mutex` | Mutex contention profile |
| `/debug/pprof/threadcreate` | Thread creation profile |

## Available Profiles

### CPU Profile

Samples CPU usage over time. Default duration is 30 seconds.

```bash
# Collect 30-second CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile

# Collect 60-second CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=60
```

### Heap Profile

Shows current memory allocations.

```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

### Goroutine Profile

Lists all goroutines and their stack traces.

```bash
# Interactive analysis
go tool pprof http://localhost:8080/debug/pprof/goroutine

# View in browser (debug=1 for human-readable)
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

### Allocations Profile

Shows memory allocations since program start.

```bash
go tool pprof http://localhost:8080/debug/pprof/allocs
```

### Block Profile

Shows where goroutines block waiting on synchronization primitives. Requires `runtime.SetBlockProfileRate()` to be called.

```bash
go tool pprof http://localhost:8080/debug/pprof/block
```

### Mutex Profile

Shows mutex contention. Requires `runtime.SetMutexProfileFraction()` to be called.

```bash
go tool pprof http://localhost:8080/debug/pprof/mutex
```

### Execution Trace

Captures detailed execution trace for analysis with `go tool trace`.

```bash
# Collect 5-second trace
curl -o trace.out http://localhost:8080/debug/pprof/trace?seconds=5
go tool trace trace.out
```

## Patterns

### Protected pprof (Recommended)

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Public routes
    r.Get("/", homeHandler)
    r.Route("/api", apiRoutes)

    // Protected debug endpoints
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.DebugKey, apikey.Options{
            Realm:      "debug",
            CookieName: "debug_auth",
        }, logger))
        pprof.Mount(r)
    })

    return r, nil
}
```

### Development-Only pprof

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Only expose pprof in development
    if core.Env != "prod" {
        pprof.Mount(r)
    }

    // ... routes

    return r, nil
}
```

### Enable Block and Mutex Profiling

```go
import "runtime"

func main() {
    // Enable block profiling (1 = track all blocking events)
    runtime.SetBlockProfileRate(1)

    // Enable mutex profiling (fraction of mutex events to track)
    runtime.SetMutexProfileFraction(1)

    // ... start application
}
```

### Using go tool pprof

```bash
# Interactive mode
go tool pprof http://localhost:8080/debug/pprof/heap
(pprof) top 10
(pprof) list functionName
(pprof) web  # Opens in browser (requires graphviz)

# Direct commands
go tool pprof -top http://localhost:8080/debug/pprof/heap
go tool pprof -png http://localhost:8080/debug/pprof/heap > heap.png

# Compare two profiles
go tool pprof -base old.prof new.prof
```

### Browser Access with API Key

```bash
# First request sets a cookie
curl "http://localhost:8080/debug/pprof/?api_key=your-key"

# Or access directly in browser with query param
# http://localhost:8080/debug/pprof/?api_key=your-key

# Subsequent requests use the cookie
open http://localhost:8080/debug/pprof/
```

### Continuous Profiling Script

```bash
#!/bin/bash
# collect-profiles.sh - Collect profiles periodically

HOST="http://localhost:8080"
DIR="profiles/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$DIR"

# Heap profile
curl -s "$HOST/debug/pprof/heap" > "$DIR/heap.prof"

# Goroutine profile
curl -s "$HOST/debug/pprof/goroutine" > "$DIR/goroutine.prof"

# 10-second CPU profile
curl -s "$HOST/debug/pprof/profile?seconds=10" > "$DIR/cpu.prof"

echo "Profiles saved to $DIR"
```

## Common Analysis Tasks

### Find Memory Leaks

```bash
# Collect heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

(pprof) top 20 -cum  # Sort by cumulative allocation
(pprof) list MyFunction  # See allocations in specific function
```

### Find CPU Bottlenecks

```bash
go tool pprof http://localhost:8080/debug/pprof/profile

(pprof) top 10  # Top CPU consumers
(pprof) web  # Visualize call graph
```

### Find Goroutine Leaks

```bash
# Human-readable goroutine dump
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# Or in pprof
go tool pprof http://localhost:8080/debug/pprof/goroutine
(pprof) top  # See where goroutines are stuck
```

## See Also

- [metrics](../metrics/metrics.md) — Prometheus metrics
- [health](../health/health.md) — Health check endpoints
- [auth/apikey](../auth/auth.md#api-key) — API key middleware for protection

