// fcm/errors.go
package fcm

import (
	"errors"
	"fmt"
)

// Common errors.
var (
	// ErrCredentialsRequired is returned when credentials are not provided.
	ErrCredentialsRequired = errors.New("fcm: credentials required")

	// ErrInvalidCredentials is returned when credentials are invalid.
	ErrInvalidCredentials = errors.New("fcm: invalid credentials")

	// ErrInvalidPrivateKey is returned when the private key is invalid.
	ErrInvalidPrivateKey = errors.New("fcm: invalid private key")

	// ErrProjectIDRequired is returned when project ID is not specified.
	ErrProjectIDRequired = errors.New("fcm: project ID required")

	// ErrMissingProjectID is returned when project ID is not specified.
	ErrMissingProjectID = errors.New("fcm: missing project ID")

	// ErrMessageRequired is returned when message is nil.
	ErrMessageRequired = errors.New("fcm: message required")

	// ErrTargetRequired is returned when no target is specified.
	ErrTargetRequired = errors.New("fcm: message must have exactly one target (token, topic, or condition)")

	// ErrTokensRequired is returned when no tokens are provided.
	ErrTokensRequired = errors.New("fcm: tokens required")

	// ErrMessagesRequired is returned when no messages are provided.
	ErrMessagesRequired = errors.New("fcm: messages required")

	// ErrTopicRequired is returned when topic is empty.
	ErrTopicRequired = errors.New("fcm: topic required")

	// ErrMissingTarget is returned when no target (token, topic, or condition) is specified.
	ErrMissingTarget = errors.New("fcm: message must have exactly one target (token, topic, or condition)")

	// ErrMultipleTargets is returned when multiple targets are specified.
	ErrMultipleTargets = errors.New("fcm: message must have exactly one target (token, topic, or condition)")

	// ErrInvalidToken is returned when a device token is invalid.
	ErrInvalidToken = errors.New("fcm: invalid registration token")

	// ErrTokenNotRegistered is returned when a token is no longer registered.
	ErrTokenNotRegistered = errors.New("fcm: token not registered")

	// ErrTooManyTokens is returned when too many tokens are provided.
	ErrTooManyTokens = errors.New("fcm: too many tokens (max 500)")

	// ErrQuotaExceeded is returned when the message quota is exceeded.
	ErrQuotaExceeded = errors.New("fcm: message quota exceeded")

	// ErrServerError is returned when FCM server returns an error.
	ErrServerError = errors.New("fcm: server error")

	// ErrTimeout is returned when the request times out.
	ErrTimeout = errors.New("fcm: request timeout")

	// ErrUnavailable is returned when FCM is temporarily unavailable.
	ErrUnavailable = errors.New("fcm: service unavailable")

	// ErrInvalidPayload is returned when the message payload is invalid.
	ErrInvalidPayload = errors.New("fcm: invalid message payload")

	// ErrPayloadTooLarge is returned when the message payload exceeds size limits.
	ErrPayloadTooLarge = errors.New("fcm: payload too large (max 4KB)")

	// ErrInvalidTopic is returned when a topic name is invalid.
	ErrInvalidTopic = errors.New("fcm: invalid topic name")

	// ErrAuthenticationFailed is returned when authentication fails.
	ErrAuthenticationFailed = errors.New("fcm: authentication failed")

	// ErrSenderMismatch is returned when the sender doesn't match.
	ErrSenderMismatch = errors.New("fcm: sender ID mismatch")
)

// ErrorCode represents FCM error codes.
type ErrorCode string

const (
	ErrorCodeUnspecified         ErrorCode = "UNSPECIFIED_ERROR"
	ErrorCodeInvalidArgument     ErrorCode = "INVALID_ARGUMENT"
	ErrorCodeUnregistered        ErrorCode = "UNREGISTERED"
	ErrorCodeSenderIDMismatch    ErrorCode = "SENDER_ID_MISMATCH"
	ErrorCodeQuotaExceeded       ErrorCode = "QUOTA_EXCEEDED"
	ErrorCodeUnavailable         ErrorCode = "UNAVAILABLE"
	ErrorCodeInternal            ErrorCode = "INTERNAL"
	ErrorCodeThirdPartyAuthError ErrorCode = "THIRD_PARTY_AUTH_ERROR"
)

// APIError represents an error returned by the FCM API.
type APIError struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Code is the FCM error code.
	Code ErrorCode

	// Message is the error message.
	Message string

	// Details contains additional error details.
	Details map[string]any
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("fcm: %s (code: %s, status: %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("fcm: error code %s (status: %d)", e.Code, e.StatusCode)
}

