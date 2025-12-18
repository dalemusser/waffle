# httpnav

HTTP navigation helpers for resolving safe "back" URLs in web applications.

## When to Use

Use `httpnav` when you need to redirect users back to a previous page after an action (form submission, login, delete, etc.) while ensuring the redirect target is safe and local.

Common scenarios:
- Redirect after login to the page the user was trying to access
- Redirect after form submission back to the list view
- Redirect after delete back to the parent page
- Preserving navigation context across multi-step flows

## Import

```go
import "github.com/dalemusser/waffle/pantry/httpnav"
```

## Quick Start

```go
func createHandler(w http.ResponseWriter, r *http.Request) {
    // ... create the item ...

    // Redirect back to where the user came from, or /items as fallback
    backURL := httpnav.ResolveBackURL(r, "/items")
    http.Redirect(w, r, backURL, http.StatusSeeOther)
}
```

## API

### ResolveBackURL

```go
func ResolveBackURL(r *http.Request, fallback string) string
```

Returns a safe local URL to redirect the user back to. Checks multiple sources in priority order:

1. **Query parameter** `?return=/path` — explicit return URL
2. **Form field** `return` — from POST/PUT body (urlencoded or multipart)
3. **Referer header** — if same host and local path
4. **Fallback** — the provided default

All sources are validated to ensure the URL is local (starts with `/`) and same-origin. External URLs, scheme-relative URLs (`//example.com`), and malformed inputs are rejected.

**Parameters:**
- `r` — the HTTP request
- `fallback` — safe local path to use if no valid return URL is found (e.g., `"/items"`)

**Example:**

```go
// User visits /items/new?return=/items
// After form submission, redirect back to /items
backURL := httpnav.ResolveBackURL(r, "/dashboard")
http.Redirect(w, r, backURL, http.StatusSeeOther)
```

### HasExplicitReturn

```go
func HasExplicitReturn(r *http.Request) bool
```

Reports whether the request has an explicit `?return=` query parameter with a valid local path. Useful for conditional logic based on whether a return URL was explicitly provided.

**Example:**

```go
if httpnav.HasExplicitReturn(r) {
    // User came from somewhere specific, preserve context
} else {
    // User navigated directly, use default behavior
}
```

### CurrentPath

```go
func CurrentPath(r *http.Request) string
```

Returns the current request path including query string. Useful for generating return URLs to pass to other pages.

**Example:**

```go
// Generate a link that preserves the return context
loginURL := "/login?return=" + url.QueryEscape(httpnav.CurrentPath(r))
```

## Patterns

### Login Flow with Return URL

```go
// In protected handler - redirect to login with return URL
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    if !isAuthenticated(r) {
        returnURL := url.QueryEscape(httpnav.CurrentPath(r))
        http.Redirect(w, r, "/login?return="+returnURL, http.StatusSeeOther)
        return
    }
    // ... handle request ...
}

// In login handler - redirect back after successful login
func loginHandler(w http.ResponseWriter, r *http.Request) {
    // ... authenticate user ...

    backURL := httpnav.ResolveBackURL(r, "/dashboard")
    http.Redirect(w, r, backURL, http.StatusSeeOther)
}
```

### Form with Hidden Return Field

When the return URL needs to survive a form POST:

```html
<form method="POST" action="/items/create">
    <input type="hidden" name="return" value="{{ .ReturnURL }}">
    <!-- form fields -->
    <button type="submit">Create</button>
</form>
```

```go
func showCreateForm(w http.ResponseWriter, r *http.Request) {
    data := map[string]any{
        "ReturnURL": httpnav.ResolveBackURL(r, "/items"),
    }
    templates.Render(w, "create_form", data)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
    // ... create item ...

    // ResolveBackURL checks form fields, so this works
    backURL := httpnav.ResolveBackURL(r, "/items")
    http.Redirect(w, r, backURL, http.StatusSeeOther)
}
```

### Delete with Safe Redirect

After deleting a resource, redirect somewhere sensible:

```go
func deleteHandler(w http.ResponseWriter, r *http.Request) {
    itemID := chi.URLParam(r, "id")

    // ... delete the item ...

    // Don't redirect to the deleted item's page
    // ResolveBackURL will use Referer if available, or fallback
    backURL := httpnav.ResolveBackURL(r, "/items")
    http.Redirect(w, r, backURL, http.StatusSeeOther)
}
```

For additional safety when deleting, combine with `httputil.SafeReturn` to exclude the deleted resource's URL:

```go
import "github.com/dalemusser/waffle/pantry/httputil"

func deleteHandler(w http.ResponseWriter, r *http.Request) {
    itemID := chi.URLParam(r, "id")
    // ... delete ...

    // Exclude any URL containing the deleted item's ID
    ret := r.FormValue("return")
    backURL := httputil.SafeReturn(ret, itemID, "/items")
    http.Redirect(w, r, backURL, http.StatusSeeOther)
}
```

## Security

`ResolveBackURL` validates all inputs to prevent open redirect vulnerabilities:

- Only accepts paths starting with `/`
- Rejects scheme-relative URLs (`//evil.com`)
- Rejects absolute URLs (`https://evil.com`)
- Validates Referer is same-host before using
- Falls back to safe default on any validation failure

## See Also

- [httputil](../httputil/httputil.md) — URL validation and safe redirects
