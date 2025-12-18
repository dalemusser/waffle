// audit/errors.go
package audit

import "errors"

// Common audit errors.
var (
	// ErrLoggerNotFound is returned when no logger is found in context.
	ErrLoggerNotFound = errors.New("audit: logger not found")

	// ErrStoreNotConfigured is returned when store operations are attempted without a store.
	ErrStoreNotConfigured = errors.New("audit: store not configured")

	// ErrQueryNotSupported is returned when a store doesn't support querying.
	ErrQueryNotSupported = errors.New("audit: query not supported by this store")

	// ErrChannelFull is returned when the event channel is full.
	ErrChannelFull = errors.New("audit: event channel full")

	// ErrEventRequired is returned when an event is nil.
	ErrEventRequired = errors.New("audit: event is required")

	// ErrActionRequired is returned when an action is empty.
	ErrActionRequired = errors.New("audit: action is required")
)
