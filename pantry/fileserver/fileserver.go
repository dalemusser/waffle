// pantry/fileserver/fileserver.go

// Package fileserver provides an HTTP handler for serving static files from a
// filesystem directory with support for pre-compressed files (gzip and Brotli).
//
// This is particularly useful for serving Unity WebGL builds and other large
// assets that benefit from pre-compression, where on-the-fly compression is
// not viable due to file size.
//
// When a request comes in, the handler checks if the client supports Brotli
// or gzip (via Accept-Encoding header) and looks for pre-compressed versions
// of the file (.br or .gz suffix). If found, it serves the compressed file
// with the appropriate Content-Encoding header, allowing the browser to
// decompress natively during download.
package fileserver

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// Handler returns an http.Handler that serves static files from the given
// root directory with support for pre-compressed files.
//
// The urlPrefix is stripped from the request URL before looking up files.
// For example, if urlPrefix is "/static" and the request is for "/static/js/app.js",
// the handler looks for "js/app.js" in the root directory.
//
// Pre-compressed file lookup:
//   - If Accept-Encoding includes "br", checks for file.br
//   - If Accept-Encoding includes "gzip", checks for file.gz
//   - Falls back to the uncompressed file if no compressed version exists
//
// Example usage:
//
//	r.Handle("/static/*", fileserver.Handler("/static", "public"))
//	r.Handle("/games/*", fileserver.Handler("/games", "/var/www/games"))
func Handler(urlPrefix, rootDir string) http.Handler {
	root := http.Dir(rootDir)
	fs := http.FileServer(root)

	return http.StripPrefix(urlPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only optimize GET/HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fs.ServeHTTP(w, r)
			return
		}

		// Canonicalize the requested path and strip leading slash for http.Dir.Open
		req := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

		// Try pre-compressed variants: .br first (better compression), then .gz
		candidates := []struct {
			ext      string
			encoding string
			accepted bool
		}{
			{".br", "br", acceptsEncoding(r, "br")},
			{".gz", "gzip", acceptsEncoding(r, "gzip")},
		}

		for _, cand := range candidates {
			if !cand.accepted {
				continue
			}

			f, err := root.Open(req + cand.ext)
			if err != nil {
				continue
			}

			fi, err := f.Stat()
			if err != nil || fi.IsDir() {
				_ = f.Close()
				continue
			}

			// Serve the pre-compressed file with appropriate headers
			w.Header().Set("Content-Encoding", cand.encoding)
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Set("Content-Type", mimeTypeByOriginal(req))

			// Serve with proper modtime and range support
			http.ServeContent(w, r, req, fi.ModTime(), f)
			_ = f.Close()
			return
		}

		// Fallback: serve the uncompressed file
		fs.ServeHTTP(w, r)
	}))
}

// HandlerWithOptions returns an http.Handler with additional configuration options.
func HandlerWithOptions(urlPrefix, rootDir string, opts Options) http.Handler {
	root := http.Dir(rootDir)
	fs := http.FileServer(root)

	return http.StripPrefix(urlPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only optimize GET/HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fs.ServeHTTP(w, r)
			return
		}

		// Canonicalize the requested path
		req := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

		// Apply cache headers if configured
		if opts.CacheControl != "" {
			w.Header().Set("Cache-Control", opts.CacheControl)
		}

		// Try pre-compressed variants if enabled
		if !opts.DisablePrecompressed {
			candidates := []struct {
				ext      string
				encoding string
				accepted bool
			}{
				{".br", "br", acceptsEncoding(r, "br")},
				{".gz", "gzip", acceptsEncoding(r, "gzip")},
			}

			for _, cand := range candidates {
				if !cand.accepted {
					continue
				}

				f, err := root.Open(req + cand.ext)
				if err != nil {
					continue
				}

				fi, err := f.Stat()
				if err != nil || fi.IsDir() {
					_ = f.Close()
					continue
				}

				w.Header().Set("Content-Encoding", cand.encoding)
				w.Header().Set("Vary", "Accept-Encoding")
				w.Header().Set("Content-Type", mimeTypeByOriginal(req))

				http.ServeContent(w, r, req, fi.ModTime(), f)
				_ = f.Close()
				return
			}
		}

		// Fallback: serve the uncompressed file
		fs.ServeHTTP(w, r)
	}))
}

// Options configures the static file handler behavior.
type Options struct {
	// CacheControl sets the Cache-Control header for all responses.
	// Example: "public, max-age=31536000, immutable" for content-hashed files.
	CacheControl string

	// DisablePrecompressed disables checking for .br and .gz variants.
	DisablePrecompressed bool
}

// acceptsEncoding checks if the client accepts the given encoding.
func acceptsEncoding(r *http.Request, encoding string) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if strings.EqualFold(enc, encoding) {
			return true
		}
	}
	return false
}

// mimeTypeByOriginal returns the MIME type for the original filename
// (without .gz/.br suffix).
func mimeTypeByOriginal(name string) string {
	// Strip compression suffix to get original extension
	base := name
	for strings.HasSuffix(base, ".br") || strings.HasSuffix(base, ".gz") {
		base = strings.TrimSuffix(strings.TrimSuffix(base, ".br"), ".gz")
	}
	ext := strings.ToLower(filepath.Ext(base))

	// Use Go's MIME table first
	if mt := mime.TypeByExtension(ext); mt != "" {
		return mt
	}

	// Additional MIME types for Unity WebGL and common web assets
	switch ext {
	case ".wasm":
		return "application/wasm"
	case ".js":
		return "application/javascript"
	case ".mjs":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".data":
		return "application/octet-stream"
	case ".unityweb":
		return "application/octet-stream"
	case ".mem":
		return "application/octet-stream"
	case ".symbols":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}
