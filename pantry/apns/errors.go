// apns/errors.go
package apns

import (
	"errors"
	"fmt"
)

// Configuration errors.
var (
	// ErrAuthRequired is returned when no authentication is provided.
	ErrAuthRequired = errors.New("apns: auth key or certificate required")

	// ErrKeyIDRequired is returned when key ID is not provided.
	ErrKeyIDRequired = errors.New("apns: key ID required for token auth")

	// ErrTeamIDRequired is returned when team ID is not provided.
	ErrTeamIDRequired = errors.New("apns: team ID required for token auth")

	// ErrInvalidAuthKey is returned when the auth key is invalid.
	ErrInvalidAuthKey = errors.New("apns: invalid auth key (must be ECDSA P-256)")

	// ErrNotificationRequired is returned when notification is nil.
	ErrNotificationRequired = errors.New("apns: notification required")

	// ErrDeviceTokenRequired is returned when device token is empty.
	ErrDeviceTokenRequired = errors.New("apns: device token required")

	// ErrPayloadRequired is returned when payload is nil.
	ErrPayloadRequired = errors.New("apns: payload required")

	// ErrNoNotifications is returned when no notifications provided.
	ErrNoNotifications = errors.New("apns: no notifications provided")

	// ErrNoDefaultClient is returned when default client is not set.
	ErrNoDefaultClient = errors.New("apns: default client not set")
)

// APNs response errors (400 Bad Request).
var (
	// ErrBadCollapseID is returned when collapse ID is invalid.
	ErrBadCollapseID = errors.New("apns: bad collapse ID")

	// ErrBadDeviceToken is returned when device token is invalid.
	ErrBadDeviceToken = errors.New("apns: bad device token")

	// ErrBadExpirationDate is returned when expiration date is invalid.
	ErrBadExpirationDate = errors.New("apns: bad expiration date")

	// ErrBadMessageID is returned when message ID is invalid.
	ErrBadMessageID = errors.New("apns: bad message ID")

	// ErrBadPriority is returned when priority is invalid.
	ErrBadPriority = errors.New("apns: bad priority")

	// ErrBadTopic is returned when topic is invalid.
	ErrBadTopic = errors.New("apns: bad topic")

	// ErrDeviceTokenNotForTopic is returned when token doesn't match topic.
	ErrDeviceTokenNotForTopic = errors.New("apns: device token not for topic")

	// ErrDuplicateHeaders is returned when headers are duplicated.
	ErrDuplicateHeaders = errors.New("apns: duplicate headers")

	// ErrIdleTimeout is returned when idle timeout occurred.
	ErrIdleTimeout = errors.New("apns: idle timeout")

	// ErrInvalidPushType is returned when push type is invalid.
	ErrInvalidPushType = errors.New("apns: invalid push type")

	// ErrMissingDeviceToken is returned when device token is missing.
	ErrMissingDeviceToken = errors.New("apns: missing device token")

	// ErrMissingTopic is returned when topic is missing.
	ErrMissingTopic = errors.New("apns: missing topic")

	// ErrPayloadEmpty is returned when payload is empty.
	ErrPayloadEmpty = errors.New("apns: payload empty")

	// ErrTopicDisallowed is returned when topic is not allowed.
	ErrTopicDisallowed = errors.New("apns: topic disallowed")
)

// APNs response errors (403 Forbidden).
var (
	// ErrBadCertificate is returned when certificate is invalid.
	ErrBadCertificate = errors.New("apns: bad certificate")

	// ErrBadCertificateEnvironment is returned when certificate environment doesn't match.
	ErrBadCertificateEnvironment = errors.New("apns: bad certificate environment")

	// ErrExpiredProviderToken is returned when provider token is expired.
	ErrExpiredProviderToken = errors.New("apns: expired provider token")

	// ErrForbidden is returned when action is forbidden.
	ErrForbidden = errors.New("apns: forbidden")

	// ErrInvalidProviderToken is returned when provider token is invalid.
	ErrInvalidProviderToken = errors.New("apns: invalid provider token")

	// ErrMissingProviderToken is returned when provider token is missing.
	ErrMissingProviderToken = errors.New("apns: missing provider token")
)

