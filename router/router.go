// router/router.go
package router

import (
	"github.com/dalemusser/waffle/config"
	"github.com/dalemusser/waffle/logging"
	"github.com/dalemusser/waffle/metrics"
	"github.com/dalemusser/waffle/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// New creates a chi.Router pre-wired with Waffle's standard middleware stack:
// - RequestID
// - RealIP
// - Recoverer (panic → 500)
// - body size limit (MaxRequestBodyBytes)
// - metrics HTTP middleware
// - request logging
// - NotFound / MethodNotAllowed JSON handlers
// It does NOT mount health, version, pprof, etc.; those remain app-level decisions.
func New(coreCfg *config.CoreConfig, logger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	// Request context & safety
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(logging.Recoverer(logger))

	// Body size limit (if configured)
	r.Use(middleware.LimitBodySize(coreCfg.MaxRequestBodyBytes))

	// Metrics
	r.Use(metrics.HTTPMetrics)

	// Access logging
	r.Use(logging.RequestLogger(logger))

	// NotFound / MethodNotAllowed JSON handlers
	r.NotFound(middleware.NotFoundHandler(logger))
	r.MethodNotAllowed(middleware.MethodNotAllowedHandler(logger))

	// CORS is better applied at app-level depending on routes, but we could
	// add an option later to wire it here using coreCfg.CORS.

	return r
}
