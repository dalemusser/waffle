// auth/jwt/middleware.go
package jwt

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is the type for context keys.
type contextKey string

const claimsContextKey contextKey = "jwt_claims"

// Config configures the JWT middleware.
type Config struct {
	// Signer is used to verify tokens. Required.
	Signer *Signer

	// TokenLookup defines where to find the token.
	// Format: "header:Authorization", "query:token", "cookie:jwt"
	// Default: "header:Authorization"
	TokenLookup string

	// AuthScheme is the authorization scheme (e.g., "Bearer").
	// Default: "Bearer"
	AuthScheme string

	// ErrorHandler is called when authentication fails.
	// Default: returns 401 with error message.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// SuccessHandler is called after successful authentication.
	// Can be used to add claims to context differently.
	SuccessHandler func(w http.ResponseWriter, r *http.Request, claims any)

	// Skip returns true to skip authentication for a request.
	Skip func(r *http.Request) bool

	// ClaimsFactory creates a new claims instance for parsing.
	// Default: creates *Claims
	ClaimsFactory func() any
}

// Middleware returns JWT authentication middleware.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.TokenLookup == "" {
		cfg.TokenLookup = "header:Authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultErrorHandler
	}
	if cfg.ClaimsFactory == nil {
		cfg.ClaimsFactory = func() any { return &Claims{} }
	}

	extractor := createExtractor(cfg.TokenLookup, cfg.AuthScheme)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token
			token := extractor(r)
			if token == "" {
				cfg.ErrorHandler(w, r, ErrInvalidToken)
				return
			}

			// Create claims instance and verify
			claims := cfg.ClaimsFactory()
			if err := cfg.Signer.Verify(token, claims); err != nil {
				cfg.ErrorHandler(w, r, err)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			r = r.WithContext(ctx)

			// Call success handler if provided
			if cfg.SuccessHandler != nil {
				cfg.SuccessHandler(w, r, claims)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MiddlewareWithClaims returns middleware that parses tokens into a specific claims type.
func MiddlewareWithClaims[T any](signer *Signer) func(http.Handler) http.Handler {
	return Middleware(Config{
		Signer: signer,
		ClaimsFactory: func() any {
			return new(T)
		},
	})
}

// Optional returns middleware that extracts JWT if present but doesn't require it.
func Optional(cfg Config) func(http.Handler) http.Handler {
	if cfg.TokenLookup == "" {
		cfg.TokenLookup = "header:Authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}
	if cfg.ClaimsFactory == nil {
		cfg.ClaimsFactory = func() any { return &Claims{} }
	}

	extractor := createExtractor(cfg.TokenLookup, cfg.AuthScheme)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token
			token := extractor(r)
			if token != "" {
				claims := cfg.ClaimsFactory()
				if err := cfg.Signer.Verify(token, claims); err == nil {
					ctx := context.WithValue(r.Context(), claimsContextKey, claims)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// FromContext retrieves claims from the request context.
func FromContext(ctx context.Context) any {
	return ctx.Value(claimsContextKey)
}

// ClaimsFromContext retrieves standard claims from context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey).(*Claims)
	return claims
}

// UserClaimsFromContext retrieves user claims from context.
func UserClaimsFromContext(ctx context.Context) *UserClaims {
	claims, _ := ctx.Value(claimsContextKey).(*UserClaims)
	return claims
}

// GetClaims retrieves typed claims from context.
func GetClaims[T any](ctx context.Context) *T {
	claims, _ := ctx.Value(claimsContextKey).(*T)
	return claims
}

// RequireRole returns middleware that checks for a specific role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := UserClaimsFromContext(r.Context())
			if claims == nil || !claims.HasRole(role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole returns middleware that checks for any of the specified roles.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := UserClaimsFromContext(r.Context())
			if claims == nil || !claims.HasAnyRole(roles...) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllRoles returns middleware that checks for all of the specified roles.
func RequireAllRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := UserClaimsFromContext(r.Context())
			if claims == nil || !claims.HasAllRoles(roles...) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// createExtractor creates a token extractor based on the lookup string.
func createExtractor(lookup, scheme string) func(*http.Request) string {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return func(r *http.Request) string { return "" }
	}

	source := parts[0]
	name := parts[1]

	switch source {
	case "header":
		return func(r *http.Request) string {
			auth := r.Header.Get(name)
			if auth == "" {
				return ""
			}
			if scheme != "" {
				prefix := scheme + " "
				if strings.HasPrefix(auth, prefix) {
					return auth[len(prefix):]
				}
				return ""
			}
			return auth
		}
	case "query":
		return func(r *http.Request) string {
			return r.URL.Query().Get(name)
		}
	case "cookie":
		return func(r *http.Request) string {
			cookie, err := r.Cookie(name)
			if err != nil {
				return ""
			}
			return cookie.Value
		}
	default:
		return func(r *http.Request) string { return "" }
	}
}

// defaultErrorHandler writes a 401 response.
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	http.Error(w, err.Error(), http.StatusUnauthorized)
}
