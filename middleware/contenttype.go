// middleware/contenttype.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/dalemusser/waffle/httputil"
)

// RequireJSON returns a middleware that ensures requests have a JSON Content-Type,
// typically "application/json" or something ending in "+json", e.g.
// "application/problem+json".
//
// If the Content-Type is missing or not JSON, it returns 415 Unsupported Media Type
// with a JSON error body.
func RequireJSON() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := strings.TrimSpace(r.Header.Get("Content-Type"))
			if ct == "" {
				httputil.JSONError(w, http.StatusUnsupportedMediaType,
					"unsupported_media_type",
					"Content-Type must be application/json",
				)
				return
			}

			// Strip any parameters, e.g. "; charset=utf-8"
			if idx := strings.Index(ct, ";"); idx != -1 {
				ct = ct[:idx]
			}
			ct = strings.ToLower(strings.TrimSpace(ct))

			if ct != "application/json" && !strings.HasSuffix(ct, "+json") {
				httputil.JSONError(w, http.StatusUnsupportedMediaType,
					"unsupported_media_type",
					"Content-Type must be application/json",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
