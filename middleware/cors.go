// middleware/cors.go
package middleware

import (
	"net/http"

	"github.com/dalemusser/waffle/config"
	"github.com/go-chi/cors"
)

// CORSFromConfig returns a middleware that applies CORS behavior based on the
// given CoreConfig's CORS section.
//
// If coreCfg.CORS.EnableCORS is false, it returns an identity middleware that
// does nothing. This makes it safe to unconditionally call:
//
//	r.Use(middleware.CORSFromConfig(coreCfg))
//
// and let config decide whether CORS is active.
func CORSFromConfig(coreCfg *config.CoreConfig) func(next http.Handler) http.Handler {
	if coreCfg == nil || !coreCfg.CORS.EnableCORS {
		// No-op middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	opts := cors.Options{
		AllowedOrigins:   coreCfg.CORS.CORSAllowedOrigins,
		AllowedMethods:   coreCfg.CORS.CORSAllowedMethods,
		AllowedHeaders:   coreCfg.CORS.CORSAllowedHeaders,
		ExposedHeaders:   coreCfg.CORS.CORSExposedHeaders,
		AllowCredentials: coreCfg.CORS.CORSAllowCredentials,
		MaxAge:           coreCfg.CORS.CORSMaxAge,
	}

	return cors.Handler(opts)
}