// Is checks if this error matches the target.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrInvalidToken:
		return e.Code == ErrorCodeInvalidArgument
	case ErrTokenNotRegistered:
		return e.Code == ErrorCodeUnregistered
	case ErrQuotaExceeded:
		return e.Code == ErrorCodeQuotaExceeded
	case ErrUnavailable:
		return e.Code == ErrorCodeUnavailable
	case ErrServerError:
		return e.Code == ErrorCodeInternal
	case ErrSenderMismatch:
		return e.Code == ErrorCodeSenderIDMismatch
	case ErrAuthenticationFailed:
		return e.Code == ErrorCodeThirdPartyAuthError
	}
	return false
}

// IsRetryable returns true if the error is retryable.
func (e *APIError) IsRetryable() bool {
	switch e.Code {
	case ErrorCodeUnavailable, ErrorCodeInternal:
		return true
	}
	return e.StatusCode >= 500
}

// SendError represents an error for a specific message in a batch.
type SendError struct {
	// Index is the index of the failed message.
	Index int

	// Token is the token that failed (if applicable).
	Token string

	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *SendError) Error() string {
	if e.Token != "" {
		return fmt.Sprintf("fcm: send failed for token %s at index %d: %v", e.Token, e.Index, e.Err)
	}
	return fmt.Sprintf("fcm: send failed at index %d: %v", e.Index, e.Err)
}

// Unwrap returns the underlying error.
func (e *SendError) Unwrap() error {
	return e.Err
}

// BatchError represents multiple errors from a batch send.
type BatchError struct {
	// Errors is the list of individual errors.
	Errors []*SendError

	// SuccessCount is the number of successful sends.
	SuccessCount int

	// FailureCount is the number of failed sends.
	FailureCount int
}

// Error implements the error interface.
func (e *BatchError) Error() string {
	return fmt.Sprintf("fcm: batch send had %d failures out of %d messages",
		e.FailureCount, e.SuccessCount+e.FailureCount)
}

// IsPartialSuccess returns true if some messages were sent successfully.
func (e *BatchError) IsPartialSuccess() bool {
	return e.SuccessCount > 0
}

// FailedTokens returns the list of tokens that failed.
func (e *BatchError) FailedTokens() []string {
	tokens := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err.Token != "" {
			tokens = append(tokens, err.Token)
		}
	}
	return tokens
}

// UnregisteredTokens returns tokens that are no longer registered.
func (e *BatchError) UnregisteredTokens() []string {
	var tokens []string
	for _, err := range e.Errors {
		if err.Token != "" && errors.Is(err.Err, ErrTokenNotRegistered) {
			tokens = append(tokens, err.Token)
		}
	}
	return tokens
}

// ValidationError represents a message validation error.
type ValidationError struct {
	// Field is the field that failed validation.
	Field string

	// Message is the validation error message.
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("fcm: validation error on field %q: %s", e.Field, e.Message)
}

// TopicManagementError represents an error from topic management operations.
type TopicManagementError struct {
	// Topic is the topic that had an error.
	Topic string

	// FailedTokens is the list of tokens that failed.
	FailedTokens []string

	// Errors maps tokens to their errors.
	Errors map[string]error
}

// Error implements the error interface.
func (e *TopicManagementError) Error() string {
	return fmt.Sprintf("fcm: topic %q operation failed for %d tokens", e.Topic, len(e.FailedTokens))
}

// parseErrorCode converts an FCM error string to an ErrorCode.
func parseErrorCode(code string) ErrorCode {
	switch code {
	case "INVALID_ARGUMENT":
		return ErrorCodeInvalidArgument
	case "UNREGISTERED":
		return ErrorCodeUnregistered
	case "SENDER_ID_MISMATCH":
		return ErrorCodeSenderIDMismatch
	case "QUOTA_EXCEEDED":
		return ErrorCodeQuotaExceeded
	case "UNAVAILABLE":
		return ErrorCodeUnavailable
	case "INTERNAL":
		return ErrorCodeInternal
	case "THIRD_PARTY_AUTH_ERROR":
		return ErrorCodeThirdPartyAuthError
	default:
		return ErrorCodeUnspecified
	}
}

// newAPIError creates an APIError from HTTP response data.
func newAPIError(statusCode int, code, message string, details map[string]any) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       parseErrorCode(code),
		Message:    message,
		Details:    details,
	}
}

// IsUnregistered checks if the error indicates the token is no longer registered.
func IsUnregistered(err error) bool {
	return errors.Is(err, ErrTokenNotRegistered)
}

// IsQuotaExceeded checks if the error indicates quota was exceeded.
func IsQuotaExceeded(err error) bool {
	return errors.Is(err, ErrQuotaExceeded)
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrServerError) || errors.Is(err, ErrTimeout)
}

// IsSenderMismatch checks if the error indicates a sender mismatch.
func IsSenderMismatch(err error) bool {
	return errors.Is(err, ErrSenderMismatch)
}

// IsInvalidToken checks if the error indicates an invalid token.
func IsInvalidToken(err error) bool {
	return errors.Is(err, ErrInvalidToken)
}
