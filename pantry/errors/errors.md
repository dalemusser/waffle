# errors

Structured error handling for WAFFLE applications.

## Overview

The `errors` package provides consistent error handling with error codes, HTTP status mapping, and JSON serialization. It wraps Go's standard `errors` package while adding structure for API responses.

## Import

```go
import "github.com/dalemusser/waffle/errors"
```

---

## Error Type

**Location:** `errors.go`

```go
type Error struct {
    Code    string         `json:"code"`              // Machine-readable code
    Message string         `json:"message"`           // Human-readable message
    Status  int            `json:"-"`                 // HTTP status (not in JSON)
    Details map[string]any `json:"details,omitempty"` // Additional context
    Err     error          `json:"-"`                 // Underlying error
}
```

The `Error` type implements the standard `error` interface and supports `errors.Is` and `errors.As` via `Unwrap()`.

---

## Creating Errors

### New

**Location:** `errors.go`

```go
func New(code, message string, status int) *Error
```

Creates a new error with code, message, and HTTP status.

**Example:**

```go
err := errors.New("invalid_token", "the authentication token is invalid", http.StatusUnauthorized)
```

### Wrap

**Location:** `errors.go`

```go
func Wrap(err error, code, message string, status int) *Error
```

Creates a new error wrapping an existing error.

**Example:**

```go
user, err := db.FindUser(id)
if err != nil {
    return errors.Wrap(err, "user_not_found", "user not found", http.StatusNotFound)
}
```

### From

**Location:** `errors.go`

```go
func From(err error) *Error
```

Extracts an `*Error` from err if possible, or wraps it as an internal error.

**Example:**

```go
e := errors.From(err)
log.Printf("status=%d code=%s", e.Status, e.Code)
```

---

## Convenience Constructors

All constructors take a message and return an `*Error` with the appropriate code and HTTP status.

| Function | Code | HTTP Status |
|----------|------|-------------|
| `BadRequest(msg)` | `bad_request` | 400 |
| `Unauthorized(msg)` | `unauthorized` | 401 |
| `Forbidden(msg)` | `forbidden` | 403 |
| `NotFound(msg)` | `not_found` | 404 |
| `MethodNotAllowed(msg)` | `method_not_allowed` | 405 |
| `Conflict(msg)` | `conflict` | 409 |
| `Gone(msg)` | `gone` | 410 |
| `UnprocessableEntity(msg)` | `unprocessable_entity` | 422 |
| `TooManyRequests(msg)` | `too_many_requests` | 429 |
| `Internal(msg)` | `internal_error` | 500 |
| `NotImplemented(msg)` | `not_implemented` | 501 |
| `ServiceUnavailable(msg)` | `service_unavailable` | 503 |
| `Timeout(msg)` | `timeout` | 504 |

**Semantic constructors:**

| Function | Code | HTTP Status |
|----------|------|-------------|
| `Validation(msg)` | `validation_failed` | 400 |
| `InvalidInput(msg)` | `invalid_input` | 400 |
| `AlreadyExists(msg)` | `already_exists` | 409 |
| `AuthenticationFailed(msg)` | `authentication_failed` | 401 |
| `PermissionDenied(msg)` | `permission_denied` | 403 |

**Example:**

```go
func GetUser(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id")
    if id == "" {
        return errors.BadRequest("user ID is required")
    }

    user, err := db.FindUser(id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return errors.NotFound("user not found")
        }
        return errors.Internal("failed to fetch user").Wrap(err)
    }

    return json.NewEncoder(w).Encode(user)
}
```

---

## Error Methods

### WithDetails

```go
func (e *Error) WithDetails(details map[string]any) *Error
```

Adds a details map to the error.

**Example:**

```go
return errors.BadRequest("invalid request").WithDetails(map[string]any{
    "field": "email",
    "reason": "invalid format",
})
```

### WithDetail

```go
func (e *Error) WithDetail(key string, value any) *Error
```

Adds a single detail to the error.

**Example:**

```go
return errors.NotFound("resource not found").WithDetail("id", resourceID)
```

### Wrap

```go
func (e *Error) Wrap(err error) *Error
```

Wraps an underlying error for unwrapping later.

**Example:**

```go
return errors.Internal("database error").Wrap(dbErr)
```

### HTTPStatus

```go
func (e *Error) HTTPStatus() int
```

Returns the HTTP status code (defaults to 500 if not set).

---

## HTTP Helpers

### Write

**Location:** `http.go`

```go
func Write(w http.ResponseWriter, err error)
```

Writes an error as JSON to the response with the appropriate status code.

**Example:**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := getUser(r)
    if err != nil {
        errors.Write(w, err)
        return
    }
    json.NewEncoder(w).Encode(user)
}
```

**Response format:**

```json
{
    "error": {
        "code": "not_found",
        "message": "user not found",
        "details": {
            "id": "123"
        }
    }
}
```

### WriteWithLogger

**Location:** `http.go`

```go
func WriteWithLogger(w http.ResponseWriter, err error, logger *zap.Logger)
```

Writes an error and logs internal errors (status >= 500).

**Example:**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := processRequest(r); err != nil {
        errors.WriteWithLogger(w, err, logger)
        return
    }
}
```

---

## Error Handler Pattern

### ErrorHandlerFunc

**Location:** `http.go`

```go
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error
```

A handler function that returns an error instead of writing it directly.

### WrapHandler

**Location:** `http.go`

```go
func WrapHandler(h ErrorHandlerFunc) http.HandlerFunc
```

Converts an `ErrorHandlerFunc` to a standard `http.HandlerFunc`.

**Example:**

