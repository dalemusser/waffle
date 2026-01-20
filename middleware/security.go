// middleware/security.go
package middleware

import (
	"net/http"
	"strconv"

	"github.com/dalemusser/waffle/config"
)

// SecurityHeadersOptions configures the security headers middleware.
//
// All headers have sensible defaults that provide good security without
// breaking most applications. Only customize if you have specific needs.
type SecurityHeadersOptions struct {
	// XFrameOptions controls whether the page can be embedded in iframes.
	// Values: "DENY", "SAMEORIGIN", or "ALLOW-FROM uri"
	// Default: "SAMEORIGIN"
	// Set to empty string to disable this header.
	XFrameOptions string

	// XContentTypeOptions prevents MIME type sniffing.
	// Should always be "nosniff" unless you have a specific reason to disable.
	// Default: "nosniff"
	// Set to empty string to disable this header.
	XContentTypeOptions string

	// ReferrerPolicy controls how much referrer information is sent.
	// Common values:
	//   - "no-referrer" - Never send referrer
	//   - "origin" - Send only the origin (scheme + host)
	//   - "strict-origin-when-cross-origin" - Full URL to same origin, origin only to others
	// Default: "strict-origin-when-cross-origin"
	// Set to empty string to disable this header.
	ReferrerPolicy string

	// XSSProtection enables/configures the browser's XSS filter.
	// Note: This is deprecated in modern browsers in favor of CSP.
	// Values: "0" (disable), "1" (enable), "1; mode=block"
	// Default: "1; mode=block"
	// Set to empty string to disable this header.
	XSSProtection string

	// HSTSMaxAge sets the Strict-Transport-Security max-age in seconds.
	// Only sent when the request is over HTTPS.
	// Default: 31536000 (1 year)
	// Set to 0 to disable HSTS.
	HSTSMaxAge int

	// HSTSIncludeSubDomains adds includeSubDomains to the HSTS header.
	// Default: true
	HSTSIncludeSubDomains bool

	// HSTSPreload adds the preload directive to HSTS.
	// Only enable if you've submitted your domain to the HSTS preload list.
	// Default: false
	HSTSPreload bool

	// ContentSecurityPolicy sets the Content-Security-Policy header.
	// This is powerful but complex - misconfiguration can break your site.
	// Default: empty (not set)
	// Example: "default-src 'self'; script-src 'self' 'unsafe-inline'"
	ContentSecurityPolicy string

	// PermissionsPolicy (formerly Feature-Policy) controls browser features.
	// Default: empty (not set)
	// Example: "geolocation=(), microphone=(), camera=()"
	PermissionsPolicy string
}

// DefaultSecurityHeadersOptions returns options with secure defaults suitable
// for most web applications.
func DefaultSecurityHeadersOptions() SecurityHeadersOptions {
	return SecurityHeadersOptions{
		XFrameOptions:         "SAMEORIGIN",
		XContentTypeOptions:   "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		XSSProtection:         "1; mode=block",
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubDomains: true,
		HSTSPreload:           false,
		// CSP and Permissions-Policy left empty by default as they require
		// application-specific configuration
	}
}

// SecurityHeaders returns middleware that sets common security headers.
// Uses the provided options to configure which headers are set.
//
// Example:
//
//	r.Use(middleware.SecurityHeaders(middleware.DefaultSecurityHeadersOptions()))
//
// Or with custom options:
//
//	r.Use(middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
//	    XFrameOptions:       "DENY",
//	    XContentTypeOptions: "nosniff",
//	    HSTSMaxAge:          63072000, // 2 years
//	}))
func SecurityHeaders(opts SecurityHeadersOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Frame-Options: Prevents clickjacking by controlling iframe embedding
			if opts.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", opts.XFrameOptions)
			}

			// X-Content-Type-Options: Prevents MIME type sniffing
			if opts.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", opts.XContentTypeOptions)
			}

			// Referrer-Policy: Controls referrer information leakage
			if opts.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", opts.ReferrerPolicy)
			}

			// X-XSS-Protection: Legacy XSS filter (deprecated but still useful for old browsers)
			if opts.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", opts.XSSProtection)
			}

			// Strict-Transport-Security: Forces HTTPS
			// Only set for HTTPS requests to avoid issues with HTTP development
			if opts.HSTSMaxAge > 0 && r.TLS != nil {
				hsts := "max-age=" + strconv.Itoa(opts.HSTSMaxAge)
				if opts.HSTSIncludeSubDomains {
					hsts += "; includeSubDomains"
				}
				if opts.HSTSPreload {
					hsts += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hsts)
			}

			// Content-Security-Policy: Powerful XSS prevention (application-specific)
			if opts.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", opts.ContentSecurityPolicy)
			}

			// Permissions-Policy: Controls browser features
			if opts.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", opts.PermissionsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersFromConfig returns middleware configured from CoreConfig.
//
// If security headers are disabled in config (enable_security_headers = false),
// returns a no-op middleware. This makes it safe to unconditionally use:
//
//	r.Use(middleware.SecurityHeadersFromConfig(coreCfg))
//
// The middleware respects the production/development environment:
//   - HSTS is only sent for HTTPS requests
//   - In dev mode with use_https=false, HSTS is effectively disabled
func SecurityHeadersFromConfig(coreCfg *config.CoreConfig) func(next http.Handler) http.Handler {
	if coreCfg == nil {
		// No config - return no-op
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	if !coreCfg.Security.EnableSecurityHeaders {
		// Security headers explicitly disabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	opts := SecurityHeadersOptions{
		XFrameOptions:         coreCfg.Security.XFrameOptions,
		XContentTypeOptions:   coreCfg.Security.XContentTypeOptions,
		ReferrerPolicy:        coreCfg.Security.ReferrerPolicy,
		XSSProtection:         coreCfg.Security.XSSProtection,
		HSTSMaxAge:            coreCfg.Security.HSTSMaxAge,
		HSTSIncludeSubDomains: coreCfg.Security.HSTSIncludeSubDomains,
		HSTSPreload:           coreCfg.Security.HSTSPreload,
		ContentSecurityPolicy: coreCfg.Security.ContentSecurityPolicy,
		PermissionsPolicy:     coreCfg.Security.PermissionsPolicy,
	}

	return SecurityHeaders(opts)
}

// SecureDefaults returns middleware with secure default headers.
// This is a convenience function equivalent to:
//
//	SecurityHeaders(DefaultSecurityHeadersOptions())
//
// Use this when you want good security defaults without configuration.
func SecureDefaults() func(next http.Handler) http.Handler {
	return SecurityHeaders(DefaultSecurityHeadersOptions())
}
