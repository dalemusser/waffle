// toolkit/cors/cors.go
package cors

import (
	"net/http"

	"github.com/go-chi/cors"
)

// Default returns a CORS middleware suitable for most WAFFLE-based services.
// It applies the following policy:
//   - Allow all origins ("*")
//   - Allow GET, POST, OPTIONS
//   - Allow common headers including Authorization
//   - No credentials
//
// If/when you want to tighten this (e.g., specific origins), we change it
// here once and all apps pick it up.
func Default() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300, // seconds to cache preflight
	})
}
