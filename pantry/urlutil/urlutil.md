# urlutil

URL validation and manipulation utilities for HTTP applications, with a focus on security.

## When to Use

Use `urlutil` when you need to:

- Validate and sanitize redirect URLs to prevent open redirect attacks
- Validate user-provided URLs before storing them
- Manipulate query parameters on URLs

## Import

```go
import "github.com/dalemusser/waffle/pantry/urlutil"
```

## API

### SafeReturn

```go
func SafeReturn(ret, badID, fallback string) string
```

Validates and sanitizes a redirect path to prevent open redirect vulnerabilities and header injection attacks. Returns the sanitized path if valid, or `fallback` if validation fails.

**Parameters:**
- `ret` — the redirect URL to validate (typically from user input)
- `badID` — optional resource ID to exclude from the path (use `""` to skip)
- `fallback` — safe default path if validation fails

**Validation rules:**
- Must start with `/` (local path)
- Must not start with `//` (scheme-relative URL)
- Must not contain `\r` or `\n` (header injection)
- Must not contain `\` (ambiguous parsing)
- Must not contain `badID` as a path segment (if provided)
- Path is normalized via `path.Clean()`

**Example:**

```go
func deleteHandler(w http.ResponseWriter, r *http.Request) {
    itemID := chi.URLParam(r, "id")
    // ... delete the item ...

    ret := r.FormValue("return")
    // Validate return URL, excluding paths containing the deleted item's ID
    safeURL := urlutil.SafeReturn(ret, itemID, "/items")
    http.Redirect(w, r, safeURL, http.StatusSeeOther)
}
```

**Why exclude badID?**

After deleting `/items/abc123`, you don't want to redirect to `/items/abc123` (404) or `/items/abc123/edit`. The `badID` parameter catches these cases:

```go
urlutil.SafeReturn("/items/abc123", "abc123", "/items")      // returns "/items"
urlutil.SafeReturn("/items/abc123/edit", "abc123", "/items") // returns "/items"
urlutil.SafeReturn("/items", "abc123", "/items")             // returns "/items"
urlutil.SafeReturn("/dashboard", "abc123", "/items")         // returns "/dashboard"
```

### IsValidAbsHTTPURL

```go
func IsValidAbsHTTPURL(s string) bool
```

Reports whether `s` is a valid absolute HTTP or HTTPS URL suitable for storing or linking to.

**Validation rules:**
- Must not be empty
- Must not contain `\r` or `\n`
- Must parse as a valid URL
- Must have `http` or `https` scheme
- Must have a non-empty host
- Must not contain credentials (`user:pass@host`)

**Example:**

```go
func createResourceHandler(w http.ResponseWriter, r *http.Request) {
    launchURL := r.FormValue("launch_url")

    if launchURL != "" && !urlutil.IsValidAbsHTTPURL(launchURL) {
        http.Error(w, "Invalid launch URL", http.StatusBadRequest)
        return
    }

    // Safe to store launchURL in database
}
```

**Valid examples:**
```go
urlutil.IsValidAbsHTTPURL("https://example.com")           // true
urlutil.IsValidAbsHTTPURL("http://example.com/path")       // true
urlutil.IsValidAbsHTTPURL("https://example.com:8080/path") // true
```

**Invalid examples:**
```go
urlutil.IsValidAbsHTTPURL("example.com")             // false (no scheme)
urlutil.IsValidAbsHTTPURL("ftp://example.com")       // false (wrong scheme)
urlutil.IsValidAbsHTTPURL("//example.com")           // false (scheme-relative)
urlutil.IsValidAbsHTTPURL("javascript:alert(1)")     // false (wrong scheme)
urlutil.IsValidAbsHTTPURL("https://user:pass@x.com") // false (credentials)
urlutil.IsValidAbsHTTPURL("")                        // false (empty)
```

### AddOrSetQueryParams

```go
func AddOrSetQueryParams(base string, kv map[string]string) string
```

Adds or overwrites query parameters on a URL string. Empty values are skipped. Returns the original string unchanged if parsing fails.

**Example:**

```go
// Add parameters to a URL
url := urlutil.AddOrSetQueryParams("/search", map[string]string{
    "q":    "waffle",
    "page": "2",
})
// Result: "/search?page=2&q=waffle"

// Overwrite existing parameters
url := urlutil.AddOrSetQueryParams("/search?q=old&page=1", map[string]string{
    "q": "new",
})
// Result: "/search?page=1&q=new"

// Empty values are skipped
url := urlutil.AddOrSetQueryParams("/search", map[string]string{
    "q":    "waffle",
    "page": "", // skipped
})
// Result: "/search?q=waffle"
```

## Security

### Open Redirect Prevention

Open redirects allow attackers to craft URLs that redirect users to malicious sites:

```
https://yoursite.com/login?return=https://evil.com
```

`SafeReturn` prevents this by only accepting local paths:

```go
// These return the fallback "/dashboard"
urlutil.SafeReturn("https://evil.com", "", "/dashboard")
urlutil.SafeReturn("//evil.com", "", "/dashboard")
urlutil.SafeReturn("javascript:alert(1)", "", "/dashboard")

// This returns the validated path
urlutil.SafeReturn("/profile", "", "/dashboard") // returns "/profile"
```

### Header Injection Prevention

CR/LF characters can be used to inject HTTP headers:

```
/page\r\nSet-Cookie: session=hacked
```

Both `SafeReturn` and `IsValidAbsHTTPURL` reject URLs containing `\r` or `\n`.

### Path Traversal Normalization

`SafeReturn` normalizes paths using `path.Clean()`:

```go
urlutil.SafeReturn("/foo/../bar", "", "/fallback") // returns "/bar"
urlutil.SafeReturn("/foo/./bar", "", "/fallback")  // returns "/foo/bar"
```

## See Also

- [httpnav](../httpnav/httpnav.md) — Navigation helpers for resolving back URLs
