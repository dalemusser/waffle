// feature/errors.go
package feature

import "errors"

// Common feature flag errors.
var (
	// ErrFlagNotFound is returned when a flag does not exist.
	ErrFlagNotFound = errors.New("feature: flag not found")

	// ErrInvalidKey is returned when a flag key is empty or invalid.
	ErrInvalidKey = errors.New("feature: invalid flag key")

	// ErrInvalidPercentage is returned when percentage is out of range (0-100).
	ErrInvalidPercentage = errors.New("feature: percentage must be between 0 and 100")

	// ErrStoreNotConfigured is returned when attempting storage operations without a store.
	ErrStoreNotConfigured = errors.New("feature: store not configured")
)
