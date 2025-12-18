// logging/recovermw.go
package logging

import (
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Recoverer returns a middleware that recovers from panics, logs them with a
// stack trace, and returns HTTP 500 if headers haven't been written yet.
//
// If the panic occurs after response headers have already been sent, the
// middleware logs the panic but cannot change the response status code.
// In this case, it logs an additional warning indicating the response may
// be incomplete.
func Recoverer(logger *zap.Logger) func(next http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap response writer to track if headers have been written.
			// Default to HTTP/1.x if ProtoMajor is invalid (e.g., malformed request).
			protoMajor := r.ProtoMajor
			if protoMajor < 1 {
				protoMajor = 1
			}
			ww := middleware.NewWrapResponseWriter(w, protoMajor)

			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						zap.Any("panic_value", rec),
						zap.ByteString("stacktrace", debug.Stack()),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote_ip", r.RemoteAddr),
					)

					// Only send error response if headers haven't been written yet.
					// If headers are already sent, the client will receive an incomplete
					// response, but we can't change the status code at this point.
					if ww.Status() == 0 {
						http.Error(w, "internal server error", http.StatusInternalServerError)
					} else {
						logger.Warn("panic occurred after headers written; response may be incomplete",
							zap.Int("status_already_sent", ww.Status()),
							zap.String("path", r.URL.Path))
					}
				}
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
