# Serving Documentation

*Guides for serving static files, APIs, and other content from WAFFLE applications.*

---

## Overview

This section covers patterns for serving content from WAFFLE applications, including static files (both embedded and from the filesystem), APIs, and other server-side content delivery patterns.

---

## Documentation

| Document | Description |
|----------|-------------|
| [**Static Files (Embedded)**](./static-files.md) | Embedding and serving assets using `go:embed` for single-binary deployment |
| [**Static Files (Filesystem)**](./static-files-filesystem.md) | Serving from disk with pre-compressed file support (Unity WebGL, large assets) |

---

## Quick Reference

### Embedded Static Files

For most web applications, embed static assets directly in your binary:

```go
//go:embed static/*
var staticFS embed.FS

// In BuildHandler:
r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
```

### Filesystem Static Files

For large files or files that change at runtime:

```go
import "github.com/dalemusser/waffle/pantry/fileserver"

// Serve with pre-compression support
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))
```

---

## See Also

- [Frontend Documentation](../frontend/README.md) — Templates, HTMX, Tailwind
- [Routes Examples](../routes-examples.md) — General routing patterns

---

[← Back to Guides](../README.md)
