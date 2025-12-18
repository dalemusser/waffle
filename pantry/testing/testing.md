# testing

Test utilities for WAFFLE applications.

## Overview

The `testing` package provides helpers for testing HTTP handlers and WAFFLE services: test server setup, fluent request builders, response assertions, and common test utilities.

## Import

```go
import waffletest "github.com/dalemusser/waffle/testing"
```

> **Note:** Import with an alias to avoid conflicts with the standard `testing` package.

---

## Server Testing

### NewServer

**Location:** `testing.go`

```go
func NewServer(t *testing.T, r chi.Router) *Server
```

Creates a test HTTP server with the given router. The server is automatically closed when the test completes.

**Example:**

```go
func TestAPI(t *testing.T) {
    r := chi.NewRouter()
    r.Get("/users", listUsersHandler)
    r.Post("/users", createUserHandler)

    srv := waffletest.NewServer(t, r)

    srv.Get("/users").
        Do().
        StatusOK().
        ContentTypeJSON()
}
```

### NewTLSServer

**Location:** `testing.go`

```go
func NewTLSServer(t *testing.T, r chi.Router) *Server
```

Creates a test server with TLS enabled.

---

## Request Builder

The `Server` provides a fluent request builder for constructing and executing HTTP requests.

### Creating Requests

```go
srv.Get("/path")                    // GET request
srv.Post("/path")                   // POST request
srv.Put("/path")                    // PUT request
srv.Patch("/path")                  // PATCH request
srv.Delete("/path")                 // DELETE request
srv.Request("OPTIONS", "/path")     // Custom method
```

### Request Methods

| Method | Description |
|--------|-------------|
| `Header(key, value)` | Set a single header |
| `Headers(map[string]string)` | Set multiple headers |
| `Query(key, value)` | Set a query parameter |
| `Queries(map[string]string)` | Set multiple query parameters |
| `Body(io.Reader)` | Set request body |
| `BodyString(string)` | Set body from string |
| `BodyBytes([]byte)` | Set body from bytes |
| `JSON(any)` | Set JSON body and Content-Type |
| `Form(url.Values)` | Set form body and Content-Type |
| `Bearer(token)` | Set Bearer authorization |
| `BasicAuth(user, pass)` | Set Basic authorization |
| `Cookie(name, value)` | Add a cookie |
| `Do()` | Execute the request |

**Example:**

```go
// POST JSON with auth
srv.Post("/api/users").
    Bearer("my-token").
    JSON(map[string]string{
        "name":  "Alice",
        "email": "alice@example.com",
    }).
    Do().
    StatusCreated()

// GET with query params
srv.Get("/api/users").
    Query("page", "1").
    Query("limit", "10").
    Do().
    StatusOK()

// Form submission
srv.Post("/login").
    Form(url.Values{
        "username": {"alice"},
        "password": {"secret"},
    }).
    Do().
    StatusOK()
```

---

## Response Assertions

The `Response` type provides chainable assertion methods.

### Status Assertions

| Method | Status Code |
|--------|-------------|
| `Status(code)` | Assert exact status |
| `StatusOK()` | 200 |
| `StatusCreated()` | 201 |
| `StatusNoContent()` | 204 |
| `StatusBadRequest()` | 400 |
| `StatusUnauthorized()` | 401 |
| `StatusForbidden()` | 403 |
| `StatusNotFound()` | 404 |
| `StatusConflict()` | 409 |
| `StatusUnprocessableEntity()` | 422 |
| `StatusTooManyRequests()` | 429 |
| `StatusInternalServerError()` | 500 |

### Header Assertions

```go
resp.HeaderEquals("X-Request-ID", "abc123")
resp.HeaderContains("Content-Type", "json")
resp.HeaderExists("X-Request-ID")
resp.ContentType("application/json")
resp.ContentTypeJSON()
```

### Body Assertions

```go
resp.BodyEquals("exact match")
resp.BodyContains("substring")
resp.BodyNotContains("unwanted")
resp.BodyEmpty()
```

### JSON Assertions

```go
// Unmarshal into struct
var user User
resp.JSON(&user)

// Assert JSON path values
resp.JSONPathEquals("user.name", "Alice")
resp.JSONPathEquals("items.0.id", 123)
resp.JSONPathContains("error.message", "not found")

// Get value at path
name := resp.JSONPath("user.name").(string)
```

### Utilities

```go
body := resp.String()    // Get body as string
data := resp.Bytes()     // Get body as []byte
resp.Print()             // Log response for debugging
```

**Example:**

```go
srv.Get("/api/users/1").
    Do().
    StatusOK().
    ContentTypeJSON().
    JSONPathEquals("name", "Alice").
    JSONPathEquals("email", "alice@example.com")
```

---

## Recorder Testing

Test handlers without starting a server using `httptest.ResponseRecorder`.

### NewRecorder

**Location:** `recorder.go`

```go
func NewRecorder(t *testing.T) *Recorder
```

Creates a recorder for testing handlers directly.

**Example:**

```go
func TestHandler(t *testing.T) {
    rec := waffletest.NewRecorder(t)

    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    rec.Get("/health").
        Run(handler).
        StatusOK().
        ContentTypeJSON().
        JSONPathEquals("status", "ok")
}
```

### Request Methods

Same as `Server`:

```go
rec.Get("/path")
rec.Post("/path")
rec.Put("/path")
rec.Patch("/path")
rec.Delete("/path")
rec.Request("OPTIONS", "/path")
```

### RecorderRequest Methods

