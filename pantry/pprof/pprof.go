// pprof/pprof.go
package pprof

import (
	stdpprof "net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// Mount attaches the standard Go pprof handlers under /debug/pprof.
// It must be mounted inside the router where API-key or other auth middleware
// has ALREADY been applied if protection is desired.
//
// Example:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(apikey.RequireAdminKey(...))
//	    pprof.Mount(r)
//	})
func Mount(r chi.Router) {
	r.Route("/debug/pprof", func(r chi.Router) {
		// Index page
		r.Get("/", stdpprof.Index)

		// Standard handlers
		r.Get("/cmdline", stdpprof.Cmdline)
		r.Get("/profile", stdpprof.Profile)
		r.Get("/symbol", stdpprof.Symbol)
		r.Post("/symbol", stdpprof.Symbol)
		r.Get("/trace", stdpprof.Trace)

		// Named profiles such as /debug/pprof/heap, goroutine, allocs, block, etc.
		r.Get("/{name}", stdpprof.Index)
	})
}
