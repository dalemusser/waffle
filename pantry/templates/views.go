// templates/views.go
package templates

import (
	"io/fs"
	"sync"
)

// Set describes one module's template set.
type Set struct {
	// Name is for logging / debugging only (e.g., "shared", "admin_resources").
	Name string
	// FS is the embedded filesystem from the feature package.
	FS fs.FS
	// Patterns are the glob patterns to load from FS (e.g., []string{"templates/*.gohtml"}).
	Patterns []string
}

var (
	registryMu sync.RWMutex
	registry   []Set
)

// Register is typically called from a feature package’s init().
// It records a template Set so the Engine can load it at Boot().
func Register(s Set) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, s)
}

// All returns the registered template sets.
// The Engine calls this once at Boot.
func All() []Set {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Set, len(registry))
	copy(out, registry)
	return out
}

// Reset is handy for tests.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = nil
}
