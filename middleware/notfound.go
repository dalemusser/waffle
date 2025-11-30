package middleware

import (
	"net/http"

	"github.com/dalemusser/waffle/httputil"
	"go.uber.org/zap"
)

// NotFoundHandler returns a handler that logs a 404 and returns a JSON error body.
// It is designed to be passed directly to chi.Router.NotFound(..).
func NotFoundHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logger != nil {
			logger.Info("not_found",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_ip", r.RemoteAddr),
			)
		}

		httputil.JSONError(w, http.StatusNotFound,
			"not_found",
			"The requested resource was not found",
		)
	}
}

// MethodNotAllowedHandler returns a handler that logs a 405 and returns a JSON error body.
// It is designed to be passed directly to chi.Router.MethodNotAllowed(..).
func MethodNotAllowedHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logger != nil {
			logger.Info("method_not_allowed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_ip", r.RemoteAddr),
			)
		}

		httputil.JSONError(w, http.StatusMethodNotAllowed,
			"method_not_allowed",
			"The requested HTTP method is not allowed for this resource",
		)
	}
}
