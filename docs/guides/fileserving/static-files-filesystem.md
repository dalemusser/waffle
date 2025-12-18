# Filesystem Static File Serving

*Serving static files from disk with pre-compressed file support for Unity WebGL and large assets.*

---

## Overview

While WAFFLE's embedded static file serving (`go:embed`) is ideal for most web assets, some use cases require serving files from the filesystem:

- **Unity WebGL games** — Large, pre-compressed builds (often 20MB+)
- **User-uploaded content** — Files that change at runtime
- **Large media files** — Videos, audio, downloadable assets
- **Development mode** — Hot reload without rebuild

The `pantry/fileserver` package provides an HTTP handler that serves files from a directory with automatic support for pre-compressed files (Brotli and gzip).

### Pre-Compression Support

When serving large files like Unity WebGL builds, on-the-fly compression is impractical due to file size. Instead, files are pre-compressed at build time, and the server detects and serves the compressed version with appropriate headers.

| Feature | Description |
|---------|-------------|
| **Brotli support** | Looks for `.br` files, serves with `Content-Encoding: br` |
| **Gzip support** | Looks for `.gz` files, serves with `Content-Encoding: gzip` |
| **Automatic detection** | Checks `Accept-Encoding` header |
| **Correct MIME types** | Sets `Content-Type` based on original file extension |
| **Fallback** | Serves uncompressed file if no compressed version exists |

---

## Quick Start

### 1. Import the Package

```go
import "github.com/yourusername/yourapp/waffle/pantry/fileserver"
```

### 2. Mount the Handler

```go
// Mount at /static/ serving files from "public" directory
r.Handle("/static/*", fileserver.Handler("/static", "public"))

// Mount games at /games/ serving from "/var/www/games"
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))
```

### 3. Place Files in Directory

```
public/
├── css/
│   └── style.css
├── js/
│   └── app.js
└── images/
    └── logo.png
```

Files are now served at `/static/css/style.css`, `/static/js/app.js`, etc.

---

## API Reference

### Handler

```go
func Handler(urlPrefix, rootDir string) http.Handler
```

Returns an HTTP handler that serves static files from `rootDir`, stripping `urlPrefix` from request URLs.

**Parameters:**
- `urlPrefix` — URL path prefix to strip (e.g., `"/static"`)
- `rootDir` — Filesystem directory containing files

**Example:**

```go
// Request: GET /static/js/app.js
// Serves: ./public/js/app.js
r.Handle("/static/*", fileserver.Handler("/static", "public"))
```

### HandlerWithOptions

```go
func HandlerWithOptions(urlPrefix, rootDir string, opts Options) http.Handler
```

Returns a handler with additional configuration options.

**Options:**

```go
type Options struct {
    // CacheControl sets the Cache-Control header for all responses.
    CacheControl string

    // DisablePrecompressed disables checking for .br and .gz variants.
    DisablePrecompressed bool
}
```

**Example:**

```go
r.Handle("/assets/*", fileserver.HandlerWithOptions("/assets", "public", fileserver.Options{
    CacheControl: "public, max-age=31536000, immutable",
}))
```

---

## Unity WebGL Serving

Unity WebGL builds produce large files that benefit from pre-compression. Unity can output Brotli (`.br`) or gzip (`.gz`) compressed files at build time.

### Unity Build Settings

In Unity's Build Settings for WebGL:

1. **Player Settings** → **Publishing Settings**
2. **Compression Format**: Choose `Gzip` or `Brotli`
   - **Brotli**: Smaller files, but only works over HTTPS
   - **Gzip**: Larger files, works over HTTP and HTTPS

### Unity Build Output Structure

```
Build/
├── game.loader.js
├── game.data.br           # Compressed game data
├── game.framework.js.br   # Compressed framework
├── game.wasm.br           # Compressed WebAssembly
└── index.html
```

### Serving Unity Builds

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/yourusername/yourapp/waffle/pantry/fileserver"
)

