// middleware/compress.go
package middleware

import (
	"fmt"
	"net/http"

	"github.com/dalemusser/waffle/config"
	"github.com/go-chi/chi/v5/middleware"
)

// Logger is a minimal interface for logging warnings during middleware setup.
// This avoids coupling to zap while allowing optional warning logs.
type Logger interface {
	Warn(msg string, args ...any)
}

// CompressFromConfig returns a compression middleware based on the CoreConfig.
//
// If coreCfg.EnableCompression is false, it returns an identity middleware that
// does nothing. This makes it safe to unconditionally call:
//
//	r.Use(middleware.CompressFromConfig(coreCfg, nil))
//
// and let config decide whether compression is active. The compression level
// is determined by coreCfg.CompressionLevel (1-9, default 5).
//
// Compression supports gzip and deflate encodings based on the client's
// Accept-Encoding header.
//
// Note: The compression level is validated in config.validateCoreConfig().
// Invalid values reaching this function indicate a bug in config validation.
func CompressFromConfig(coreCfg *config.CoreConfig, logger Logger) func(next http.Handler) http.Handler {
	if coreCfg == nil || !coreCfg.EnableCompression {
		// No-op middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	level := coreCfg.CompressionLevel
	if level < 1 || level > 9 {
		// This should never happen if config validation is working correctly.
		// Panic to surface the bug rather than silently using a default.
		panic(fmt.Sprintf("middleware: invalid compression level %d (should have been caught by config validation)", level))
	}
	return middleware.Compress(level)
}

// Compress returns a compression middleware with the specified compression level.
//
// Level ranges from 1 (best speed) to 9 (best compression). Level 5 is a good
// balance between speed and compression ratio. Levels outside 1-9 are clamped
// to the nearest valid value.
//
// Compression supports gzip and deflate encodings based on the client's
// Accept-Encoding header.
//
// Example:
//
//	r.Use(middleware.Compress(5))
func Compress(level int) func(next http.Handler) http.Handler {
	return CompressWithLogger(level, nil)
}

// CompressWithLogger is like Compress but logs a warning if the level is clamped.
func CompressWithLogger(level int, logger Logger) func(next http.Handler) http.Handler {
	if level < 1 {
		if logger != nil {
			logger.Warn(fmt.Sprintf("compression level %d clamped to 1 (minimum)", level))
		}
		level = 1
	}
	if level > 9 {
		if logger != nil {
			logger.Warn(fmt.Sprintf("compression level %d clamped to 9 (maximum)", level))
		}
		level = 9
	}
	return middleware.Compress(level)
}

// CompressWithTypes returns a compression middleware that only compresses
// responses with the specified content types. Levels outside 1-9 are clamped
// to the nearest valid value.
//
// Example:
//
//	r.Use(middleware.CompressWithTypes(5, "text/html", "text/css", "application/json"))
func CompressWithTypes(level int, types ...string) func(next http.Handler) http.Handler {
	return CompressWithTypesAndLogger(level, nil, types...)
}

// CompressWithTypesAndLogger is like CompressWithTypes but logs a warning if the level is clamped.
func CompressWithTypesAndLogger(level int, logger Logger, types ...string) func(next http.Handler) http.Handler {
	if level < 1 {
		if logger != nil {
			logger.Warn(fmt.Sprintf("compression level %d clamped to 1 (minimum)", level))
		}
		level = 1
	}
	if level > 9 {
		if logger != nil {
			logger.Warn(fmt.Sprintf("compression level %d clamped to 9 (maximum)", level))
		}
		level = 9
	}
	return middleware.Compress(level, types...)
}
