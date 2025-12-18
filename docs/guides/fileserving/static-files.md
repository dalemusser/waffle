# Static File Serving in WAFFLE

*Embedding and serving static assets using Go's `embed` package for single-binary deployment.*

---

## Overview

WAFFLE uses Go's `embed` package to include static assets directly in the compiled binary. This approach provides:

- **Single-binary deployment** — No external files to manage
- **Immutable assets** — Files can't be accidentally modified in production
- **Simplified deployment** — Copy one file, run it
- **Version consistency** — Assets always match the code version

---

## Basic Embedding

### The `go:embed` Directive

```go
package resources

import "embed"

// Embed a single file
//go:embed logo.png
var logoFile []byte

// Embed multiple specific files
//go:embed logo.png favicon.ico
var iconFiles embed.FS

// Embed all files matching a pattern
//go:embed static/*
var staticFS embed.FS

// Embed all files in a directory tree
//go:embed static/**/*
var allStatic embed.FS

// Embed templates
//go:embed templates/*.gohtml
var templateFS embed.FS
```

### Key Rules

| Rule | Example | Notes |
|------|---------|-------|
| Must be package-level var | `var FS embed.FS` | Not in functions |
| Path relative to source file | `//go:embed templates/*` | From file's directory |
| No `..` paths | `//go:embed ../shared` | ❌ Not allowed |
| Patterns use `/` separator | `//go:embed static/css/*.css` | Even on Windows |

---

## Directory Structure

### Recommended Layout

```
internal/app/
├── resources/
│   ├── resources.go          # Embed directives
│   ├── templates/
│   │   ├── layout.gohtml
│   │   └── menu.gohtml
│   └── static/
│       ├── css/
│       │   └── style.css
│       ├── js/
│       │   └── app.js
│       └── images/
│           ├── logo.png
│           └── favicon.ico
│
└── features/
    └── dashboard/
        ├── templates.go       # Feature templates
        ├── templates/
        │   └── dashboard.gohtml
        └── assets/            # Feature-specific assets (optional)
            └── dashboard.css
```

---

## Embedding Static Assets

### resources/resources.go

```go
package resources

import (
    "embed"
    "io/fs"
    "sync"

    "github.com/yourusername/yourapp/waffle/templates"
)

// Embed templates
//go:embed templates/*.gohtml
var templateFS embed.FS

// Embed static assets
//go:embed static/*
var staticFS embed.FS

var registerOnce sync.Once

// LoadSharedTemplates registers shared templates with the template engine.
func LoadSharedTemplates() {
    registerOnce.Do(func() {
        templates.Register(templates.Set{
            Name:     "shared",
            FS:       templateFS,
            Patterns: []string{"templates/*.gohtml"},
        })
    })
}

// StaticFS returns the embedded static filesystem.
// Use this to serve static files via HTTP.
func StaticFS() fs.FS {
    // Strip "static/" prefix so files are served from root
    subFS, _ := fs.Sub(staticFS, "static")
    return subFS
}
```

---

## Serving Static Files

### Using Chi Router

```go
// bootstrap/routes.go
package bootstrap

import (
    "io/fs"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/yourusername/yourapp/internal/app/resources"
)

func SetupRoutes(r chi.Router) {
    // Serve static files from /static/
    staticFS := resources.StaticFS()
    fileServer := http.FileServer(http.FS(staticFS))
    r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

    // Or mount at root for specific paths
    r.Handle("/css/*", http.StripPrefix("/css/",
        http.FileServer(http.FS(mustSub(staticFS, "css")))))
    r.Handle("/js/*", http.StripPrefix("/js/",
        http.FileServer(http.FS(mustSub(staticFS, "js")))))
    r.Handle("/images/*", http.StripPrefix("/images/",
        http.FileServer(http.FS(mustSub(staticFS, "images")))))

    // Favicon at root
    r.Get("/favicon.ico", faviconHandler(staticFS))

    // ... other routes
}

func mustSub(fsys fs.FS, dir string) fs.FS {
    sub, err := fs.Sub(fsys, dir)
    if err != nil {
        panic(err)
    }
    return sub
}

func faviconHandler(fsys fs.FS) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        data, err := fs.ReadFile(fsys, "images/favicon.ico")
        if err != nil {
            http.NotFound(w, r)
            return
        }
        w.Header().Set("Content-Type", "image/x-icon")
        w.Header().Set("Cache-Control", "public, max-age=86400")
        w.Write(data)
    }
}
```

### Referencing in Templates

```html
<!-- layout.gohtml -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }}</title>

    <!-- CSS from embedded static files -->
    <link rel="stylesheet" href="/static/css/style.css">

    <!-- Favicon -->
    <link rel="icon" href="/favicon.ico" type="image/x-icon">
</head>
<body>
    {{ template "content" . }}

    <!-- JavaScript from embedded static files -->
    <script src="/static/js/app.js"></script>
</body>
</html>
```

---

## Caching Strategies

### Cache Headers

```go
func staticHandler(fsys fs.FS) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set cache headers for static assets
        w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year

        http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
    })
}
```

### Cache Busting with Version

```go
package resources

import "fmt"

// Version can be set at build time
var Version = "dev"

// VersionedPath returns a path with version query param
func VersionedPath(path string) string {
    return fmt.Sprintf("%s?v=%s", path, Version)
}
```

**Build with version:**

```bash
go build -ldflags "-X github.com/yourusername/yourapp/internal/app/resources.Version=$(git rev-parse --short HEAD)"
```