func main() {
    r := chi.NewRouter()

    // Serve Unity WebGL games from /games/{game-name}/
    r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))

    http.ListenAndServe(":8080", r)
}
```

### Directory Structure for Multiple Games

```
/var/www/games/
├── math-adventure/
│   ├── index.html
│   ├── Build/
│   │   ├── math-adventure.loader.js
│   │   ├── math-adventure.data.br
│   │   ├── math-adventure.framework.js.br
│   │   └── math-adventure.wasm.br
│   └── TemplateData/
│       ├── style.css
│       └── favicon.ico
│
└── word-puzzle/
    ├── index.html
    ├── Build/
    │   ├── word-puzzle.loader.js
    │   ├── word-puzzle.data.gz
    │   ├── word-puzzle.framework.js.gz
    │   └── word-puzzle.wasm.gz
    └── TemplateData/
        └── ...
```

### How It Works

1. Browser requests `/games/math-adventure/Build/math-adventure.wasm`
2. Handler checks `Accept-Encoding` header for `br` support
3. Finds `math-adventure.wasm.br` on disk
4. Serves compressed file with headers:
   - `Content-Encoding: br`
   - `Content-Type: application/wasm`
   - `Vary: Accept-Encoding`
5. Browser decompresses natively during download

### Unity WebGL Template Integration

In your Unity WebGL template's `index.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{{ PRODUCT_NAME }}}</title>
    <link rel="stylesheet" href="TemplateData/style.css">
</head>
<body>
    <div id="unity-container">
        <canvas id="unity-canvas"></canvas>
    </div>
    <script src="Build/{{{ LOADER_FILENAME }}}"></script>
    <script>
        createUnityInstance(document.querySelector("#unity-canvas"), {
            dataUrl: "Build/{{{ DATA_FILENAME }}}",
            frameworkUrl: "Build/{{{ FRAMEWORK_FILENAME }}}",
            codeUrl: "Build/{{{ CODE_FILENAME }}}",
        });
    </script>
</body>
</html>
```

---

## Feature Integration

### Serving Static Files in a Feature

```go
// internal/app/features/home/routes.go
package home

import (
    "github.com/go-chi/chi/v5"
    "github.com/yourusername/yourapp/waffle/pantry/fileserver"
)

func MountRoutes(r chi.Router) {
    // Serve static files from "static" directory at project root
    r.Handle("/static/*", fileserver.Handler("/static", "static"))

    // Other routes
    r.Get("/", homeHandler)
}
```

### With Configuration

```go
// internal/app/features/games/routes.go
package games

import (
    "github.com/go-chi/chi/v5"
    "github.com/yourusername/yourapp/internal/app/config"
    "github.com/yourusername/yourapp/waffle/pantry/fileserver"
)

func MountRoutes(r chi.Router, cfg *config.AppConfig) {
    // Serve games from configured directory
    r.Handle("/games/*", fileserver.HandlerWithOptions("/games", cfg.GamesDir, fileserver.Options{
        CacheControl: "public, max-age=86400", // 1 day cache
    }))

    r.Get("/games", listGamesHandler)
}
```

---

## Caching Strategies

### Content-Hashed Files (Immutable)

For files with content hashes in their names:

```go
r.Handle("/assets/*", fileserver.HandlerWithOptions("/assets", "public/assets", fileserver.Options{
    CacheControl: "public, max-age=31536000, immutable",
}))
```

### Frequently Updated Files

For files that change often:

```go
r.Handle("/uploads/*", fileserver.HandlerWithOptions("/uploads", "uploads", fileserver.Options{
    CacheControl: "public, max-age=3600", // 1 hour
}))
```

### No Cache (Development)

```go
r.Handle("/static/*", fileserver.HandlerWithOptions("/static", "public", fileserver.Options{
    CacheControl: "no-cache, no-store, must-revalidate",
}))
```

---

## Pre-Compressing Files

### Using gzip

```bash
# Compress a single file
gzip -k file.wasm  # Creates file.wasm.gz, keeps original

# Compress all WASM files
find . -name "*.wasm" -exec gzip -k {} \;

# Compress with best compression
gzip -9 -k file.wasm
```

### Using Brotli

```bash
# Install brotli CLI (Ubuntu/Debian)
sudo apt install brotli

# Compress a single file
brotli -k file.wasm  # Creates file.wasm.br, keeps original

# Compress with best compression (slower)
brotli -q 11 -k file.wasm

# Compress all JS files
find . -name "*.js" -exec brotli -k {} \;
```

### Build Script Example

```bash
#!/bin/bash
# compress-assets.sh

