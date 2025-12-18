// errors/http.go
package errors

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// Response is the JSON structure returned for errors.
type Response struct {
	Error *Error `json:"error"`
}

// Write writes an error as JSON to the response.
// It sets the appropriate HTTP status code and Content-Type header.
func Write(w http.ResponseWriter, err error) {
	e := From(err)
	writeError(w, e)
}

// WriteWithLogger writes an error and logs internal errors.
func WriteWithLogger(w http.ResponseWriter, err error, logger *zap.Logger) {
	e := From(err)

	// Log internal errors
	if e.Status >= 500 {
		logger.Error("internal error",
			zap.String("code", e.Code),
			zap.String("message", e.Message),
			zap.Error(e.Err),
		)
	}

	writeError(w, e)
}

func writeError(w http.ResponseWriter, e *Error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(e.HTTPStatus())

	resp := Response{Error: e}
	json.NewEncoder(w).Encode(resp)
}

// Handler wraps an http.Handler and recovers from panics, writing an error response.
func Handler(h http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.String("path", r.URL.Path),
				)
				Write(w, Internal("an unexpected error occurred"))
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// HandlerFunc is like Handler but for http.HandlerFunc.
func HandlerFunc(h http.HandlerFunc, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.String("path", r.URL.Path),
				)
				Write(w, Internal("an unexpected error occurred"))
			}
		}()
		h(w, r)
	}
}

// NotFoundHandler returns an http.Handler that responds with 404 Not Found.
func NotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Write(w, NotFound("the requested resource was not found"))
	})
}

// MethodNotAllowedHandler returns an http.Handler that responds with 405 Method Not Allowed.
func MethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Write(w, MethodNotAllowed("the requested method is not allowed"))
	})
}

// ErrorHandlerFunc is a handler function that returns an error.
// If the error is non-nil, it is written as JSON.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error

// WrapHandler converts an ErrorHandlerFunc to a standard http.HandlerFunc.
func WrapHandler(h ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			Write(w, err)
		}
	}
}

// WrapWithLogger converts an ErrorHandlerFunc to a standard http.HandlerFunc,
// logging internal errors.
func WrapWithLogger(h ErrorHandlerFunc, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteWithLogger(w, err, logger)
		}
	}
}

// Middleware returns middleware that handles errors returned from handlers.
// Use with ErrorHandlerFunc-style handlers via context.
func Middleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Handler(next, logger)
	}
}

// StatusFromError returns the HTTP status code for an error.
// Returns 500 if the error is not an *Error.
func StatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return From(err).HTTPStatus()
}

// CodeFromError returns the error code for an error.
// Returns "internal_error" if the error is not an *Error.
func CodeFromError(err error) string {
	if err == nil {
		return ""
	}
	return From(err).Code
}
