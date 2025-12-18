# retry

Retry utilities with backoff strategies and circuit breaker for WAFFLE applications.

## Overview

The `retry` package provides:
- **Retry logic** — Configurable retry with exponential/linear/constant backoff
- **HTTP transport** — Automatic retries for HTTP requests
- **Circuit breaker** — Prevent cascading failures
- **Permanent errors** — Mark errors that should not be retried

## Import

```go
import "github.com/dalemusser/waffle/retry"
```

---

## Quick Start

```go
// Simple retry with defaults (3 attempts, exponential backoff)
err := retry.Simple(ctx, func(ctx context.Context) error {
    return callExternalAPI(ctx)
})

// HTTP client with automatic retries
client := retry.Client()
resp, err := client.Get("https://api.example.com/data")
```

---

## Basic Retry

### Simple Retry

**Location:** `retry.go`

```go
// Retry with default settings
err := retry.Simple(ctx, func(ctx context.Context) error {
    return doSomething(ctx)
})

// Retry with specific attempt count
err := retry.WithAttempts(ctx, 5, func(ctx context.Context) error {
    return doSomething(ctx)
})
```

### Retry with Configuration

```go
err := retry.Do(ctx, retry.Config{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
    Jitter:       0.1,
    RetryIf: func(err error) bool {
        return !errors.Is(err, ErrNotFound)
    },
    OnRetry: func(attempt int, err error, delay time.Duration) {
        log.Printf("Attempt %d failed: %v, retrying in %v", attempt, err, delay)
    },
}, func(ctx context.Context) error {
    return callAPI(ctx)
})
```

**Config:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| MaxAttempts | int | 3 | Maximum attempts including first |
| InitialDelay | time.Duration | 100ms | Delay before first retry |
| MaxDelay | time.Duration | 30s | Maximum delay cap |
| Multiplier | float64 | 2.0 | Delay multiplier per attempt |
| Jitter | float64 | 0.1 | Randomness (0.0-1.0) |
| RetryIf | func(error) bool | nil | Custom retry condition |
| OnRetry | func(int, error, Duration) | nil | Callback before retry |

### Retry with Result

```go
result, err := retry.DoWithResult(ctx, retry.DefaultConfig(), func(ctx context.Context) (Data, error) {
    return fetchData(ctx)
})
```

---

## Backoff Strategies

### Exponential Backoff (Default)

Delays grow exponentially: 100ms → 200ms → 400ms → 800ms...

```go
cfg := retry.ExponentialBackoff(
    100*time.Millisecond, // initial
    30*time.Second,       // max
    2.0,                  // multiplier
)
```

### Constant Backoff

Same delay between each retry:

```go
cfg := retry.ConstantBackoff(
    500*time.Millisecond, // delay
    5,                    // attempts
)
```

### Linear Backoff

Delays grow linearly: 100ms → 200ms → 300ms → 400ms...

```go
cfg := retry.LinearBackoff(
    100*time.Millisecond, // initial
    100*time.Millisecond, // increment
    5,                    // attempts
)
```

### Fibonacci Backoff

Delays follow Fibonacci-like growth:

```go
cfg := retry.FibonacciBackoff(
    100*time.Millisecond, // initial
    5,                    // attempts
)
```

---

## Permanent Errors

Mark errors that should not be retried:

```go
func processItem(ctx context.Context, item Item) error {
    if !item.Valid() {
        // Don't retry validation errors
        return retry.PermanentError(ErrInvalidItem)
    }

    return sendToAPI(ctx, item)
}

// Use with default skip
err := retry.Do(ctx, retry.Config{
    MaxAttempts: 5,
    RetryIf:     retry.SkipPermanent,
}, func(ctx context.Context) error {
    return processItem(ctx, item)
})

// Check if error is permanent
if retry.IsPermanent(err) {
    log.Println("Permanent failure, won't retry")
}
```

---

## HTTP Retry

**Location:** `http.go`

