# HTMX Integration Patterns

*WAFFLE's preferred approach to dynamic frontend interactions without JavaScript frameworks.*

---

## Overview

WAFFLE embraces [HTMX](https://htmx.org/) as its primary approach for dynamic frontend interactions. HTMX allows you to build modern, interactive web applications using HTML attributes instead of writing JavaScript.

### Why HTMX?

| Benefit | Description |
|---------|-------------|
| **Simplicity** | HTML attributes instead of JavaScript frameworks |
| **Server-rendered** | Keep business logic on the server |
| **Progressive enhancement** | Works without JavaScript (links still work) |
| **Reduced complexity** | No build steps, no bundlers, no npm |
| **SEO-friendly** | Server-rendered HTML is easily indexed |

---

## Basic Setup

### Include HTMX in Your Layout

```html
<!-- resources/templates/layout.gohtml -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>

    <!-- HTMX from CDN -->
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>

    <!-- Optional: HTMX extensions -->
    <script src="https://unpkg.com/htmx.org/dist/ext/loading-states.js"></script>
</head>
<body hx-boost="true">
    {{ template "menu" . }}

    <main id="content" class="container">
        {{ template "content" . }}
    </main>
</body>
</html>
```

### Key Attributes

| Attribute | Purpose | Example |
|-----------|---------|---------|
| `hx-get` | Make GET request | `hx-get="/users"` |
| `hx-post` | Make POST request | `hx-post="/users"` |
| `hx-put` | Make PUT request | `hx-put="/users/1"` |
| `hx-delete` | Make DELETE request | `hx-delete="/users/1"` |
| `hx-target` | Where to put response | `hx-target="#content"` |
| `hx-swap` | How to swap content | `hx-swap="outerHTML"` |
| `hx-push-url` | Update browser URL | `hx-push-url="true"` |
| `hx-trigger` | What triggers request | `hx-trigger="click"` |
| `hx-confirm` | Confirmation dialog | `hx-confirm="Are you sure?"` |

---

## WAFFLE's HTMX-Aware Rendering

WAFFLE's template system automatically detects HTMX requests and renders appropriately.

### How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    Request arrives                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │ Is HX-Request header present? │
              └───────────────────────────────┘
                      │              │
                     YES             NO
                      │              │
                      ▼              ▼
         ┌─────────────────┐  ┌─────────────────────┐
         │ Check HX-Target │  │ Render full page    │
         │    header       │  │ with layout         │
         └─────────────────┘  └─────────────────────┘
                 │
        ┌────────┼────────┐
        │        │        │
        ▼        ▼        ▼
   "content"  Mapped   Other
        │     target     │
        │        │       │
        ▼        ▼       ▼
   Render    Render   Render
   content   snippet  full page
   block     only
```

### RenderAuto Function

```go
// Handler that works for both full page loads and HTMX partial updates
func listHandler(w http.ResponseWriter, r *http.Request) {
    users := getAllUsers()

    data := PageData{
        Title: "Users",
        Users: users,
    }

    // Automatically handles:
    // - Full page request → renders "users_list" with layout
    // - HTMX to #content → renders just the content block
    // - HTMX to #users-table → renders just "users_table" snippet
    templates.RenderAuto(w, r, "users_list", "users_table", "users-table", data)
}
```

### RenderAutoMap for Multiple Targets

```go
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    data := getDashboardData()

    // Multiple possible HTMX targets
    templates.RenderAutoMap(w, r, "dashboard", map[string]string{
        "stats-panel":   "dashboard_stats",
        "activity-feed": "dashboard_activity",
        "alerts-list":   "dashboard_alerts",
    }, data)
}
```

---

## Common Patterns

### 1. Navigation with Content Swap

Replace only the main content area when navigating:

```html
<!-- menu.gohtml -->
<nav>
    <a href="/users"
       hx-get="/users"
       hx-target="#content"
       hx-push-url="true">Users</a>

    <a href="/products"
       hx-get="/products"
       hx-target="#content"
       hx-push-url="true">Products</a>
</nav>
```

**Benefits**:
- Navigation updates URL (back button works)
- Only content area reloads
- Falls back to normal link if JavaScript disabled

### 2. Table with Inline Actions

Update just the table after delete/edit:

```html
{{ define "users_table" }}
<div id="users-table">
    <table class="table">
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Users }}
            <tr>
                <td>{{ .Name }}</td>
                <td>{{ .Email }}</td>
                <td>
                    <button hx-delete="/users/{{ .ID }}"
                            hx-target="#users-table"
                            hx-swap="outerHTML"
                            hx-confirm="Delete {{ .Name }}?"
                            class="btn btn-danger btn-sm">
                        Delete
                    </button>
                </td>
            </tr>
            {{ end }}
        </tbody>
    </table>
