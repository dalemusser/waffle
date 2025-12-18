// notify/errors.go
package notify

import (
	"errors"
	"fmt"
)

// Common errors.
var (
	// ErrVAPIDKeysRequired is returned when VAPID keys are not provided.
	ErrVAPIDKeysRequired = errors.New("notify: VAPID keys required")

	// ErrSubjectRequired is returned when VAPID subject is not set.
	ErrSubjectRequired = errors.New("notify: VAPID subject required (mailto: or https: URL)")

	// ErrInvalidPEM is returned when PEM data is invalid.
	ErrInvalidPEM = errors.New("notify: invalid PEM data")

	// ErrInvalidKeyType is returned when the key type is not ECDSA.
	ErrInvalidKeyType = errors.New("notify: key must be ECDSA P-256")

	// ErrInvalidCurve is returned when the curve is not P-256.
	ErrInvalidCurve = errors.New("notify: key must use P-256 curve")

	// ErrInvalidPublicKey is returned when the public key is invalid.
	ErrInvalidPublicKey = errors.New("notify: invalid public key")

	// ErrSubscriptionRequired is returned when subscription is nil.
	ErrSubscriptionRequired = errors.New("notify: subscription required")

	// ErrEndpointRequired is returned when endpoint is empty.
	ErrEndpointRequired = errors.New("notify: endpoint required")

	// ErrInvalidEndpoint is returned when endpoint URL is invalid.
	ErrInvalidEndpoint = errors.New("notify: invalid endpoint URL")

	// ErrEndpointNotHTTPS is returned when endpoint is not HTTPS.
	ErrEndpointNotHTTPS = errors.New("notify: endpoint must use HTTPS")

	// ErrP256dhRequired is returned when p256dh key is missing.
	ErrP256dhRequired = errors.New("notify: p256dh key required")

	// ErrAuthRequired is returned when auth secret is missing.
	ErrAuthRequired = errors.New("notify: auth secret required")

	// ErrInvalidP256dh is returned when p256dh key is invalid.
	ErrInvalidP256dh = errors.New("notify: invalid p256dh key (must be 65 bytes uncompressed)")

	// ErrInvalidAuth is returned when auth secret is invalid.
	ErrInvalidAuth = errors.New("notify: invalid auth secret (must be 16 bytes)")

	// ErrPayloadTooLarge is returned when payload exceeds 4KB limit.
	ErrPayloadTooLarge = errors.New("notify: payload too large (max 4096 bytes)")

	// ErrSubscriptionExpired is returned when subscription is no longer valid.
	ErrSubscriptionExpired = errors.New("notify: subscription expired or unsubscribed")

	// ErrRateLimited is returned when rate limited by push service.
	ErrRateLimited = errors.New("notify: rate limited by push service")

	// ErrNoDefaultClient is returned when default client is not set.
	ErrNoDefaultClient = errors.New("notify: default client not set")

	// ErrNoSubscriptions is returned when no subscriptions provided.
	ErrNoSubscriptions = errors.New("notify: no subscriptions provided")
)

// HTTPError represents an HTTP error from the push service.
type HTTPError struct {
	StatusCode int
	Message    string
	Body       string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("notify: HTTP %d %s: %s", e.StatusCode, e.Message, e.Body)
	}
	return fmt.Sprintf("notify: HTTP %d %s", e.StatusCode, e.Message)
}

// Is checks if this error matches the target.
func (e *HTTPError) Is(target error) bool {
	switch e.StatusCode {
	case 404, 410:
		return errors.Is(target, ErrSubscriptionExpired)
	case 413:
		return errors.Is(target, ErrPayloadTooLarge)
	case 429:
		return errors.Is(target, ErrRateLimited)
	}
	return false
}

// IsRetryable returns true if the error is retryable.
func (e *HTTPError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// IsExpired checks if the error indicates an expired subscription.
func IsExpired(err error) bool {
	return errors.Is(err, ErrSubscriptionExpired)
}

// IsRateLimited checks if the error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.IsRetryable()
	}
	return errors.Is(err, ErrRateLimited)
}