// APNs response errors (404, 405, 410).
var (
	// ErrBadPath is returned when path is invalid.
	ErrBadPath = errors.New("apns: bad path")

	// ErrMethodNotAllowed is returned when method is not allowed.
	ErrMethodNotAllowed = errors.New("apns: method not allowed")

	// ErrUnregistered is returned when device is unregistered.
	ErrUnregistered = errors.New("apns: device unregistered")

	// ErrExpiredToken is returned when token is expired.
	ErrExpiredToken = errors.New("apns: token expired")
)

// APNs response errors (429, 500, 503).
var (
	// ErrTooManyProviderTokenUpdates is returned when too many token updates.
	ErrTooManyProviderTokenUpdates = errors.New("apns: too many provider token updates")

	// ErrTooManyRequests is returned when rate limited.
	ErrTooManyRequests = errors.New("apns: too many requests")

	// ErrInternalServerError is returned when APNs has an internal error.
	ErrInternalServerError = errors.New("apns: internal server error")

	// ErrServiceUnavailable is returned when APNs is unavailable.
	ErrServiceUnavailable = errors.New("apns: service unavailable")

	// ErrShutdown is returned when APNs is shutting down.
	ErrShutdown = errors.New("apns: server shutting down")
)

// Error represents an APNs error response.
type Error struct {
	StatusCode int
	Reason     string
	Timestamp  int64
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("apns: %s (status: %d)", e.Reason, e.StatusCode)
	}
	return fmt.Sprintf("apns: error (status: %d)", e.StatusCode)
}

// Is checks if this error matches the target.
func (e *Error) Is(target error) bool {
	switch e.Reason {
	case "BadDeviceToken":
		return errors.Is(target, ErrBadDeviceToken)
	case "Unregistered":
		return errors.Is(target, ErrUnregistered)
	case "DeviceTokenNotForTopic":
		return errors.Is(target, ErrDeviceTokenNotForTopic)
	case "TooManyRequests":
		return errors.Is(target, ErrTooManyRequests)
	case "InternalServerError":
		return errors.Is(target, ErrInternalServerError)
	case "ServiceUnavailable":
		return errors.Is(target, ErrServiceUnavailable)
	}
	return false
}

// IsRetryable returns true if the error is retryable.
func (e *Error) IsRetryable() bool {
	switch e.Reason {
	case "TooManyRequests", "InternalServerError", "ServiceUnavailable", "Shutdown":
		return true
	}
	return e.StatusCode >= 500
}

// IsBadToken checks if the error indicates an invalid device token.
func IsBadToken(err error) bool {
	return errors.Is(err, ErrBadDeviceToken) ||
		errors.Is(err, ErrUnregistered) ||
		errors.Is(err, ErrDeviceTokenNotForTopic) ||
		errors.Is(err, ErrExpiredToken)
}

// IsUnregistered checks if the device is no longer registered.
func IsUnregistered(err error) bool {
	return errors.Is(err, ErrUnregistered) || errors.Is(err, ErrExpiredToken)
}

// IsRateLimited checks if the error is a rate limit error.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrTooManyRequests)
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	var apnsErr *Error
	if errors.As(err, &apnsErr) {
		return apnsErr.IsRetryable()
	}
	return errors.Is(err, ErrTooManyRequests) ||
		errors.Is(err, ErrInternalServerError) ||
		errors.Is(err, ErrServiceUnavailable) ||
		errors.Is(err, ErrShutdown)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrBadCertificate) ||
		errors.Is(err, ErrBadCertificateEnvironment) ||
		errors.Is(err, ErrExpiredProviderToken) ||
		errors.Is(err, ErrInvalidProviderToken) ||
		errors.Is(err, ErrMissingProviderToken)
}