### HTTP Client

```go
// Client with default retry settings
client := retry.Client()

// Custom configuration
client := retry.ClientWithConfig(retry.HTTPConfig{
    Config: retry.Config{
        MaxAttempts:  5,
        InitialDelay: 200 * time.Millisecond,
    },
    RetryStatusCodes:       []int{429, 500, 502, 503, 504},
    RetryOnConnectionError: true,
    RespectRetryAfter:      true,
    MaxRetryAfter:          5 * time.Minute,
})

// With custom base transport
client := retry.ClientWithBase(customTransport, retry.DefaultHTTPConfig())
```

**HTTPConfig:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| RetryStatusCodes | []int | [429,500,502,503,504] | Status codes to retry |
| RetryOnConnectionError | bool | true | Retry connection errors |
| RespectRetryAfter | bool | true | Honor Retry-After header |
| MaxRetryAfter | time.Duration | 5 min | Cap on Retry-After |

### HTTP Helper Functions

```go
// Simple GET with retries
resp, err := retry.Get(ctx, "https://api.example.com/data")

// Custom request with retries
req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
resp, err := retry.DoHTTP(ctx, client, req)
```

### Checking Retryability

```go
// Check status code
if retry.IsRetryableStatus(resp.StatusCode) {
    // Status like 429, 500, 502, 503, 504
}

// Check error
if retry.IsRetryableError(err) {
    // Not a context cancellation
}
```

---

## Circuit Breaker

**Location:** `circuit.go`

Prevents cascading failures by temporarily stopping requests to failing services.

### States

- **Closed**: Normal operation, requests go through
- **Open**: Failure threshold exceeded, requests fail fast
- **Half-Open**: Testing if service recovered

### Basic Usage

```go
circuit := retry.NewCircuit(retry.CircuitConfig{
    FailureThreshold:    5,  // Open after 5 failures
    SuccessThreshold:    2,  // Close after 2 successes in half-open
    Timeout:             30 * time.Second, // Time before half-open
    MaxHalfOpenRequests: 1,  // Concurrent requests in half-open
})

err := circuit.Do(ctx, func(ctx context.Context) error {
    return callExternalService(ctx)
})

if err == retry.ErrCircuitOpen {
    // Circuit is open, request was not attempted
}
```

### Circuit with Result

```go
result, err := retry.Execute(circuit, ctx, func(ctx context.Context) (Data, error) {
    return fetchData(ctx)
})
```

### State Monitoring

```go
circuit := retry.NewCircuit(retry.CircuitConfig{
    FailureThreshold: 5,
    OnStateChange: func(from, to retry.CircuitState) {
        log.Printf("Circuit changed: %s -> %s", from, to)
        metrics.RecordCircuitState(to.String())
    },
})

// Check current state
state := circuit.State()
log.Printf("Circuit state: %s", state)

// Manual control
circuit.Reset() // Force close
circuit.Trip()  // Force open
```

### Custom Failure Detection

```go
circuit := retry.NewCircuit(retry.CircuitConfig{
    FailureThreshold: 5,
    IsFailure: func(err error) bool {
        // Only count certain errors as failures
        var netErr net.Error
        if errors.As(err, &netErr) && netErr.Timeout() {
            return true
        }
        // Don't count 4xx as failures
        var httpErr *HTTPError
        if errors.As(err, &httpErr) && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
            return false
        }
        return err != nil
    },
})
```

### Combined Retry + Circuit Breaker

```go
cb := retry.NewCircuitBreaker(
    retry.CircuitConfig{
        FailureThreshold: 5,
        Timeout:          30 * time.Second,
    },
    retry.Config{
        MaxAttempts:  3,
        InitialDelay: 100 * time.Millisecond,
    },
)

// Retries within circuit breaker
err := cb.Do(ctx, func(ctx context.Context) error {
    return callAPI(ctx)
})

// With result
result, err := retry.ExecuteWithBreaker(cb, ctx, func(ctx context.Context) (Data, error) {
    return fetchData(ctx)
})
```

