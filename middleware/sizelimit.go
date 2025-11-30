// middleware/sizelimit.go
package middleware

import (
	"net/http"
)

// LimitBodySize returns a middleware that limits the size of the request body
// to maxBytes. If maxBytes <= 0, it is a no-op and does not wrap the body.
//
// This should typically be applied early in the middleware chain to prevent
// handlers from processing huge bodies.
func LimitBodySize(maxBytes int64) func(next http.Handler) http.Handler {
	if maxBytes <= 0 {
		// No limit: return identity middleware.
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
