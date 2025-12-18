// middleware/cors.go
package middleware

import (
	"fmt"
	"net/http"

	"github.com/dalemusser/waffle/config"
	"github.com/go-chi/cors"
)

// CORSOptions defines CORS policy settings.
//
// Security Considerations:
//
// When AllowCredentials is true, the browser will include cookies and
// authentication headers in cross-origin requests. This has significant
// security implications:
//
//   - NEVER use AllowCredentials with wildcard origins (["*"]). The browser
//     will reject this combination, and WAFFLE's config validation enforces it.
//
//   - Only allow origins you explicitly trust. Each allowed origin can make
//     authenticated requests on behalf of your users.
//
//   - Consider using SameSite cookie attributes alongside CORS for defense
//     in depth against CSRF attacks.
//
//   - If you don't need credentials (cookies/auth), keep AllowCredentials false
//     to reduce your attack surface.
type CORSOptions struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// Use ["*"] to allow any origin, or specify exact origins like ["https://example.com"].
	// IMPORTANT: Using ["*"] with AllowCredentials=true is not allowed by browsers.
	AllowedOrigins []string

	// AllowedMethods is a list of methods the client is allowed to use.
	// Default: ["GET", "POST", "OPTIONS"]
	AllowedMethods []string

	// AllowedHeaders is a list of headers the client is allowed to use.
	// Default: ["Accept", "Authorization", "Content-Type"]
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that are safe to expose to the client.
	ExposedHeaders []string

	// AllowCredentials indicates whether the request can include credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	// Cannot be true when AllowedOrigins is ["*"].
	// See the security considerations above before enabling this.
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Default: 300 (5 minutes)
	MaxAge int
}

// CORSFromConfig returns a middleware that applies CORS behavior based on the
// given CoreConfig's CORS section.
//
// If coreCfg is nil or coreCfg.CORS.EnableCORS is false, it returns an identity
// middleware that does nothing. This makes it safe to unconditionally call:
//
//	r.Use(middleware.CORSFromConfig(coreCfg))
//
// and let config decide whether CORS is active.
//
// Note: Passing nil config is supported but discouraged. If you need CORS but
// don't have a CoreConfig, use the CORS() function with explicit options instead.
func CORSFromConfig(coreCfg *config.CoreConfig) func(next http.Handler) http.Handler {
	if coreCfg == nil {
		// No config provided - return no-op. This is intentional to allow
		// unconditional use in middleware chains, but callers should prefer
		// passing a valid config.
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	if !coreCfg.CORS.EnableCORS {
		// CORS explicitly disabled via config
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	opts := cors.Options{
		AllowedOrigins:   coreCfg.CORS.CORSAllowedOrigins,
		AllowedMethods:   coreCfg.CORS.CORSAllowedMethods,
		AllowedHeaders:   coreCfg.CORS.CORSAllowedHeaders,
		ExposedHeaders:   coreCfg.CORS.CORSExposedHeaders,
		AllowCredentials: coreCfg.CORS.CORSAllowCredentials,
		MaxAge:           coreCfg.CORS.CORSMaxAge,
	}

	return cors.Handler(opts)
}

// maxCORSMaxAge is the maximum allowed preflight cache duration (1 week).
// This matches the validation in config.CoreConfig to ensure consistency.
const maxCORSMaxAge = 7 * 24 * 60 * 60 // 1 week in seconds

// CORS returns a middleware with the specified CORS options.
// Use this when you need custom CORS settings not driven by config.
//
// Panics immediately (at middleware construction time, not at first request)
// if AllowCredentials is true and AllowedOrigins contains "*", as this
// combination is rejected by browsers and indicates a misconfiguration.
// Also panics if MaxAge exceeds 1 week (604800 seconds).
// This ensures misconfigurations are caught during application startup
// when building the router, not at runtime.
//
// Example:
//
//	r.Use(middleware.CORS(middleware.CORSOptions{
//	    AllowedOrigins:   []string{"https://app.example.com"},
//	    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
//	    AllowCredentials: true,
//	    MaxAge:           600,
//	}))
func CORS(opts CORSOptions) func(next http.Handler) http.Handler {
	// Validate: credentials + wildcard origin is not allowed by browsers
	if opts.AllowCredentials {
		for _, origin := range opts.AllowedOrigins {
			if origin == "*" {
				panic("middleware.CORS: AllowCredentials=true cannot be used with wildcard origin \"*\" - browsers will reject this")
			}
		}
	}

	// Validate: MaxAge should not exceed 1 week (browsers may ignore larger values)
	if opts.MaxAge > maxCORSMaxAge {
		panic(fmt.Sprintf("middleware.CORS: MaxAge %d exceeds maximum of %d seconds (1 week)", opts.MaxAge, maxCORSMaxAge))
	}

	// Apply defaults for unset fields
	if len(opts.AllowedMethods) == 0 {
		opts.AllowedMethods = []string{"GET", "POST", "OPTIONS"}
	}
	if len(opts.AllowedHeaders) == 0 {
		opts.AllowedHeaders = []string{"Accept", "Authorization", "Content-Type"}
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 300
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   opts.AllowedOrigins,
		AllowedMethods:   opts.AllowedMethods,
		AllowedHeaders:   opts.AllowedHeaders,
		ExposedHeaders:   opts.ExposedHeaders,
		AllowCredentials: opts.AllowCredentials,
		MaxAge:           opts.MaxAge,
	})
}

// CORSPermissive returns a permissive CORS middleware suitable for development
// or internal APIs where security restrictions are not needed.
//
// Policy:
//   - Allow all origins ("*")
//   - Allow GET, POST, PUT, PATCH, DELETE, OPTIONS
//   - Allow common headers including Authorization
//   - No credentials (cannot use credentials with "*" origins)
//   - 5 minute preflight cache
//
// WARNING: Do not use in production with sensitive APIs. Use CORS() or
// CORSFromConfig() with specific allowed origins instead.
func CORSPermissive() func(next http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}
