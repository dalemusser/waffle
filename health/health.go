// health/health.go
package health

import (
	"context"
	"net/http"

	"github.com/dalemusser/waffle/httputil"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Check represents a single health probe. It should return nil if the
// dependency is healthy, or a non-nil error describing the problem.
// The ctx passed in is derived from the incoming request context.
type Check func(ctx context.Context) error

// Response is the JSON structure returned by the health handler.
type Response struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Handler returns an http.Handler that runs the provided checks on each
// request and returns a JSON response.
// If checks is nil or empty, it behaves as a simple liveness probe:
//
//	{ "status": "ok" }
//
// If any check returns an error, the handler responds with 503 and:
//
//	{ "status": "error", "checks": { "db": "error: ...", ... } }
//
// Otherwise it responds with 200 and:
//
//	{ "status": "ok", "checks": { "db": "ok", ... } }.
func Handler(checks map[string]Check, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No checks: simple liveness.
		if len(checks) == 0 {
			resp := Response{Status: "ok"}
			httputil.WriteJSON(w, http.StatusOK, resp)
			return
		}

		ctx := r.Context()
		results := make(map[string]string, len(checks))
		anyErr := false

		for name, check := range checks {
			if check == nil {
				// Treat nil check as "ok" by default.
				results[name] = "ok"
				continue
			}
			if err := check(ctx); err != nil {
				anyErr = true
				msg := "error"
				if err.Error() != "" {
					msg = "error: " + err.Error()
				}
				results[name] = msg

				if logger != nil {
					logger.Warn("health check failed",
						zap.String("check", name),
						zap.Error(err),
					)
				}
			} else {
				results[name] = "ok"
			}
		}

		if anyErr {
			resp := Response{
				Status: "error",
				Checks: results,
			}
			httputil.WriteJSON(w, http.StatusServiceUnavailable, resp)
			return
		}

		resp := Response{
			Status: "ok",
			Checks: results,
		}
		httputil.WriteJSON(w, http.StatusOK, resp)
	})
}

// Mount attaches a /health route to the given chi.Router using the provided
// checks and logger.
//
// Example:
//
//	checks := map[string]health.Check{
//	    "db": func(ctx context.Context) error {
//	        return client.Ping(ctx, readpref.Primary())
//	    },
//	}
//	health.Mount(r, checks, logger)
func Mount(r chi.Router, checks map[string]Check, logger *zap.Logger) {
	r.Method(http.MethodGet, "/health", Handler(checks, logger))
}

// MountAt is like Mount but allows specifying a custom path, e.g. "/ready"
// or "/live".
func MountAt(r chi.Router, path string, checks map[string]Check, logger *zap.Logger) {
	r.Method(http.MethodGet, path, Handler(checks, logger))
}
