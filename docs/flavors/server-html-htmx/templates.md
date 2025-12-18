# HTML Templates and Views in WAFFLE

*Complete guide to Go template rendering, layouts, partials, and HTMX integration.*

---

## Overview

WAFFLE provides a powerful template system built on Go's `html/template` package with added features for:

- **Embedded templates** via `go:embed` for single-binary deployment
- **Shared layouts** with content blocks for consistent page structure
- **Feature-specific templates** organized by module
- **HTMX integration** for dynamic partial updates without full page reloads
- **Template functions** for common operations

---

## Architecture

### How Templates Work in WAFFLE

```
┌─────────────────────────────────────────────────────────────────┐
│                        Template System                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────-─┐         ┌───────────────────────────┐    │
│  │  Shared (layout)  │         │   Feature Templates       │    │
│  │  resources/       │         │   features/*/templates/   │    │
│  │  templates/       │         │                           │    │
│  │  ├── layout.gohtml|         │   ├── list.gohtml         │    │
│  │  └── menu.gohtml  |         │   ├── detail.gohtml       │    │
│  └─────────────────-─┘         │   └── form.gohtml         │    │
│           │                    └───────────────────────────┘    │
│           │                               │                     │
│           └──────────┬───────────────-────┘                     │
│                      ▼                                          │
│              ┌──────────────┐                                   │
│              │ Engine.Boot()│  Compiles all templates           │
│              └──────────────┘                                   │
│                      │                                          │
│                      ▼                                          │
│              ┌──────────────────────────────────────────┐       │
│              │  Per-page clones with shared layout      │       │
│              │  Each page gets its own compiled set     │       │
│              └──────────────────────────────────────────┘       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `templates.Set` | `waffle/templates/views.go` | Describes a template set |
| `templates.Register()` | `waffle/templates/views.go` | Registers templates at startup |
| `templates.Engine` | `waffle/templates/engine.go` | Compiles and executes templates |
| `templates.Render()` | `waffle/templates/adapter.go` | Renders full pages |
| `templates.RenderSnippet()` | `waffle/templates/adapter.go` | Renders partials |
| `templates.RenderAuto()` | `waffle/templates/adapter.go` | HTMX-aware rendering |

---

## Directory Structure

### Recommended Layout

```
internal/app/
├── features/
│   ├── users/
│   │   ├── routes.go
│   │   ├── templates.go          # Embed and register
│   │   └── templates/
│   │       ├── users_list.gohtml
│   │       ├── users_detail.gohtml
│   │       └── users_form.gohtml
│   │
│   └── products/
│       ├── routes.go
│       ├── templates.go
│       └── templates/
│           ├── products_list.gohtml
│           └── products_detail.gohtml
│
└── resources/
    ├── resources.go              # Shared template registration
    └── templates/
        ├── layout.gohtml         # Main page layout
        └── menu.gohtml           # Navigation menu
```

---

## Shared Templates (Layout)

Shared templates define the overall page structure and are used by all features.

### resources/resources.go

```go
package resources

import (
    "embed"
    "sync"

    "github.com/yourusername/yourapp/waffle/templates"
)

//go:embed templates/*.gohtml
var FS embed.FS

var registerOnce sync.Once

// LoadSharedTemplates registers shared templates with WAFFLE's template engine.
// Call this from the Startup hook in bootstrap/startup.go.
func LoadSharedTemplates() {
    registerOnce.Do(func() {
        templates.Register(templates.Set{
            Name:     "shared",
            FS:       FS,
            Patterns: []string{"templates/*.gohtml"},
        })
    })
}
```

**Important**: The shared templates MUST be registered with `Name: "shared"`. The engine expects this name for the base layout.

### resources/templates/layout.gohtml

The layout defines the HTML structure and includes a `content` block that each page fills:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }} - My Application</title>

    <!-- CSS -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
    {{ template "menu" . }}

    <main id="content" class="container mt-4">
        {{ template "content" . }}
    </main>

    <footer class="container mt-5 py-3 text-muted">
        <p>&copy; 2024 My Application</p>
    </footer>

    <!-- JavaScript -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>
```

### resources/templates/menu.gohtml

The navigation menu, included in the layout:

