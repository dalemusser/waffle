# query

HTTP query parameter extraction with trimming and length limits.

## When to Use

Use `query` when extracting query parameters from HTTP requests. It provides:
- Automatic whitespace trimming
- Length limits for search inputs (prevents unbounded strings in DB queries)
- Consistent, clean parameter handling across your application

Common scenarios:
- Extracting search/filter terms from URLs
- Reading pagination cursors
- Getting return URLs for redirects
- Any query parameter that needs sanitization

## Import

```go
import "github.com/dalemusser/waffle/pantry/query"
```

## Quick Start

```go
func listHandler(w http.ResponseWriter, r *http.Request) {
    // Get a simple query parameter (trimmed)
    status := query.Get(r, "status")

    // Get a search parameter (trimmed + length-limited to 200 chars)
    search := query.Search(r, "q")

    // Use in your query...
}
```

## API

### Get

```go
func Get(r *http.Request, key string) string
```

Returns the trimmed value of a query parameter. Returns an empty string if the parameter is missing or blank after trimming.

**Parameters:**
- `r` — the HTTP request
- `key` — the query parameter name

**Example:**

```go
// URL: /items?status=active&org=
status := query.Get(r, "status")  // "active"
org := query.Get(r, "org")        // "" (empty after trim)
missing := query.Get(r, "foo")    // "" (not present)
```

### GetMax

```go
func GetMax(r *http.Request, key string, maxLen int) string
```

Returns the trimmed value of a query parameter, truncated to `maxLen` characters. Use this when you need a custom length limit.

**Parameters:**
- `r` — the HTTP request
- `key` — the query parameter name
- `maxLen` — maximum length to return

**Example:**

```go
// URL: /items?code=ABCDEFGHIJ
code := query.GetMax(r, "code", 5)  // "ABCDE"
```

### Search

```go
func Search(r *http.Request, key string) string
```

Returns a search query parameter, trimmed and truncated to `MaxSearchLen` (200 characters). Use this for search/filter inputs that will be used in database queries.

**Parameters:**
- `r` — the HTTP request
- `key` — the query parameter name

**Example:**

```go
// URL: /users?search=john
search := query.Search(r, "search")  // "john"

// Very long input is truncated
// URL: /users?search=<200+ chars>
search := query.Search(r, "search")  // first 200 chars only
```

### MaxSearchLen

```go
const MaxSearchLen = 200
```

The default maximum length for search query parameters. This prevents unbounded user input from being passed to database queries.

## Patterns

### List Handler with Filters

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    // Search terms - use Search() for length limiting
    search := query.Search(r, "q")

    // Enum filters - use Get() for simple values
    status := query.Get(r, "status")
    role := query.Get(r, "role")

    // Pagination cursors
    after := query.Get(r, "after")
    before := query.Get(r, "before")

    // Build query with sanitized inputs...
}
```

### Preserving Return URLs

```go
func editHandler(w http.ResponseWriter, r *http.Request) {
    // Get return URL for redirect after save
    returnURL := query.Get(r, "return")

    // Pass to template for form hidden field
    data := map[string]any{
        "ReturnURL": returnURL,
    }
    templates.Render(w, "edit_form", data)
}
```

### Organization Scoping

```go
func membersHandler(w http.ResponseWriter, r *http.Request) {
    orgParam := query.Get(r, "org")

    var scopeOrg *primitive.ObjectID
    if orgParam != "" && orgParam != "all" {
        if oid, err := primitive.ObjectIDFromHex(orgParam); err == nil {
            scopeOrg = &oid
        }
    }

    // Query with org scope...
}
```

## Security

The `Search` function provides protection against:
- **Unbounded input** — Truncates to 200 characters, preventing memory issues and slow regex/LIKE queries
- **Whitespace padding** — Trims input, preventing bypass attempts with padded strings

For additional input validation, combine with the `validate` package.

## See Also

- [httpnav](../httpnav/httpnav.md) — HTTP navigation and return URL handling
- [urlutil](../urlutil/urlutil.md) — URL parsing and safe redirects
- [validate](../validate/validate.md) — Input validation
