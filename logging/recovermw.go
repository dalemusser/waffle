// logging/recovermw.go
package logging

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recoverer returns a middleware that recovers from panics, logs them with a
// stack trace, and returns HTTP 500.
func Recoverer(logger *zap.Logger) func(next http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						zap.Any("panic_value", rec),
						zap.ByteString("stacktrace", debug.Stack()),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote_ip", r.RemoteAddr),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