```html
{{ define "menu" }}
<nav class="navbar navbar-expand-lg navbar-dark bg-primary">
    <div class="container">
        <a class="navbar-brand" href="/">My Application</a>
        <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
            <span class="navbar-toggler-icon"></span>
        </button>
        <div class="collapse navbar-collapse" id="navbarNav">
            <ul class="navbar-nav">
                <li class="nav-item">
                    <a class="nav-link{{ if eq .ActiveNav "home" }} active{{ end }}"
                       href="/"
                       hx-get="/"
                       hx-target="#content"
                       hx-push-url="true">Home</a>
                </li>
                <li class="nav-item">
                    <a class="nav-link{{ if eq .ActiveNav "users" }} active{{ end }}"
                       href="/users"
                       hx-get="/users"
                       hx-target="#content"
                       hx-push-url="true">Users</a>
                </li>
                <li class="nav-item">
                    <a class="nav-link{{ if eq .ActiveNav "products" }} active{{ end }}"
                       href="/products"
                       hx-get="/products"
                       hx-target="#content"
                       hx-push-url="true">Products</a>
                </li>
            </ul>
        </div>
    </div>
</nav>
{{ end }}
```

---

## Feature Templates

Each feature has its own templates in a `templates/` subdirectory.

### features/users/templates.go

```go
package users

import (
    "embed"

    "github.com/yourusername/yourapp/waffle/templates"
)

//go:embed templates/*.gohtml
var FS embed.FS

func init() {
    templates.Register(templates.Set{
        Name:     "users",
        FS:       FS,
        Patterns: []string{"templates/*.gohtml"},
    })
}
```

**Key points:**
- Uses `init()` for automatic registration when the package is imported
- The `Name` is for logging/debugging only
- Templates are embedded into the binary at compile time

### features/users/templates/users_list.gohtml

```html
{{ define "users_list" }}
{{ template "layout.gohtml" . }}
{{ end }}

{{ define "content" }}
<div class="d-flex justify-content-between align-items-center mb-4">
    <h1>Users</h1>
    <a href="/users/new" class="btn btn-primary"
       hx-get="/users/new"
       hx-target="#content"
       hx-push-url="true">Add User</a>
</div>

{{ template "users_table" . }}
{{ end }}

{{ define "users_table" }}
<div id="users-table">
    <table class="table table-striped">
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Role</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Users }}
            <tr>
                <td>{{ .Name }}</td>
                <td>{{ .Email }}</td>
                <td>{{ .Role }}</td>
                <td>
                    <a href="/users/{{ .ID }}" class="btn btn-sm btn-outline-primary"
                       hx-get="/users/{{ .ID }}"
                       hx-target="#content"
                       hx-push-url="true">View</a>
                    <a href="/users/{{ .ID }}/edit" class="btn btn-sm btn-outline-secondary"
                       hx-get="/users/{{ .ID }}/edit"
                       hx-target="#content"
                       hx-push-url="true">Edit</a>
                    <button class="btn btn-sm btn-outline-danger"
                            hx-delete="/users/{{ .ID }}"
                            hx-target="#users-table"
                            hx-swap="outerHTML"
                            hx-confirm="Delete this user?">Delete</button>
                </td>
            </tr>
            {{ else }}
            <tr>
                <td colspan="4" class="text-center text-muted">No users found</td>
            </tr>
            {{ end }}
        </tbody>
    </table>
</div>
{{ end }}
```

### features/users/templates/users_detail.gohtml

```html
{{ define "users_detail" }}
{{ template "layout.gohtml" . }}
{{ end }}

{{ define "content" }}
<div class="card">
    <div class="card-header d-flex justify-content-between align-items-center">
        <h2>{{ .User.Name }}</h2>
        <div>
            <a href="/users/{{ .User.ID }}/edit" class="btn btn-outline-secondary"
               hx-get="/users/{{ .User.ID }}/edit"
               hx-target="#content"
               hx-push-url="true">Edit</a>
            <a href="/users" class="btn btn-outline-primary"
               hx-get="/users"
               hx-target="#content"
               hx-push-url="true">Back to List</a>
        </div>
    </div>
    <div class="card-body">
        <dl class="row">
            <dt class="col-sm-3">Email</dt>
            <dd class="col-sm-9">{{ .User.Email }}</dd>

            <dt class="col-sm-3">Role</dt>
            <dd class="col-sm-9">{{ .User.Role }}</dd>

            <dt class="col-sm-3">Created</dt>
            <dd class="col-sm-9">{{ .User.CreatedAt.Format "January 2, 2006" }}</dd>
        </dl>
    </div>
</div>
{{ end }}
```