**Use in templates:**

```html
<link rel="stylesheet" href="{{ versionedPath "/static/css/style.css" }}">
```

**Add template function:**

```go
// templates/funcs.go
func Funcs() template.FuncMap {
    return template.FuncMap{
        // ... other funcs
        "versionedPath": resources.VersionedPath,
    }
}
```

---

## Feature-Specific Assets

Features can have their own embedded assets:

### features/dashboard/assets.go

```go
package dashboard

import (
    "embed"
    "io/fs"
)

//go:embed assets/*
var assetsFS embed.FS

// AssetsFS returns the feature's embedded assets.
func AssetsFS() fs.FS {
    sub, _ := fs.Sub(assetsFS, "assets")
    return sub
}
```

### Mount in Routes

```go
// bootstrap/routes.go
r.Handle("/dashboard/assets/*", http.StripPrefix("/dashboard/assets/",
    http.FileServer(http.FS(dashboard.AssetsFS()))))
```

---

## Common Asset Types

### CSS Files

```
static/css/
├── style.css          # Main styles
├── components.css     # Component styles
└── utilities.css      # Utility classes
```

### JavaScript Files

```
static/js/
├── app.js            # Main application JS
├── htmx.min.js       # HTMX library (if not using CDN)
└── alpine.min.js     # Alpine.js (if used)
```

### Images

```
static/images/
├── logo.png          # Application logo
├── logo.svg          # Vector logo
├── favicon.ico       # Favicon
├── favicon-16x16.png # Modern favicon sizes
├── favicon-32x32.png
└── icons/            # UI icons
    ├── edit.svg
    ├── delete.svg
    └── ...
```

### Fonts

```
static/fonts/
├── inter-regular.woff2
├── inter-bold.woff2
└── inter.css
```

---

## Development vs Production

### Development Mode (Optional Hot Reload)

For development, you might want to serve files from disk instead of embedded:

```go
package resources

import (
    "embed"
    "io/fs"
    "os"
)

//go:embed static/*
var staticFS embed.FS

// StaticFS returns either embedded or disk filesystem based on mode.
func StaticFS(devMode bool) fs.FS {
    if devMode {
        // Serve from disk for hot reload during development
        return os.DirFS("internal/app/resources/static")
    }
    // Production: use embedded files
    sub, _ := fs.Sub(staticFS, "static")
    return sub
}
```

### Production Considerations

1. **Embedded is immutable** — Changes require rebuild
2. **No file modification** — Can't update assets at runtime
3. **Binary size** — Large assets increase binary size
4. **Memory usage** — Assets loaded into memory on access

---

## Serving Single-Page Apps (SPAs)

If you have an SPA frontend (React, Vue, etc.), embed the built files:

```go
package spa

import (
    "embed"
    "io/fs"
    "net/http"
    "strings"
)

//go:embed dist/*
var spaFS embed.FS

// Handler serves the SPA with fallback to index.html
func Handler() http.Handler {
    subFS, _ := fs.Sub(spaFS, "dist")

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path

        // Try to serve the file
        f, err := subFS.Open(strings.TrimPrefix(path, "/"))
        if err == nil {
            f.Close()
            http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
            return
        }

        // Fallback to index.html for SPA routing
        index, _ := fs.ReadFile(subFS, "index.html")
        w.Header().Set("Content-Type", "text/html")
        w.Write(index)
    })
}
```

---

## Security Considerations

### Content-Type Headers

```go
func staticHandler(fsys fs.FS) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent content-type sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")

        http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
    })
}
```

### CSP Headers

```go
// Set Content Security Policy for static assets
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; "+
    "script-src 'self' https://unpkg.com; "+
    "style-src 'self' 'unsafe-inline'")
```

---

## Troubleshooting

### "pattern matches no files"

**Cause**: The embed pattern doesn't match any files.

**Solutions**:
1. Check path is relative to Go file
2. Verify files exist at specified paths
3. Check pattern syntax (`*` vs `**`)

### Binary size too large

**Cause**: Large assets embedded in binary.

**Solutions**:
1. Optimize images (compress, resize)
2. Use CDN for large libraries
3. Consider external asset hosting for very large files

### Files not found at runtime

**Cause**: Wrong path or prefix stripping.

**Solutions**:
1. Use `fs.Sub()` to strip directory prefixes
2. Verify URL path matches file structure
3. Check `http.StripPrefix()` matches route

### Stale assets in development

**Cause**: Using embedded files in dev mode.

**Solution**: Use disk-based serving for development:

```go
if os.Getenv("DEV") == "true" {
    // Serve from disk
}
```

---

## Best Practices

1. **Organize by type** — Group CSS, JS, images in subdirectories
2. **Minimize assets** — Compress CSS/JS, optimize images
3. **Use CDN for large libraries** — HTMX, Bootstrap, etc.
4. **Cache appropriately** — Long cache for versioned assets
5. **Separate concerns** — Feature-specific assets stay with features
6. **Version your assets** — Use cache busting in production

---

## See Also

- [Static File Serving (Filesystem)](./static-files-filesystem.md) — For serving from disk with pre-compressed file support (Unity WebGL, large assets)
- [Templates and Views](../frontend/templates-and-views.md)
- [HTMX Integration](../frontend/htmx-integration.md)
- [Feature Structure Examples](../feature-structure-examples.md)
- [Go embed Package](https://pkg.go.dev/embed)

---

[← Back to Serving Documentation](./README.md)