```go
func getUser(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id")
    if id == "" {
        return errors.BadRequest("user ID is required")
    }

    user, err := db.FindUser(id)
    if err != nil {
        return errors.NotFound("user not found")
    }

    return json.NewEncoder(w).Encode(user)
}

// In routes
r.Get("/users/{id}", errors.WrapHandler(getUser))
```

### WrapHandlerWithLogger

**Location:** `http.go`

```go
func WrapHandlerWithLogger(h ErrorHandlerFunc, logger *zap.Logger) http.HandlerFunc
```

Like `WrapHandler` but logs internal errors.

---

## Validation Errors

### ValidationErrors

**Location:** `errors.go`

```go
type ValidationErrors struct {
    Errors []FieldError `json:"errors"`
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"`
}
```

For collecting multiple field-level validation errors.

### NewValidationErrors

**Location:** `errors.go`

```go
func NewValidationErrors() *ValidationErrors
```

Creates a new ValidationErrors collector.

**Example:**

```go
func validateUser(u *User) error {
    v := errors.NewValidationErrors()

    if u.Email == "" {
        v.Add("email", "email is required")
    } else if !isValidEmail(u.Email) {
        v.AddWithCode("email", "invalid email format", "invalid_format")
    }

    if u.Name == "" {
        v.Add("name", "name is required")
    }

    if len(u.Password) < 8 {
        v.Add("password", "password must be at least 8 characters")
    }

    return v.ToError() // Returns nil if no errors
}
```

**Response format:**

```json
{
    "error": {
        "code": "validation_failed",
        "message": "validation failed",
        "details": {
            "errors": [
                {"field": "email", "message": "email is required"},
                {"field": "password", "message": "password must be at least 8 characters"}
            ]
        }
    }
}
```

---

## Standard Library Compatibility

The package re-exports functions from Go's `errors` package:

```go
func Is(err, target error) bool
func As(err error, target any) bool
func Join(errs ...error) error
```

**Example:**

```go
if errors.Is(err, sql.ErrNoRows) {
    return errors.NotFound("record not found")
}

var appErr *errors.Error
if errors.As(err, &appErr) {
    log.Printf("code=%s status=%d", appErr.Code, appErr.Status)
}
```

---

## Error Codes

Pre-defined error code constants:

```go
const (
    CodeBadRequest           = "bad_request"
    CodeUnauthorized         = "unauthorized"
    CodeForbidden            = "forbidden"
    CodeNotFound             = "not_found"
    CodeMethodNotAllowed     = "method_not_allowed"
    CodeConflict             = "conflict"
    CodeGone                 = "gone"
    CodeUnprocessableEntity  = "unprocessable_entity"
    CodeTooManyRequests      = "too_many_requests"
    CodeInternalError        = "internal_error"
    CodeNotImplemented       = "not_implemented"
    CodeServiceUnavailable   = "service_unavailable"
    CodeTimeout              = "timeout"
    CodeValidationFailed     = "validation_failed"
    CodeAlreadyExists        = "already_exists"
    CodeInvalidInput         = "invalid_input"
    CodeAuthenticationFailed = "authentication_failed"
    CodePermissionDenied     = "permission_denied"
)
```

---

## WAFFLE Integration

### Handler Pattern

```go
// internal/features/users/handlers.go
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id")

    user, err := h.repo.Find(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return errors.NotFound("user not found").WithDetail("id", id)
        }
        return errors.Internal("failed to fetch user").Wrap(err)
    }

    return httputil.JSON(w, http.StatusOK, user)
}

// internal/features/users/routes.go
func (h *Handler) Routes(r chi.Router) {
    r.Get("/{id}", errors.WrapHandlerWithLogger(h.GetUser, h.logger))
}
```

### Middleware Integration

```go
// Custom error handler middleware
func SetupRoutes(r chi.Router, logger *zap.Logger) {
    // Panic recovery with error response
    r.Use(errors.Middleware(logger))

    // Custom 404 handler
    r.NotFound(errors.NotFoundHandler().ServeHTTP)

    // Custom 405 handler
    r.MethodNotAllowed(errors.MethodNotAllowedHandler().ServeHTTP)
}
```

### Repository Pattern

```go
// internal/features/users/repository.go
var ErrUserNotFound = errors.New("user_not_found", "user not found", http.StatusNotFound)

func (r *Repository) Find(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", id).Scan(&user)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, errors.Wrap(err, "database_error", "failed to query user", http.StatusInternalServerError)
    }
    return &user, nil
}
```

---

## Best Practices

1. **Use semantic constructors** — `NotFound()`, `BadRequest()`, etc. are clearer than `New()`

2. **Wrap underlying errors** — Preserve the error chain for debugging:
   ```go
   return errors.Internal("operation failed").Wrap(err)
   ```

3. **Add context with details** — Help clients understand what went wrong:
   ```go
   return errors.NotFound("resource not found").WithDetail("id", id)
   ```

4. **Use validation errors for forms** — Collect all field errors before responding:
   ```go
   v := errors.NewValidationErrors()
   v.Add("email", "required")
   v.Add("password", "too short")
   return v.ToError()
   ```

5. **Don't expose internal details** — Use generic messages for 500 errors:
   ```go
   // Bad: exposes SQL
   return errors.Internal(err.Error())

   // Good: generic message, wrap for logging
   return errors.Internal("database operation failed").Wrap(err)
   ```

6. **Log internal errors** — Use `WriteWithLogger` or check status:
   ```go
   if e.Status >= 500 {
       logger.Error("internal error", zap.Error(e.Err))
   }
   ```

---

## See Also

- [httputil](../httputil/httputil.md) — JSON response helpers
- [middleware](../middleware/middleware.md) — HTTP middleware
- [logging](../logging/logging.md) — Structured logging
