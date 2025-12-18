# metrics

Prometheus metrics for HTTP request monitoring and Go runtime statistics.

## Overview

The `metrics` package provides Prometheus instrumentation for WAFFLE applications. It includes HTTP request duration tracking, Go runtime metrics, and process metrics. Use this with Prometheus, Grafana, or any metrics aggregation system.

## Import

```go
import "github.com/dalemusser/waffle/metrics"
```

## Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    // Register default metrics collectors
    metrics.RegisterDefault(logger)

    r := chi.NewRouter()

    // Add metrics middleware to track request durations
    r.Use(metrics.HTTPMetrics)

    // Expose metrics endpoint for Prometheus scraping
    r.Handle("/metrics", metrics.Handler())

    // ... other routes

    return r, nil
}
```

## API

### RegisterDefault

**Location:** `metrics.go`

```go
func RegisterDefault(logger *zap.Logger)
```

Registers the default Prometheus collectors. Call once at startup before using other metrics functions.

**Registered collectors:**
- **Go collector** — Goroutines, GC stats, memory allocation
- **Process collector** — CPU, memory, file descriptors, start time
- **HTTP request histogram** — Request durations by path, method, status

Safe to call multiple times (ignores already-registered errors).

**Example:**

```go
func main() {
    logger := logging.MustBuildLogger(cfg.LogLevel, cfg.Env)
    metrics.RegisterDefault(logger)
}
```

### HTTPMetrics

**Location:** `metrics.go`

```go
func HTTPMetrics(next http.Handler) http.Handler
```

Middleware that records HTTP request durations into the `http_request_duration_seconds` histogram.

**Labels:**
| Label | Description |
|-------|-------------|
| `path` | Request URL path |
| `method` | HTTP method (GET, POST, etc.) |
| `status` | Response status code |

**Histogram buckets:** 10ms, 100ms, 300ms, 1.2s, 5s

**Example:**

```go
r := chi.NewRouter()
r.Use(metrics.HTTPMetrics)
```

### Handler

**Location:** `metrics.go`

```go
func Handler() http.Handler
```

Returns an HTTP handler that exposes Prometheus metrics in the standard text format. Mount this at `/metrics` for Prometheus to scrape.

**Example:**

```go
r.Handle("/metrics", metrics.Handler())
```

## Exposed Metrics

### HTTP Request Duration

```
# HELP http_request_duration_seconds Duration of HTTP requests.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="0.01"} 150
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="0.1"} 200
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="0.3"} 210
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="1.2"} 212
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="5"} 212
http_request_duration_seconds_bucket{method="GET",path="/api/users",status="200",le="+Inf"} 212
http_request_duration_seconds_sum{method="GET",path="/api/users",status="200"} 15.234
http_request_duration_seconds_count{method="GET",path="/api/users",status="200"} 212
```

### Go Runtime Metrics

```
go_goroutines 42
go_gc_duration_seconds{quantile="0.5"} 0.000123
go_memstats_alloc_bytes 1.234567e+07
go_memstats_heap_objects 12345
```

### Process Metrics

```
process_cpu_seconds_total 123.45
process_resident_memory_bytes 5.6789e+07
process_open_fds 12
process_start_time_seconds 1.7023456789e+09
```

## Patterns

### Standard Setup with Protected Metrics

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    metrics.RegisterDefault(logger)

    r := chi.NewRouter()
    r.Use(metrics.HTTPMetrics)

    // Public routes
    r.Get("/", homeHandler)
    r.Route("/api", func(r chi.Router) {
        r.Get("/users", listUsers)
    })

    // Protected metrics endpoint
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.MetricsKey, apikey.Options{
            Realm: "metrics",
        }, logger))
        r.Handle("/metrics", metrics.Handler())
    })

    return r, nil
}
```

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'myapp'
    scrape_interval: 15s
    static_configs:
      - targets: ['myapp:8080']
    metrics_path: /metrics
```

### Kubernetes ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: myapp
spec:
  selector:
    matchLabels:
      app: myapp
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

### Grafana Dashboard Queries

```promql
# Request rate by endpoint
rate(http_request_duration_seconds_count[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate (5xx responses)
sum(rate(http_request_duration_seconds_count{status=~"5.."}[5m]))
  /
sum(rate(http_request_duration_seconds_count[5m]))

# Memory usage
process_resident_memory_bytes / 1024 / 1024

# Goroutine count
go_goroutines
```

### Adding Custom Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    ordersProcessed = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "orders_processed_total",
        Help: "Total number of orders processed",
    })

    orderValue = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "order_value_dollars",
        Help:    "Order values in dollars",
        Buckets: []float64{10, 50, 100, 500, 1000},
    })
)

func init() {
    prometheus.MustRegister(ordersProcessed)
    prometheus.MustRegister(orderValue)
}

func processOrder(order Order) {
    // ... process order ...
    ordersProcessed.Inc()
    orderValue.Observe(order.Total)
}
```

## See Also

- [health](../health/health.md) — Health check endpoints
- [pprof](../pprof/pprof.md) — Profiling endpoints
- [logging](../logging/logging.md) — Structured logging

