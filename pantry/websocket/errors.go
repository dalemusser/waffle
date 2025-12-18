// websocket/errors.go
package websocket

import "errors"

// Common WebSocket errors.
var (
	// ErrConnectionClosed is returned when attempting to write to a closed connection.
	ErrConnectionClosed = errors.New("websocket: connection closed")

	// ErrExpectedTextMessage is returned when a text message was expected but not received.
	ErrExpectedTextMessage = errors.New("websocket: expected text message")

	// ErrExpectedBinaryMessage is returned when a binary message was expected but not received.
	ErrExpectedBinaryMessage = errors.New("websocket: expected binary message")

	// ErrHubClosed is returned when attempting to use a closed hub.
	ErrHubClosed = errors.New("websocket: hub closed")

	// ErrRoomNotFound is returned when a room does not exist.
	ErrRoomNotFound = errors.New("websocket: room not found")

	// ErrClientNotFound is returned when a client is not registered.
	ErrClientNotFound = errors.New("websocket: client not found")

	// ErrInvalidMessageType is returned for unsupported message types.
	ErrInvalidMessageType = errors.New("websocket: invalid message type")
)

// CloseError represents a WebSocket close error.
type CloseError struct {
	Code   StatusCode
	Reason string
}

// Error implements the error interface.
func (e *CloseError) Error() string {
	if e.Reason != "" {
		return "websocket: closed: " + e.Reason
	}
	return "websocket: closed"
}

// IsCloseError returns true if the error is a WebSocket close error.
func IsCloseError(err error) bool {
	var closeErr *CloseError
	return errors.As(err, &closeErr)
}

// IsNormalClose returns true if the error is a normal close.
func IsNormalClose(err error) bool {
	var closeErr *CloseError
	if errors.As(err, &closeErr) {
		return closeErr.Code == StatusNormalClosure || closeErr.Code == StatusGoingAway
	}
	return false
}
