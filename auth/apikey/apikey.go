// auth/apikey/apikey.go
package apikey

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Options control how the API-key middleware behaves.
type Options struct {
	// Realm is used in the WWW-Authenticate header, e.g. "waffle-admin".
	Realm string

	// CookieName, if non-empty, enables cookie-based auth for browser flows.
	// On successful authentication, the middleware will set a cookie with this name
	// and the expected key as its value. On subsequent requests, if no key is
	// present in header/query, the cookie value will be used as the key.
	CookieName string
}

// Require returns a middleware that enforces a static API key.
// The key is considered valid if it matches the provided expected string.
// Key lookup order:
//  1. Authorization: Bearer <token>
//  2. X-API-Key header
//  3. api_key query param
//  4. Cookie (if Options.CookieName is set)
func Require(expected string, opts Options, logger *zap.Logger) func(next http.Handler) http.Handler {
	expected = strings.TrimSpace(expected)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				// Config validation should prevent this in prod, but don't panic at runtime.
				if logger != nil {
					logger.Warn("apikey.Require used with empty expected key")
				}
				http.Error(w, "server misconfigured", http.StatusInternalServerError)
				return
			}

			key, ok := apiKeyFromRequest(r)
			if !ok && opts.CookieName != "" {
				// Fallback: cookie-based auth for browser flows (e.g., /debug/pprof/*).
				if c, err := r.Cookie(opts.CookieName); err == nil {
					if val := strings.TrimSpace(c.Value); val != "" {
						key = val
						ok = true
					}
				}
			}

			if !ok || key != expected {
				if logger != nil {
					logger.Warn("API key unauthorized",
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("remote_ip", r.RemoteAddr),
					)
				}
				realm := opts.Realm
				if strings.TrimSpace(realm) == "" {
					realm = "waffle"
				}
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Successful auth: refresh a cookie for browser flows if configured.
			if opts.CookieName != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     opts.CookieName,
					Value:    expected,
					Path:     "/",
					Secure:   true,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyFromRequest extracts an API key from the request. It checks, in order:
//  1. Authorization: Bearer <token>
//  2. X-API-Key header
//  3. api_key query parameter
func apiKeyFromRequest(r *http.Request) (string, bool) {
	// 1) Authorization: Bearer <token>
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[len("Bearer "):])
		if token != "" {
			return token, true
		}
	}

	// 2) X-API-Key header
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key, true
	}

	// 3) api_key query parameter
	if key := strings.TrimSpace(r.URL.Query().Get("api_key")); key != "" {
		return key, true
	}

	return "", false
}