---

## WAFFLE Integration

### External API Calls

```go
func callPaymentAPI(ctx context.Context, payment Payment) error {
    client := retry.ClientWithConfig(retry.HTTPConfig{
        Config: retry.Config{
            MaxAttempts:  3,
            InitialDelay: 500 * time.Millisecond,
            OnRetry: func(attempt int, err error, delay time.Duration) {
                log.Printf("Payment API retry %d: %v", attempt, err)
            },
        },
        RetryStatusCodes: []int{429, 502, 503, 504},
        // Don't retry 500 - might indicate duplicate charge
    })

    body, _ := json.Marshal(payment)
    req, _ := http.NewRequestWithContext(ctx, "POST", paymentURL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("payment failed: %d", resp.StatusCode)
    }
    return nil
}
```

### Database Operations

```go
func queryWithRetry(ctx context.Context, db *sql.DB, query string) (*sql.Rows, error) {
    return retry.DoWithResult(ctx, retry.Config{
        MaxAttempts:  3,
        InitialDelay: 50 * time.Millisecond,
        RetryIf: func(err error) bool {
            // Retry connection errors, not query errors
            return errors.Is(err, sql.ErrConnDone) ||
                   strings.Contains(err.Error(), "connection reset")
        },
    }, func(ctx context.Context) (*sql.Rows, error) {
        return db.QueryContext(ctx, query)
    })
}
```

### Service Health with Circuit Breaker

```go
type ServiceClient struct {
    baseURL string
    circuit *retry.Circuit
    client  *http.Client
}

func NewServiceClient(baseURL string) *ServiceClient {
    return &ServiceClient{
        baseURL: baseURL,
        circuit: retry.NewCircuit(retry.CircuitConfig{
            FailureThreshold: 5,
            SuccessThreshold: 2,
            Timeout:          30 * time.Second,
            OnStateChange: func(from, to retry.CircuitState) {
                log.Printf("Service %s circuit: %s -> %s", baseURL, from, to)
            },
        }),
        client: retry.Client(),
    }
}

func (s *ServiceClient) Call(ctx context.Context, path string) (*Response, error) {
    return retry.Execute(s.circuit, ctx, func(ctx context.Context) (*Response, error) {
        req, _ := http.NewRequestWithContext(ctx, "GET", s.baseURL+path, nil)
        resp, err := s.client.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        var result Response
        json.NewDecoder(resp.Body).Decode(&result)
        return &result, nil
    })
}

func (s *ServiceClient) IsHealthy() bool {
    return s.circuit.State() != retry.CircuitOpen
}
```

### Background Job Retry

```go
func processJob(ctx context.Context, job Job) error {
    return retry.Do(ctx, retry.Config{
        MaxAttempts:  5,
        InitialDelay: 1 * time.Second,
        MaxDelay:     1 * time.Minute,
        Multiplier:   2.0,
        RetryIf: func(err error) bool {
            // Don't retry permanent failures
            if retry.IsPermanent(err) {
                return false
            }
            // Don't retry after context cancellation
            if errors.Is(err, context.Canceled) {
                return false
            }
            return true
        },
        OnRetry: func(attempt int, err error, delay time.Duration) {
            job.SetStatus("retrying")
            job.SetLastError(err.Error())
            log.Printf("Job %s retry %d: %v", job.ID, attempt, err)
        },
    }, func(ctx context.Context) error {
        return executeJob(ctx, job)
    })
}
```

---

## Errors

```go
retry.ErrCircuitOpen    // Circuit breaker is open
retry.ErrCircuitTimeout // Circuit breaker timeout (unused)
```

---

## See Also

- [timeout](../timeout/timeout.md) — Request timeouts
- [ratelimit](../ratelimit/ratelimit.md) — Rate limiting
- [jobs](../jobs/jobs.md) — Background jobs with retry
