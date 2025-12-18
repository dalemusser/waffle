// version/version.go
package version

import (
	"net/http"
	"runtime"

	"github.com/dalemusser/waffle/httputil"
	"github.com/go-chi/chi/v5"
)

// These variables are meant to be set at build time using ldflags:
//
//	go build -ldflags "-X github.com/dalemusser/waffle/version.Version=1.0.0 \
//	                   -X github.com/dalemusser/waffle/version.Commit=abc123 \
//	                   -X github.com/dalemusser/waffle/version.BuildTime=2024-01-15T10:30:00Z"
var (
	// Version is the semantic version of the application (e.g., "1.2.3").
	Version = "dev"

	// Commit is the git commit SHA at build time.
	Commit = "unknown"

	// BuildTime is the timestamp when the binary was built (RFC3339 format).
	BuildTime = "unknown"
)

// Info contains version and build information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current version info.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// Handler returns an http.Handler that responds with version info as JSON.
//
// Response example:
//
//	{
//	  "version": "1.2.3",
//	  "commit": "abc123def",
//	  "build_time": "2024-01-15T10:30:00Z",
//	  "go_version": "go1.22.0",
//	  "os": "linux",
//	  "arch": "amd64"
//	}
func Handler() http.Handler {
	// Pre-compute info since it never changes after startup.
	info := Get()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, info)
	})
}

// Mount attaches a /version route to the given chi.Router.
//
// Example:
//
//	version.Mount(r)
func Mount(r chi.Router) {
	r.Method(http.MethodGet, "/version", Handler())
}

// MountAt is like Mount but allows specifying a custom path.
//
// Example:
//
//	version.MountAt(r, "/api/version")
func MountAt(r chi.Router, path string) {
	r.Method(http.MethodGet, path, Handler())
}

// String returns a human-readable version string.
//
// Example output: "1.2.3 (abc123, built 2024-01-15T10:30:00Z)"
func String() string {
	if Version == "dev" {
		return "dev"
	}
	return Version + " (" + Commit + ", built " + BuildTime + ")"
}
