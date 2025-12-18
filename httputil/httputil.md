# httputil

JSON response helpers for HTTP handlers.

## Overview

The `httputil` package provides simple, consistent functions for writing JSON responses from HTTP handlers. Use these for API endpoints that return JSON data or error responses.

## Import

```go
import "github.com/dalemusser/waffle/httputil"
```

## Quick Start

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    user, err := db.FindUser(r.Context(), userID)
    if err != nil {
        httputil.JSONError(w, http.StatusNotFound, "not_found", "User not found")
        return
    }
    httputil.WriteJSON(w, http.StatusOK, user)
}
```

## API

### ErrorResponse

**Location:** `json.go`

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}
```

Standard JSON error envelope used by `JSONError` and `JSONErrorSimple`.

### WriteJSON

**Location:** `json.go`

```go
func WriteJSON(w http.ResponseWriter, status int, v any)
```

Writes any value as a JSON response with the given status code. Sets `Content-Type: application/json` automatically.

**Example:**

```go
type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func getUser(w http.ResponseWriter, r *http.Request) {
    user := UserResponse{ID: "123", Name: "Alice", Email: "alice@example.com"}
    httputil.WriteJSON(w, http.StatusOK, user)
}
```

### JSONError

**Location:** `json.go`

```go
func JSONError(w http.ResponseWriter, status int, code, message string)
```

Writes a structured JSON error with separate error code and human-readable message.

**Example:**

```go
// Returns: {"error": "validation_failed", "message": "Email is required"}
httputil.JSONError(w, http.StatusBadRequest, "validation_failed", "Email is required")
```

### JSONErrorSimple

**Location:** `json.go`

```go
func JSONErrorSimple(w http.ResponseWriter, status int, message string)
```

Writes a simple JSON error where the message serves as the error code.

**Example:**

```go
// Returns: {"error": "unauthorized"}
httputil.JSONErrorSimple(w, http.StatusUnauthorized, "unauthorized")
```

## Patterns

### REST API Handler

```go
func createItem(w http.ResponseWriter, r *http.Request) {
    var req CreateItemRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.JSONError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
        return
    }

    if req.Name == "" {
        httputil.JSONError(w, http.StatusBadRequest, "validation_failed", "Name is required")
        return
    }

    item, err := db.CreateItem(r.Context(), req)
    if err != nil {
        httputil.JSONError(w, http.StatusInternalServerError, "internal_error", "Failed to create item")
        return
    }

    httputil.WriteJSON(w, http.StatusCreated, item)
}
```

### Consistent Error Codes

```go
// Define error codes as constants for consistency
const (
    ErrCodeNotFound       = "not_found"
    ErrCodeUnauthorized   = "unauthorized"
    ErrCodeValidation     = "validation_failed"
    ErrCodeInternalError  = "internal_error"
)

func handler(w http.ResponseWriter, r *http.Request) {
    item, err := db.FindItem(r.Context(), id)
    if errors.Is(err, db.ErrNotFound) {
        httputil.JSONError(w, http.StatusNotFound, ErrCodeNotFound, "Item not found")
        return
    }
    if err != nil {
        httputil.JSONError(w, http.StatusInternalServerError, ErrCodeInternalError, "Database error")
        return
    }
    httputil.WriteJSON(w, http.StatusOK, item)
}
```

### List Response with Metadata

```go
type ListResponse struct {
    Items      []Item `json:"items"`
    Total      int    `json:"total"`
    Page       int    `json:"page"`
    PerPage    int    `json:"per_page"`
}

func listItems(w http.ResponseWriter, r *http.Request) {
    items, total, err := db.ListItems(r.Context(), page, perPage)
    if err != nil {
        httputil.JSONError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch items")
        return
    }

    httputil.WriteJSON(w, http.StatusOK, ListResponse{
        Items:   items,
        Total:   total,
        Page:    page,
        PerPage: perPage,
    })
}
```

## See Also

- [router](../router/router.md) — HTTP routing
- [middleware](../middleware/middleware.md) — Request middleware

