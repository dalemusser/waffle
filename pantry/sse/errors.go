// sse/errors.go
package sse

import "errors"

// Common SSE errors.
var (
	// ErrFlushNotSupported is returned when the response writer doesn't support flushing.
	ErrFlushNotSupported = errors.New("sse: response writer does not support flushing")

	// ErrStreamClosed is returned when attempting to send on a closed stream.
	ErrStreamClosed = errors.New("sse: stream closed")

	// ErrBrokerClosed is returned when attempting to use a closed broker.
	ErrBrokerClosed = errors.New("sse: broker closed")

	// ErrChannelNotFound is returned when a channel does not exist.
	ErrChannelNotFound = errors.New("sse: channel not found")

	// ErrClientNotFound is returned when a client is not registered.
	ErrClientNotFound = errors.New("sse: client not found")
)