### features/users/templates/users_form.gohtml

```html
{{ define "users_form" }}
{{ template "layout.gohtml" . }}
{{ end }}

{{ define "content" }}
<div class="card">
    <div class="card-header">
        <h2>{{ if .User.ID }}Edit User{{ else }}New User{{ end }}</h2>
    </div>
    <div class="card-body">
        <form {{ if .User.ID }}
                hx-put="/users/{{ .User.ID }}"
              {{ else }}
                hx-post="/users"
              {{ end }}
              hx-target="#content"
              hx-push-url="/users">

            <div class="mb-3">
                <label for="name" class="form-label">Name</label>
                <input type="text" class="form-control" id="name" name="name"
                       value="{{ .User.Name }}" required>
            </div>

            <div class="mb-3">
                <label for="email" class="form-label">Email</label>
                <input type="email" class="form-control" id="email" name="email"
                       value="{{ .User.Email }}" required>
            </div>

            <div class="mb-3">
                <label for="role" class="form-label">Role</label>
                <select class="form-select" id="role" name="role">
                    <option value="user" {{ if eq .User.Role "user" }}selected{{ end }}>User</option>
                    <option value="admin" {{ if eq .User.Role "admin" }}selected{{ end }}>Admin</option>
                </select>
            </div>

            <div class="d-flex gap-2">
                <button type="submit" class="btn btn-primary">
                    {{ if .User.ID }}Update{{ else }}Create{{ end }}
                </button>
                <a href="/users" class="btn btn-outline-secondary"
                   hx-get="/users"
                   hx-target="#content"
                   hx-push-url="true">Cancel</a>
            </div>
        </form>
    </div>
</div>
{{ end }}
```

---

## Rendering Templates

### Handler Example

```go
package users

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/yourusername/yourapp/waffle/templates"
)

type PageData struct {
    Title     string
    ActiveNav string
    Users     []User
    User      *User
}

func listHandler(w http.ResponseWriter, r *http.Request) {
    users := getAllUsers() // Your data fetching logic

    data := PageData{
        Title:     "Users",
        ActiveNav: "users",
        Users:     users,
    }

    // HTMX-aware rendering
    templates.RenderAuto(w, r, "users_list", "users_table", "users-table", data)
}

func detailHandler(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    user := getUserByID(id)

    data := PageData{
        Title:     user.Name,
        ActiveNav: "users",
        User:      user,
    }

    templates.Render(w, r, "users_detail", data)
}
```

### Render Functions

#### `templates.Render(w, r, name, data)`

Renders a full page with layout.

```go
templates.Render(w, r, "users_list", data)
```

#### `templates.RenderSnippet(w, name, data)`

Renders just a partial template (no layout).

```go
templates.RenderSnippet(w, "users_table", data)
```

#### `templates.RenderAuto(w, r, page, snippet, targetID, data)`

HTMX-aware rendering that automatically chooses between full page and snippet:

```go
// If HTMX request targets "users-table" → renders "users_table" snippet
// If HTMX request targets "content" → renders just the content block
// Otherwise → renders full "users_list" page with layout
templates.RenderAuto(w, r, "users_list", "users_table", "users-table", data)
```

#### `templates.RenderAutoMap(w, r, page, targets, data)`

For multiple possible HTMX targets:

```go
templates.RenderAutoMap(w, r, "dashboard", map[string]string{
    "stats-panel":  "dashboard_stats",
    "recent-table": "dashboard_recent",
    "alerts-list":  "dashboard_alerts",
}, data)
```

---

## Template Functions

WAFFLE provides these built-in template functions:

| Function | Example | Result |
|----------|---------|--------|
| `urlquery` | `{{ "a b" \| urlquery }}` | `a+b` |
| `safeHTML` | `{{ .HTML \| safeHTML }}` | Renders raw HTML |
| `lower` | `{{ "ABC" \| lower }}` | `abc` |
| `upper` | `{{ "abc" \| upper }}` | `ABC` |
| `join` | `{{ .List \| join ", " }}` | `a, b, c` |
| `printf` | `{{ printf "%d items" .Count }}` | `5 items` |