</div>
{{ end }}
```

**Handler**:

```go
func deleteHandler(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    deleteUser(id)

    // Return updated table
    users := getAllUsers()
    templates.RenderSnippet(w, "users_table", PageData{Users: users})
}
```

### 3. Form Submission

Submit form and update content:

```html
{{ define "content" }}
<form hx-post="/users"
      hx-target="#content"
      hx-push-url="/users">

    <div class="mb-3">
        <label for="name">Name</label>
        <input type="text" id="name" name="name" class="form-control" required>
    </div>

    <div class="mb-3">
        <label for="email">Email</label>
        <input type="email" id="email" name="email" class="form-control" required>
    </div>

    <button type="submit" class="btn btn-primary">Create</button>
</form>
{{ end }}
```

**Handler**:

```go
func createHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    user := User{
        Name:  r.FormValue("name"),
        Email: r.FormValue("email"),
    }

    if err := createUser(&user); err != nil {
        // Re-render form with error
        data := PageData{Error: err.Error(), User: &user}
        templates.RenderAuto(w, r, "users_form", "", "", data)
        return
    }

    // Redirect to list (works for both HTMX and regular requests)
    users := getAllUsers()
    data := PageData{Users: users, Success: "User created!"}
    templates.RenderAuto(w, r, "users_list", "users_table", "users-table", data)
}
```

### 4. Search/Filter

Live search with debouncing:

```html
{{ define "content" }}
<div class="mb-3">
    <input type="search"
           name="q"
           placeholder="Search users..."
           hx-get="/users/search"
           hx-trigger="input changed delay:300ms"
           hx-target="#users-table"
           hx-swap="outerHTML"
           class="form-control">
</div>

{{ template "users_table" . }}
{{ end }}
```

**Handler**:

```go
func searchHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    users := searchUsers(query)

    templates.RenderSnippet(w, "users_table", PageData{Users: users})
}
```

### 5. Pagination

Load more items:

```html
{{ define "users_table" }}
<div id="users-table">
    <table class="table">
        <!-- table content -->
    </table>

    {{ if .HasMore }}
    <button hx-get="/users?page={{ .NextPage }}"
            hx-target="#users-table"
            hx-swap="outerHTML"
            class="btn btn-outline-primary">
        Load More
    </button>
    {{ end }}
</div>
{{ end }}
```

### 6. Modal Forms

Load form into modal:

```html
<!-- Button to open modal -->
<button hx-get="/users/new"
        hx-target="#modal-content"
        hx-trigger="click"
        data-bs-toggle="modal"
        data-bs-target="#formModal"
        class="btn btn-primary">
    Add User
</button>

<!-- Modal structure -->
<div class="modal fade" id="formModal" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content" id="modal-content">
            <!-- Form loaded here via HTMX -->
        </div>
    </div>
</div>
```

**Modal form template**:

```html
{{ define "users_modal_form" }}
<div class="modal-header">
    <h5 class="modal-title">New User</h5>
    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
</div>
<form hx-post="/users"
      hx-target="#users-table"
      hx-swap="outerHTML">
    <div class="modal-body">
        <div class="mb-3">
            <label for="name">Name</label>
            <input type="text" id="name" name="name" class="form-control" required>
        </div>
        <div class="mb-3">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" class="form-control" required>
        </div>
    </div>
    <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
        <button type="submit" class="btn btn-primary">Create</button>
    </div>
</form>
{{ end }}
```

### 7. Inline Editing

Edit in place:

```html
{{ define "user_row" }}
<tr id="user-{{ .ID }}" hx-target="this" hx-swap="outerHTML">
    <td>{{ .Name }}</td>
    <td>{{ .Email }}</td>
    <td>
        <button hx-get="/users/{{ .ID }}/edit-row"
                class="btn btn-sm btn-outline-secondary">
            Edit
        </button>
    </td>
</tr>
{{ end }}

{{ define "user_row_edit" }}
<tr id="user-{{ .ID }}" hx-target="this" hx-swap="outerHTML">
    <td>
        <input type="text" name="name" value="{{ .Name }}" class="form-control form-control-sm">
    </td>
    <td>
        <input type="email" name="email" value="{{ .Email }}" class="form-control form-control-sm">
    </td>
    <td>
        <button hx-put="/users/{{ .ID }}"
                hx-include="closest tr"
                class="btn btn-sm btn-success">Save</button>
        <button hx-get="/users/{{ .ID }}/row"
                class="btn btn-sm btn-outline-secondary">Cancel</button>
    </td>
</tr>
{{ end }}
```

---

## Loading States

Show loading indicators during requests:

```html
<!-- Using htmx-indicator class -->
<button hx-get="/slow-endpoint"
        hx-target="#result"
        class="btn btn-primary">
    <span class="htmx-indicator spinner-border spinner-border-sm"></span>
    Load Data
</button>

<!-- CSS -->
<style>
    .htmx-indicator {
        display: none;
    }
    .htmx-request .htmx-indicator {
        display: inline-block;
    }
    .htmx-request.htmx-indicator {
        display: inline-block;
    }
</style>
```

---

## Error Handling

### Client-Side Error Display

```html
<!-- Element to show errors -->
<div id="error-container"></div>

<form hx-post="/users"
      hx-target="#content"
      hx-target-error="#error-container">
    <!-- form fields -->
