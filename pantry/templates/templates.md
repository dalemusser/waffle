# templates

HTML template engine with layout support and HTMX integration.

## Overview

The `templates` package provides a template engine built on Go's `html/template` with support for shared layouts, per-feature template sets, and automatic HTMX partial rendering. Register template sets from feature packages, boot the engine at startup, and render full pages or snippets from handlers.

## Import

```go
import "github.com/dalemusser/waffle/templates"
```

## Quick Start

### 1. Register Templates (Feature Package)

```go
// features/users/templates.go
package users

import (
    "embed"
    "github.com/dalemusser/waffle/templates"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

func init() {
    templates.Register(templates.Set{
        Name:     "users",
        FS:       templateFS,
        Patterns: []string{"templates/*.gohtml"},
    })
}
```

### 2. Boot Engine (Startup)

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    // Create and boot template engine
    engine := templates.New(core.Env != "prod")
    if err := engine.Boot(logger); err != nil {
        return nil, fmt.Errorf("template boot: %w", err)
    }
    templates.UseEngine(engine, logger)

    // ... routes
}
```

### 3. Render in Handlers

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    users, _ := db.ListUsers(r.Context())
    templates.Render(w, r, "users_list", map[string]any{
        "Users": users,
    })
}
```

## Architecture

### Template Sets

Templates are organized into **sets**:

- **shared** — Required. Contains layouts and common partials used by all pages.
- **feature sets** — One per feature (e.g., "users", "admin"). Contains page templates.

Each page template defines a `content` block that gets inserted into the shared layout.

### Compilation Strategy

At boot time, the engine:

1. Parses the "shared" set as the base template
2. For each page file in feature sets, clones the base and parses all files
3. Indexes templates by their defined names for fast lookup

This ensures each page has its own compiled template with the correct `content` block.

## API

### Set

**Location:** `views.go`

```go
type Set struct {
    Name     string   // For logging (e.g., "shared", "users")
    FS       fs.FS    // Embedded filesystem
    Patterns []string // Glob patterns (e.g., ["templates/*.gohtml"])
}
```

### Register

**Location:** `views.go`

```go
func Register(s Set)
```

Registers a template set for loading at boot time. Typically called from a feature package's `init()`.

### Engine

**Location:** `engine.go`

```go
type Engine struct { ... }

func New(dev bool) *Engine
func (e *Engine) Boot(logger *zap.Logger) error
func (e *Engine) Render(w Writer, r Request, name string, data any) error
func (e *Engine) RenderSnippet(w Writer, name string, data any) error
func (e *Engine) RenderContent(w Writer, entry string, data any) error
```

### New

**Location:** `engine.go`

```go
func New(dev bool) *Engine
```

Creates a new template engine. Set `dev=true` for development mode (future: hot reload support).

### Boot

**Location:** `engine.go`

```go
func (e *Engine) Boot(logger *zap.Logger) error
```

Compiles all registered template sets. Must be called before rendering. Returns an error if the "shared" set is missing or templates fail to parse.

### UseEngine

**Location:** `adapter.go`

```go
func UseEngine(e *Engine, l *zap.Logger)
```

Installs the engine for use by the package-level render functions.

### Render

**Location:** `adapter.go`

```go
func Render(w http.ResponseWriter, r *http.Request, name string, data any)
```

Renders a full page by template name. The template should call the shared layout.

### RenderSnippet

**Location:** `adapter.go`

```go
func RenderSnippet(w http.ResponseWriter, name string, data any)
```

Renders a partial template (e.g., a table or form fragment) for HTMX responses.

### RenderAuto

**Location:** `adapter.go`

```go
func RenderAuto(w http.ResponseWriter, r *http.Request, page, tableSnippet, targetID string, data any)
```

Automatically selects between full page and snippet based on HTMX headers. If `HX-Target` matches `targetID`, renders `tableSnippet`. Otherwise renders full `page`.

### RenderAutoMap

**Location:** `adapter.go`

```go
func RenderAutoMap(w http.ResponseWriter, r *http.Request, page string, targets map[string]string, data any)
```

Like `RenderAuto` but supports multiple target-to-snippet mappings.