### Using Functions in Templates

```html
<a href="/search?q={{ .Query | urlquery }}">Search</a>

<div class="role role-{{ .Role | lower }}">{{ .Role | upper }}</div>

<p>Tags: {{ .Tags | join ", " }}</p>

{{ .RichContent | safeHTML }}
```

---

## Template Naming Conventions

### Recommended Patterns

| Type | Pattern | Example |
|------|---------|---------|
| Page entry | `{feature}_{action}` | `users_list`, `users_detail` |
| Partial/snippet | `{feature}_{component}` | `users_table`, `users_card` |
| Shared | Descriptive name | `layout`, `menu`, `footer` |

### Why Prefixes Matter

Template names must be unique across the entire application. Using feature prefixes prevents collisions:

```
✅ users_list, products_list    (unique)
❌ list, list                    (collision!)
```

---

## Bootstrap and Initialization

### bootstrap/startup.go

```go
package bootstrap

import (
    "context"

    "github.com/yourusername/yourapp/internal/app/resources"
    "github.com/yourusername/yourapp/waffle/config"
    "github.com/yourusername/yourapp/waffle/templates"
    "go.uber.org/zap"
)

func Startup(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
    // Load shared templates first
    resources.LoadSharedTemplates()

    // Feature templates are auto-registered via init() when imported
    // in routes.go or elsewhere

    // Boot the template engine
    engine := templates.New(coreCfg.DevMode)
    if err := engine.Boot(logger); err != nil {
        return err
    }
    templates.UseEngine(engine, logger)

    return nil
}
```

---

## Common Patterns

### Conditional Content

```html
{{ if .User }}
    <p>Welcome, {{ .User.Name }}!</p>
{{ else }}
    <p>Please <a href="/login">log in</a>.</p>
{{ end }}
```

### Looping with Index

```html
{{ range $index, $item := .Items }}
    <tr class="{{ if eq (mod $index 2) 0 }}even{{ else }}odd{{ end }}">
        <td>{{ $item.Name }}</td>
    </tr>
{{ end }}
```

### Nested Templates

```html
{{ define "content" }}
<div class="row">
    <div class="col-md-8">
        {{ template "main_content" . }}
    </div>
    <div class="col-md-4">
        {{ template "sidebar" . }}
    </div>
</div>
{{ end }}

{{ define "main_content" }}
<!-- Main content here -->
{{ end }}

{{ define "sidebar" }}
<!-- Sidebar content here -->
{{ end }}
```

### Error Display

```html
{{ define "content" }}
{{ if .Error }}
<div class="alert alert-danger alert-dismissible fade show" role="alert">
    {{ .Error }}
    <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
</div>
{{ end }}

{{ if .Success }}
<div class="alert alert-success alert-dismissible fade show" role="alert">
    {{ .Success }}
    <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
</div>
{{ end }}

<!-- Rest of content -->
{{ end }}
```

---

## Troubleshooting

### "template not found"

**Cause**: Template not registered or wrong name.

**Solutions**:
1. Verify the template file exists in `templates/` directory
2. Check the template name matches the `{{ define "name" }}` directive
3. Ensure `templates.go` uses correct glob pattern
4. Verify the feature package is imported somewhere (for `init()` to run)

### "shared templates not registered"

**Cause**: `LoadSharedTemplates()` not called before `Engine.Boot()`.

**Solution**: Call `resources.LoadSharedTemplates()` in your Startup hook before booting the engine.

### Template changes not appearing

**Cause**: Templates are embedded at compile time.

**Solution**: Rebuild the application after changing `.gohtml` files.

### "content" block not rendering

**Cause**: Page template missing `{{ define "content" }}` block.

**Solution**: Ensure your page template defines the content block:

```html
{{ define "content" }}
<!-- Your page content -->
{{ end }}
```

---

## See Also

- [HTMX Integration Patterns](./htmx-integration.md)
- [Static File Serving](./static-files.md)
- [Feature Structure Examples](../feature-structure-examples.md)
- [Routes Examples](../routes-examples.md)
