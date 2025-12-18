# fileserver

HTTP handler for serving static files from a filesystem directory with automatic support for pre-compressed files (Brotli and gzip).

## When to Use

Use `fileserver` when serving files from disk, especially:

- **Unity WebGL builds** — Large, pre-compressed game files (20MB+)
- **User-uploaded content** — Files that change at runtime
- **Large media files** — Videos, audio, downloadable assets
- **Development mode** — Hot reload without rebuilding

For small, rarely-changing assets, prefer `go:embed` with `http.FileServer` instead.

## Import

```go
import "github.com/dalemusser/waffle/pantry/fileserver"
```

## Quick Start

```go
// Serve files from "public" directory at /static/
r.Handle("/static/*", fileserver.Handler("/static", "public"))

// Serve Unity games from /var/www/games at /games/
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))
```

## API

### Handler

```go
func Handler(urlPrefix, rootDir string) http.Handler
```

Returns an HTTP handler that serves files from `rootDir`, stripping `urlPrefix` from request URLs. Automatically serves pre-compressed `.br` or `.gz` variants when the client supports them.

**Parameters:**
- `urlPrefix` — URL path prefix to strip (e.g., `"/static"`)
- `rootDir` — Filesystem directory containing files

**Example:**

```go
// Request: GET /static/js/app.js
// Looks for: public/js/app.js (or .br/.gz variants)
r.Handle("/static/*", fileserver.Handler("/static", "public"))
```

### HandlerWithOptions

```go
func HandlerWithOptions(urlPrefix, rootDir string, opts Options) http.Handler
```

Returns a handler with additional configuration.

**Example:**

```go
r.Handle("/assets/*", fileserver.HandlerWithOptions("/assets", "public", fileserver.Options{
    CacheControl:         "public, max-age=31536000, immutable",
    DisablePrecompressed: false,
}))
```

### Options

```go
type Options struct {
    // CacheControl sets the Cache-Control header for all responses.
    // Example: "public, max-age=31536000, immutable"
    CacheControl string

    // DisablePrecompressed disables checking for .br and .gz variants.
    DisablePrecompressed bool
}
```

## Pre-Compression Support

The handler automatically detects and serves pre-compressed files:

| Client Accepts | Handler Checks | Response Header |
|----------------|----------------|-----------------|
| `br` | `file.br` | `Content-Encoding: br` |
| `gzip` | `file.gz` | `Content-Encoding: gzip` |

Brotli is checked first (better compression), then gzip. Falls back to uncompressed if neither variant exists.

### Creating Pre-Compressed Files

**Brotli:**
```bash
brotli -q 11 -k file.wasm  # Creates file.wasm.br, keeps original
```

**Gzip:**
```bash
gzip -9 -k file.wasm  # Creates file.wasm.gz, keeps original
```

**Batch compression:**
```bash
find public -name "*.js" -exec brotli -q 11 -k {} \;
find public -name "*.js" -exec gzip -9 -k {} \;
```

## Unity WebGL

Unity WebGL builds produce large files that benefit from pre-compression. Configure Unity to output Brotli or gzip compressed files:

**Unity Build Settings:**
1. Player Settings → Publishing Settings
2. Compression Format: `Brotli` (HTTPS only) or `Gzip` (HTTP/HTTPS)

**Directory structure:**
```
/var/www/games/
└── my-game/
    ├── index.html
    └── Build/
        ├── my-game.loader.js
        ├── my-game.data.br
        ├── my-game.framework.js.br
        └── my-game.wasm.br
```

**Serve:**
```go
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))
```

Browser requests `/games/my-game/Build/my-game.wasm`, handler finds `my-game.wasm.br`, serves with `Content-Encoding: br` and `Content-Type: application/wasm`.

## Caching Strategies

**Immutable content-hashed files:**
```go
r.Handle("/assets/*", fileserver.HandlerWithOptions("/assets", "dist", fileserver.Options{
    CacheControl: "public, max-age=31536000, immutable",
}))
```

**Frequently updated:**
```go
r.Handle("/uploads/*", fileserver.HandlerWithOptions("/uploads", "uploads", fileserver.Options{
    CacheControl: "public, max-age=3600",
}))
```

**Development (no cache):**
```go
r.Handle("/static/*", fileserver.HandlerWithOptions("/static", "public", fileserver.Options{
    CacheControl: "no-cache, no-store, must-revalidate",
}))
```

## MIME Types

The handler sets `Content-Type` based on the original file extension (before `.br`/`.gz`). Go's standard MIME table is used first, with fallbacks for:

| Extension | MIME Type |
|-----------|-----------|
| `.wasm` | `application/wasm` |
| `.js`, `.mjs` | `application/javascript` |
| `.json` | `application/json` |
| `.data`, `.unityweb`, `.mem`, `.symbols` | `application/octet-stream` |

## Security

- **Directory traversal**: Prevented via `path.Clean()`. Requests like `/../../../etc/passwd` are safely handled.
- **Brotli over HTTP**: Browsers only accept Brotli over HTTPS. Use gzip for HTTP development, or enable HTTPS.

## Hybrid Approach

Combine embedded and filesystem serving:

```go
// Embedded: Small assets baked into binary
//go:embed static/*
var staticFS embed.FS
r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

// Filesystem: Large pre-compressed games
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))

// Filesystem: User uploads
r.Handle("/uploads/*", fileserver.Handler("/uploads", "uploads"))
```

## See Also

- [Static Files Guide](../../docs/pantry/static/static-files-filesystem.md) — Detailed usage guide
- [Unity WebGL Deployment](https://docs.unity3d.com/Manual/webgl-deploying.html) — Unity documentation