**Behavior:**
1. If `HX-Request` header present and `HX-Target` matches a key in `targets`, render that snippet
2. If `HX-Target` is "content", render just the content block (no layout)
3. Otherwise, render full page with layout

## Template Functions

**Location:** `funcs.go`

Built-in functions available in all templates:

| Function | Description | Example |
|----------|-------------|---------|
| `urlquery` | URL-encode a string | `{{ .Query \| urlquery }}` |
| `safeHTML` | Mark string as safe HTML | `{{ .HTML \| safeHTML }}` |
| `lower` | Convert to lowercase | `{{ .Name \| lower }}` |
| `upper` | Convert to uppercase | `{{ .Name \| upper }}` |
| `join` | Join slice with separator | `{{ .Tags \| join ", " }}` |
| `printf` | Formatted string | `{{ printf "%d items" .Count }}` |

## File Structure

```
myapp/
├── shared/
│   └── templates/
│       ├── layout.gohtml      # Main layout
│       ├── nav.gohtml         # Navigation partial
│       └── footer.gohtml      # Footer partial
├── features/
│   └── users/
│       ├── templates/
│       │   ├── list.gohtml    # Users list page
│       │   ├── edit.gohtml    # User edit page
│       │   └── table.gohtml   # Users table snippet
│       ├── templates.go       # Register call
│       └── handlers.go
```

## Template Examples

### Shared Layout

```html
{{/* shared/templates/layout.gohtml */}}
{{ define "layout" }}
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Title }}</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
    {{ template "nav" . }}
    <main id="content">
        {{ template "content" . }}
    </main>
    {{ template "footer" . }}
</body>
</html>
{{ end }}
```

### Page Template

```html
{{/* features/users/templates/list.gohtml */}}
{{ define "users_list" }}
{{ template "layout" . }}
{{ end }}

{{ define "content" }}
<h1>Users</h1>
<div id="users-table">
    {{ template "users_table" . }}
</div>
{{ end }}
```

### Snippet Template

```html
{{/* features/users/templates/table.gohtml */}}
{{ define "users_table" }}
<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Email</th>
        </tr>
    </thead>
    <tbody>
        {{ range .Users }}
        <tr>
            <td>{{ .Name }}</td>
            <td>{{ .Email }}</td>
        </tr>
        {{ end }}
    </tbody>
</table>
{{ end }}
```

## Patterns

### HTMX Table Refresh

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    users, _ := db.ListUsers(r.Context())
    data := map[string]any{"Users": users}

    // Full page on normal request, just table on HTMX targeting #users-table
    templates.RenderAuto(w, r, "users_list", "users_table", "users-table", data)
}
```

```html
<!-- Button that refreshes just the table -->
<button hx-get="/users" hx-target="#users-table">Refresh</button>
```

### Multiple HTMX Targets

```go
func dashboard(w http.ResponseWriter, r *http.Request) {
    data := map[string]any{
        "Stats":   getStats(),
        "Recent":  getRecent(),
        "Alerts":  getAlerts(),
    }

    templates.RenderAutoMap(w, r, "dashboard", map[string]string{
        "stats-panel":  "dashboard_stats",
        "recent-panel": "dashboard_recent",
        "alerts-panel": "dashboard_alerts",
    }, data)
}
```

### Register Shared Templates

```go
// shared/templates.go
package shared

import (
    "embed"
    "github.com/dalemusser/waffle/templates"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

func init() {
    templates.Register(templates.Set{
        Name:     "shared",  // Must be "shared"
        FS:       templateFS,
        Patterns: []string{"templates/*.gohtml"},
    })
}
```

### Error Handling

```go
func showUser(w http.ResponseWriter, r *http.Request) {
    user, err := db.GetUser(r.Context(), chi.URLParam(r, "id"))
    if err != nil {
        // Render error page
        templates.Render(w, r, "error_page", map[string]any{
            "Title":   "Not Found",
            "Message": "User not found",
        })
        return
    }

    templates.Render(w, r, "user_show", map[string]any{
        "User": user,
    })
}
```

## See Also

- [app](../app/app.md) — Application lifecycle
- [router](../router/router.md) — HTTP routing

