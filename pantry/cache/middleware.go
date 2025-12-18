// cache/middleware.go
package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Middleware returns HTTP middleware that caches responses.
func Middleware(c Cache, cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.TTL == 0 {
		cfg.TTL = 5 * time.Minute
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET and HEAD requests
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			// Check skip function
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Generate cache key
			key := cfg.KeyFunc(r)
			if cfg.KeyPrefix != "" {
				key = cfg.KeyPrefix + key
			}

			// Try to get from cache
			data, err := c.Get(r.Context(), key)
			if err == nil {
				// Cache hit - parse and write cached response
				entry, err := decodeCacheEntry(data)
				if err == nil {
					writeCachedResponse(w, entry, cfg.TTL)
					return
				}
			}

			// Cache miss - capture response
			rec := &responseRecorder{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			// Only cache successful responses
			if !cfg.CacheErrors && (rec.statusCode < 200 || rec.statusCode >= 300) {
				return
			}

			// Check if response should be cached based on headers
			if !shouldCache(rec) {
				return
			}

			// Store in cache
			entry := &cacheEntry{
				StatusCode: rec.statusCode,
				Headers:    rec.Header().Clone(),
				Body:       rec.body.Bytes(),
			}

			if encoded, err := encodeCacheEntry(entry); err == nil {
				c.Set(r.Context(), key, encoded, cfg.TTL)
			}
		})
	}
}

// MiddlewareConfig configures the cache middleware.
type MiddlewareConfig struct {
	// TTL is how long to cache responses. Default: 5 minutes.
	TTL time.Duration

	// KeyFunc generates cache keys from requests.
	// Default: method + path + sorted query string.
	KeyFunc func(r *http.Request) string

	// KeyPrefix is prepended to all cache keys.
	KeyPrefix string

	// Skip returns true to skip caching for a request.
	Skip func(r *http.Request) bool

	// CacheErrors caches non-2xx responses. Default: false.
	CacheErrors bool
}

// DefaultKeyFunc generates a cache key from method, path, and query string.
func DefaultKeyFunc(r *http.Request) string {
	key := r.Method + ":" + r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	return key
}

// HashKeyFunc generates a hashed cache key (useful for long URLs).
func HashKeyFunc(r *http.Request) string {
	key := DefaultKeyFunc(r)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// PathKeyFunc generates a cache key from path only (ignores query string).
func PathKeyFunc(r *http.Request) string {
	return r.Method + ":" + r.URL.Path
}

// cacheEntry holds a cached HTTP response.
type cacheEntry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// encodeCacheEntry serializes a cache entry.
func encodeCacheEntry(entry *cacheEntry) ([]byte, error) {
	var buf bytes.Buffer

	// Write status code
	buf.WriteString(strconv.Itoa(entry.StatusCode))
	buf.WriteByte('\n')

	// Write headers
	for key, values := range entry.Headers {
		for _, value := range values {
			buf.WriteString(key)
			buf.WriteByte(':')
			buf.WriteString(value)
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')

	// Write body
	buf.Write(entry.Body)

	return buf.Bytes(), nil
}

// decodeCacheEntry deserializes a cache entry.
func decodeCacheEntry(data []byte) (*cacheEntry, error) {
	entry := &cacheEntry{
		Headers: make(http.Header),
	}

	// Find first newline (status code)
	idx := bytes.IndexByte(data, '\n')
	if idx < 0 {
		return nil, ErrNotFound
	}

	statusCode, err := strconv.Atoi(string(data[:idx]))
	if err != nil {
		return nil, err
	}
	entry.StatusCode = statusCode
	data = data[idx+1:]

	// Read headers until empty line
	for {
		idx = bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}

		line := data[:idx]
		data = data[idx+1:]

		if len(line) == 0 {
			break
		}

		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx < 0 {
			continue
		}

		key := string(line[:colonIdx])
		value := string(line[colonIdx+1:])
		entry.Headers.Add(key, value)
	}

	// Rest is body
	entry.Body = data

	return entry, nil
}

// writeCachedResponse writes a cached response to the client.
func writeCachedResponse(w http.ResponseWriter, entry *cacheEntry, ttl time.Duration) {
	// Copy headers
	for key, values := range entry.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Add cache headers
	w.Header().Set("X-Cache", "HIT")
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(ttl.Seconds())))

	w.WriteHeader(entry.StatusCode)
	w.Write(entry.Body)
}

// shouldCache checks response headers to determine if caching is appropriate.
func shouldCache(rec *responseRecorder) bool {
	// Check Cache-Control header
	cc := rec.Header().Get("Cache-Control")
	if cc != "" {
		cc = strings.ToLower(cc)
		if strings.Contains(cc, "no-store") || strings.Contains(cc, "private") {
			return false
		}
	}

	// Check Vary header - don't cache if Vary: *
	vary := rec.Header().Get("Vary")
	if vary == "*" {
		return false
	}

	return true
}

// responseRecorder captures the response for caching.
type responseRecorder struct {
	http.ResponseWriter
	body        *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