ASSETS_DIR="./public"

# Compress JavaScript
find "$ASSETS_DIR" -name "*.js" ! -name "*.min.js" | while read file; do
    gzip -9 -k -f "$file"
    brotli -q 11 -k -f "$file"
done

# Compress CSS
find "$ASSETS_DIR" -name "*.css" | while read file; do
    gzip -9 -k -f "$file"
    brotli -q 11 -k -f "$file"
done

# Compress WASM
find "$ASSETS_DIR" -name "*.wasm" | while read file; do
    gzip -9 -k -f "$file"
    brotli -q 11 -k -f "$file"
done

echo "Compression complete"
```

### Makefile Integration

```makefile
.PHONY: compress-assets

compress-assets:
	@echo "Compressing static assets..."
	@find public -name "*.js" -exec gzip -9 -k -f {} \;
	@find public -name "*.js" -exec brotli -q 11 -k -f {} \;
	@find public -name "*.css" -exec gzip -9 -k -f {} \;
	@find public -name "*.css" -exec brotli -q 11 -k -f {} \;
	@find public -name "*.wasm" -exec gzip -9 -k -f {} \;
	@find public -name "*.wasm" -exec brotli -q 11 -k -f {} \;
	@echo "Done"

build: compress-assets
	go build -o bin/myapp ./cmd/myapp
```

---

## Embedded vs Filesystem

| Aspect | Embedded (`go:embed`) | Filesystem |
|--------|----------------------|------------|
| **Deployment** | Single binary | Binary + files |
| **Runtime changes** | Requires rebuild | Files can change |
| **Binary size** | Larger (includes assets) | Smaller |
| **Large files** | Not recommended | Better suited |
| **Pre-compression** | Manual handling | Automatic |
| **Use case** | CSS, JS, small images | Games, uploads, media |

### Hybrid Approach

Use both methods in the same application:

```go
// Embedded: Small, rarely-changing assets
staticFS := resources.StaticFS()
r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

// Filesystem: Large, pre-compressed game files
r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))

// Filesystem: User uploads
r.Handle("/uploads/*", fileserver.Handler("/uploads", "uploads"))
```

---

## Troubleshooting

### Compressed file not being served

**Cause**: Browser doesn't support the encoding or file doesn't exist.

**Debug steps:**
1. Check `Accept-Encoding` header in browser dev tools
2. Verify `.br` or `.gz` file exists on disk
3. Check file permissions

```bash
# Verify files exist
ls -la /path/to/file.wasm*
# Should show: file.wasm, file.wasm.br, file.wasm.gz
```

### Wrong Content-Type

**Cause**: Extension not recognized.

**Solution**: The handler recognizes common extensions. For unusual extensions, the file will be served as `application/octet-stream`.

### Unity WebGL not loading

**Cause**: Missing `Content-Encoding` header or wrong `Content-Type`.

**Debug steps:**
1. Open browser Network tab
2. Check response headers for the `.wasm` request
3. Should see `Content-Encoding: br` (or `gzip`)
4. Should see `Content-Type: application/wasm`

### Brotli not working over HTTP

**Cause**: Browsers only accept Brotli over HTTPS.

**Solution**: Use HTTPS in production, or use gzip for HTTP development.

### Large files causing timeouts

**Cause**: Very large files taking too long to serve.

**Solutions:**
1. Use a CDN for very large assets
2. Increase server timeouts
3. Consider chunked delivery for huge files

---

## Security Considerations

### Directory Traversal

The handler uses `path.Clean()` to prevent directory traversal attacks. Requests like `/static/../../../etc/passwd` are safely handled.

### Serving Sensitive Files

Never place sensitive files in served directories:

```
public/           # ✅ Safe to serve
├── css/
├── js/
└── images/

config/           # ❌ Never serve this
├── secrets.toml
└── credentials.json
```

### HTTPS for Brotli

Browsers only accept Brotli compression over HTTPS. For production, always use HTTPS.

---

## See Also

- [Static File Serving (Embedded)](./static-files.md) — For `go:embed` approach
- [Unity WebGL Deployment](https://docs.unity3d.com/Manual/webgl-deploying.html) — Unity documentation
- [Templates and Views](../frontend/templates-and-views.md)

---

[← Back to Serving Documentation](./README.md)
