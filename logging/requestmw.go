// logging/requestmw.go
package logging

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RequestLogger returns a middleware that logs HTTP requests with method, path,
// status, bytes, latency, remote IP, user agent, referer, and request ID.
func RequestLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			latency := time.Since(start)

			logger.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("host", r.Host),
				zap.String("scheme", schemeFromRequest(r)),
				zap.String("proto", r.Proto),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.String("remote_ip", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("referer", r.Referer()),
				zap.Duration("latency", latency),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}

func schemeFromRequest(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		return xf
	}
	return "http"
}
