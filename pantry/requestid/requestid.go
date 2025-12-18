// requestid/requestid.go
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync/atomic"
	"time"
)

// contextKey is the type for context keys.
type contextKey string

const requestIDKey contextKey = "request_id"

// DefaultHeader is the default HTTP header for request IDs.
const DefaultHeader = "X-Request-ID"

// Config configures the request ID middleware.
type Config struct {
	// Header is the HTTP header to use for the request ID.
	// Default: "X-Request-ID"
	Header string

	// Generator creates new request IDs.
	// Default: generates UUIDs.
	Generator func() string

	// TrustProxy trusts the incoming request ID header if present.
	// When false, always generates a new ID.
	// Default: true
	TrustProxy bool

	// Validator validates incoming request IDs.
	// If it returns false, a new ID is generated.
	// Default: accepts any non-empty string.
	Validator func(string) bool

	// SetResponseHeader adds the request ID to the response headers.
	// Default: true
	SetResponseHeader bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Header:            DefaultHeader,
		Generator:         GenerateUUID,
		TrustProxy:        true,
		Validator:         func(s string) bool { return len(s) > 0 && len(s) <= 128 },
		SetResponseHeader: true,
	}
}

// Middleware returns request ID middleware with the given configuration.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.Header == "" {
		cfg.Header = DefaultHeader
	}
	if cfg.Generator == nil {
		cfg.Generator = GenerateUUID
	}
	if cfg.Validator == nil {
		cfg.Validator = func(s string) bool { return len(s) > 0 && len(s) <= 128 }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var requestID string

			// Try to get from incoming header if trusted
			if cfg.TrustProxy {
				requestID = r.Header.Get(cfg.Header)
				if requestID != "" && !cfg.Validator(requestID) {
					requestID = "" // Invalid, generate new
				}
			}

			// Generate if not present or not trusted
			if requestID == "" {
				requestID = cfg.Generator()
			}

			// Add to response header
			if cfg.SetResponseHeader {
				w.Header().Set(cfg.Header, requestID)
			}

			// Add to context
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Simple returns middleware with default configuration.
func Simple() func(http.Handler) http.Handler {
	return Middleware(DefaultConfig())
}

// Get retrieves the request ID from the context.
// Returns an empty string if no request ID is present.
func Get(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// FromRequest retrieves the request ID from an HTTP request.
func FromRequest(r *http.Request) string {
	return Get(r.Context())
}

// Set adds a request ID to a context.
// Useful for propagating request IDs to background jobs or goroutines.
func Set(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ID generators

// GenerateUUID generates a UUID v4.
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID
		return GenerateTimestamp()
	}

	// Set version (4) and variant (RFC 4122)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return hex.EncodeToString(b[:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:])
}

// GenerateShort generates a short random ID (16 hex chars).
func GenerateShort() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return GenerateTimestamp()
	}
	return hex.EncodeToString(b)
}

// GenerateTimestamp generates a timestamp-based ID with random suffix.
func GenerateTimestamp() string {
	now := time.Now().UnixNano()
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString([]byte{
		byte(now >> 56),
		byte(now >> 48),
		byte(now >> 40),
		byte(now >> 32),
		byte(now >> 24),
		byte(now >> 16),
		byte(now >> 8),
		byte(now),
	}) + hex.EncodeToString(b)
}

// counter for sequential IDs
var counter uint64

// GenerateSequential generates a sequential ID with prefix.
// Format: prefix-timestamp-counter (e.g., "req-1234567890-00001")
func GenerateSequential(prefix string) func() string {
	return func() string {
		c := atomic.AddUint64(&counter, 1)
		ts := time.Now().Unix()
		return prefix + "-" + hex.EncodeToString([]byte{
			byte(ts >> 24),
			byte(ts >> 16),
			byte(ts >> 8),
			byte(ts),
		}) + "-" + hex.EncodeToString([]byte{
			byte(c >> 24),
			byte(c >> 16),
			byte(c >> 8),
			byte(c),
		})
	}
}

// GeneratePrefixed returns a generator that adds a prefix to UUIDs.
func GeneratePrefixed(prefix string) func() string {
	return func() string {
		return prefix + GenerateUUID()
	}
}

// Validators

// ValidateUUID checks if a string looks like a UUID.
func ValidateUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// ValidateHex checks if a string is valid hexadecimal.
func ValidateHex(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ValidateAlphanumeric checks if a string contains only alphanumeric characters and hyphens.
func ValidateAlphanumeric(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