</form>
```

### Server-Side Error Response

```go
func createHandler(w http.ResponseWriter, r *http.Request) {
    // ... validation ...

    if err != nil {
        // For HTMX requests, you can return error HTML
        w.WriteHeader(http.StatusBadRequest)
        templates.RenderSnippet(w, "error_alert", map[string]string{
            "Message": err.Error(),
        })
        return
    }

    // Success...
}
```

### Error Alert Template

```html
{{ define "error_alert" }}
<div class="alert alert-danger alert-dismissible fade show" role="alert">
    {{ .Message }}
    <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
</div>
{{ end }}
```

---

## Response Headers

HTMX supports special response headers for advanced control:

### HX-Redirect

Force a full page redirect:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // After successful action, redirect
    w.Header().Set("HX-Redirect", "/dashboard")
    w.WriteHeader(http.StatusOK)
}
```

### HX-Refresh

Trigger a full page refresh:

```go
w.Header().Set("HX-Refresh", "true")
```

### HX-Trigger

Trigger client-side events:

```go
// Trigger a custom event
w.Header().Set("HX-Trigger", "userCreated")

// Trigger with data
w.Header().Set("HX-Trigger", `{"showMessage": {"level": "success", "message": "User created!"}}`)
```

### HX-Reswap

Change the swap method:

```go
w.Header().Set("HX-Reswap", "innerHTML")
```

---

## Best Practices

### 1. Progressive Enhancement

Always provide a fallback:

```html
<!-- Works with and without JavaScript -->
<a href="/users"
   hx-get="/users"
   hx-target="#content"
   hx-push-url="true">Users</a>
```

### 2. Use hx-boost for Simple Navigation

Enable automatic AJAX for all links and forms:

```html
<body hx-boost="true">
    <!-- All links and forms automatically use HTMX -->
</body>
```

### 3. Target Specific Elements

Be precise with targets to minimize re-rendering:

```html
<!-- ✅ Good: Only update what changed -->
<button hx-delete="/users/1" hx-target="#user-row-1">Delete</button>

<!-- ❌ Avoid: Re-renders more than necessary -->
<button hx-delete="/users/1" hx-target="#content">Delete</button>
```

### 4. Use Appropriate Swap Methods

| Swap | Use When |
|------|----------|
| `innerHTML` | Replace children only |
| `outerHTML` | Replace entire element |
| `beforebegin` | Insert before element |
| `afterbegin` | Insert as first child |
| `beforeend` | Insert as last child |
| `afterend` | Insert after element |
| `delete` | Remove target element |
| `none` | Don't swap (use for side effects) |

### 5. Debounce User Input

Prevent excessive requests:

```html
<input type="search"
       hx-get="/search"
       hx-trigger="input changed delay:300ms, search"
       hx-target="#results">
```

---

## Debugging

### Enable HTMX Logging

Add `htmx.logAll()` to enable verbose console logging of all HTMX events:

```html
<script>
    htmx.logAll();
</script>
```

This logs every HTMX event to the browser console, including:
- **Request lifecycle**: `htmx:beforeRequest`, `htmx:afterRequest`, `htmx:beforeSend`
- **Response handling**: `htmx:beforeSwap`, `htmx:afterSwap`, `htmx:afterSettle`
- **Configuration**: `htmx:configRequest` (shows headers and parameters being sent)
- **Errors**: `htmx:responseError`, `htmx:sendError`, `htmx:timeout`

**Example output** when clicking an HTMX-enabled button:

```
htmx:configRequest <button hx-get="/users"> {parameters: {}, headers: {...}, target: div#content}
htmx:beforeRequest <button hx-get="/users"> {xhr: XMLHttpRequest, target: div#content}
htmx:afterRequest <button hx-get="/users"> {successful: true, xhr: XMLHttpRequest}
htmx:beforeSwap <button hx-get="/users"> {xhr: XMLHttpRequest, target: div#content}
htmx:afterSwap <button hx-get="/users"> {target: div#content}
htmx:afterSettle <button hx-get="/users"> {target: div#content}
```

**When to use**:
- Debugging why a request isn't firing
- Understanding swap behavior
- Checking what headers/parameters are being sent
- Diagnosing why content isn't appearing where expected

**Disabling**: Remove the script or set `htmx.logger = null;`

**Note**: Remove `htmx.logAll()` before deploying to production—it's verbose and can impact performance.

### Browser DevTools

1. Open Network tab
2. Filter by XHR/Fetch
3. Inspect HTMX requests and responses
4. Check for HX-* headers

### Check Request Type in Handler

```go
func handler(w http.ResponseWriter, r *http.Request) {
    isHTMX := r.Header.Get("HX-Request") != ""
    target := r.Header.Get("HX-Target")

    log.Printf("HTMX: %v, Target: %s", isHTMX, target)
}
```

---

## See Also

- [Templates and Views](./templates-and-views.md)
- [Static File Serving](./static-files.md)
- [HTMX Documentation](https://htmx.org/docs/)
- [HTMX Examples](https://htmx.org/examples/)