Same as `RequestBuilder` plus:

```go
rr.Run(handler http.Handler) *Response  // Execute against handler
```

---

## Test Helpers

### Logging

**Location:** `helpers.go`

```go
func TestLogger() *zap.Logger   // No-op logger for tests
func DevLogger() *zap.Logger    // Development logger for debugging
```

### Context

```go
func Context(t *testing.T) context.Context
func ContextWithTimeout(t *testing.T, timeout time.Duration) context.Context
```

Creates a context with automatic cleanup. Default timeout is 30 seconds.

**Example:**

```go
func TestDatabase(t *testing.T) {
    ctx := waffletest.Context(t)

    result, err := db.Query(ctx, "SELECT * FROM users")
    // ...
}
```

### Temporary Files

```go
func TempDir(t *testing.T) string
func TempFile(t *testing.T, name, content string) string
```

Creates temporary directories and files that are cleaned up after the test.

**Example:**

```go
func TestFileProcessing(t *testing.T) {
    path := waffletest.TempFile(t, "config.json", `{"key": "value"}`)

    cfg, err := LoadConfig(path)
    // ...
}
```

### Fixtures

```go
func ReadFixture(t *testing.T, path string) []byte
func ReadJSONFixture(t *testing.T, path string, v any)
```

Reads files from the `testdata` directory.

**Example:**

```go
func TestParsing(t *testing.T) {
    data := waffletest.ReadFixture(t, "sample.xml")

    var config Config
    waffletest.ReadJSONFixture(t, "config.json", &config)
}
```

### JSON Helpers

```go
func MustJSON(t *testing.T, v any) []byte
func MustJSONString(t *testing.T, v any) string
```

Marshals to JSON, failing the test on error.

### Environment Variables

```go
func SetEnv(t *testing.T, key, value string)
func UnsetEnv(t *testing.T, key string)
func RequireEnv(t *testing.T, key string) string
```

Set or unset environment variables for the duration of a test. Original values are restored after the test.

**Example:**

```go
func TestConfigFromEnv(t *testing.T) {
    waffletest.SetEnv(t, "DATABASE_URL", "postgres://localhost/test")

    cfg := LoadConfig()
    // ...
}

func TestIntegration(t *testing.T) {
    apiKey := waffletest.RequireEnv(t, "API_KEY")  // Skip if not set

    client := NewClient(apiKey)
    // ...
}
```

### Async Testing

```go
func Eventually(t *testing.T, check func() bool, timeout, interval time.Duration)
```

Retries a check function until it passes or times out.

**Example:**

```go
func TestAsync(t *testing.T) {
    startBackgroundJob()

    waffletest.Eventually(t, func() bool {
        return jobCompleted()
    }, 5*time.Second, 100*time.Millisecond)
}
```

### Test Control

```go
func Parallel(t *testing.T)                           // Mark test as parallel
func Skip(t *testing.T, condition bool, reason string) // Conditional skip
func SkipShort(t *testing.T)                          // Skip in -short mode
func SkipCI(t *testing.T)                             // Skip in CI environment
```

**Example:**

```go
func TestSlow(t *testing.T) {
    waffletest.SkipShort(t)

    // Long-running test...
}

func TestLocalOnly(t *testing.T) {
    waffletest.SkipCI(t)

    // Test that requires local resources...
}
```

---

## Complete Example

```go
package users_test

import (
    "testing"

    "github.com/go-chi/chi/v5"
    waffletest "github.com/dalemusser/waffle/testing"
)

func TestUserAPI(t *testing.T) {
    waffletest.Parallel(t)

    // Setup router with handlers
    r := chi.NewRouter()
    r.Get("/api/users", listUsersHandler)
    r.Post("/api/users", createUserHandler)
    r.Get("/api/users/{id}", getUserHandler)

    srv := waffletest.NewServer(t, r)
    logger := waffletest.TestLogger()

    t.Run("list users", func(t *testing.T) {
        srv.Get("/api/users").
            Do().
            StatusOK().
            ContentTypeJSON().
            JSONPathEquals("total", float64(0))
    })

    t.Run("create user", func(t *testing.T) {
        srv.Post("/api/users").
            JSON(map[string]string{
                "name":  "Alice",
                "email": "alice@example.com",
            }).
            Do().
            StatusCreated().
            JSONPathEquals("name", "Alice")
    })

    t.Run("get user not found", func(t *testing.T) {
        srv.Get("/api/users/999").
            Do().
            StatusNotFound().
            JSONPathEquals("code", "not_found")
    })

    t.Run("unauthorized without token", func(t *testing.T) {
        srv.Get("/api/admin/users").
            Do().
            StatusUnauthorized()
    })

    t.Run("authorized with token", func(t *testing.T) {
        srv.Get("/api/admin/users").
            Bearer("valid-token").
            Do().
            StatusOK()
    })
}

func TestUserHandler(t *testing.T) {
    rec := waffletest.NewRecorder(t)

    handler := NewUserHandler(nil, waffletest.TestLogger())

    t.Run("valid request", func(t *testing.T) {
        rec.Post("/users").
            JSON(map[string]string{"name": "Bob"}).
            Run(handler.Create).
            StatusCreated()
    })

    t.Run("invalid request", func(t *testing.T) {
        rec.Post("/users").
            BodyString("invalid json").
            Run(handler.Create).
            StatusBadRequest()
    })
}
```

---

## See Also

- [errors](../errors/errors.md) — Structured error handling
- [middleware](../middleware/middleware.md) — HTTP middleware
