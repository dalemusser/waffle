// errors/errors.go
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error represents a structured application error with code, message, and HTTP status.
type Error struct {
	// Code is a machine-readable error code (e.g., "not_found", "validation_failed")
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Status is the HTTP status code (not included in JSON by default)
	Status int `json:"-"`

	// Details contains additional error context (optional)
	Details map[string]any `json:"details,omitempty"`

	// Err is the underlying error (not included in JSON)
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *Error) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// Wrap wraps an underlying error.
func (e *Error) Wrap(err error) *Error {
	e.Err = err
	return e
}

// HTTPStatus returns the HTTP status code for the error.
func (e *Error) HTTPStatus() int {
	if e.Status == 0 {
		return http.StatusInternalServerError
	}
	return e.Status
}

// MarshalJSON implements json.Marshaler.
func (e *Error) MarshalJSON() ([]byte, error) {
	type alias Error
	return json.Marshal(&struct {
		*alias
	}{
		alias: (*alias)(e),
	})
}

// New creates a new Error with code, message, and HTTP status.
func New(code, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// Wrap creates a new Error wrapping an existing error.
func Wrap(err error, code, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// From extracts an *Error from err if possible, or wraps it as an internal error.
func From(err error) *Error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		return e
	}

	return &Error{
		Code:    "internal_error",
		Message: "an internal error occurred",
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

// Is reports whether any error in err's chain matches target.
// Re-exported from standard errors package for convenience.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
// Re-exported from standard errors package for convenience.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Join returns an error that wraps the given errors.
// Re-exported from standard errors package for convenience.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Common error codes as constants for consistency.
const (
	CodeBadRequest          = "bad_request"
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeMethodNotAllowed    = "method_not_allowed"
	CodeConflict            = "conflict"
	CodeGone                = "gone"
	CodeUnprocessableEntity = "unprocessable_entity"
	CodeTooManyRequests     = "too_many_requests"
	CodeInternalError       = "internal_error"
	CodeNotImplemented      = "not_implemented"
	CodeServiceUnavailable  = "service_unavailable"
	CodeTimeout             = "timeout"
	CodeValidationFailed    = "validation_failed"
	CodeAlreadyExists       = "already_exists"
	CodeInvalidInput        = "invalid_input"
	CodeAuthenticationFailed = "authentication_failed"
	CodePermissionDenied    = "permission_denied"
)

// Pre-defined common errors. Clone these with specific messages using the constructor functions.

// BadRequest creates a 400 Bad Request error.
func BadRequest(message string) *Error {
	return New(CodeBadRequest, message, http.StatusBadRequest)
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(message string) *Error {
	return New(CodeForbidden, message, http.StatusForbidden)
}

// NotFound creates a 404 Not Found error.
func NotFound(message string) *Error {
	return New(CodeNotFound, message, http.StatusNotFound)
}

// MethodNotAllowed creates a 405 Method Not Allowed error.
func MethodNotAllowed(message string) *Error {
	return New(CodeMethodNotAllowed, message, http.StatusMethodNotAllowed)
}

// Conflict creates a 409 Conflict error.
func Conflict(message string) *Error {
	return New(CodeConflict, message, http.StatusConflict)
}

// Gone creates a 410 Gone error.
func Gone(message string) *Error {
	return New(CodeGone, message, http.StatusGone)
}

// UnprocessableEntity creates a 422 Unprocessable Entity error.
func UnprocessableEntity(message string) *Error {
	return New(CodeUnprocessableEntity, message, http.StatusUnprocessableEntity)
}

// TooManyRequests creates a 429 Too Many Requests error.
func TooManyRequests(message string) *Error {
	return New(CodeTooManyRequests, message, http.StatusTooManyRequests)
}

// Internal creates a 500 Internal Server Error.
func Internal(message string) *Error {
	return New(CodeInternalError, message, http.StatusInternalServerError)
}

// NotImplemented creates a 501 Not Implemented error.
func NotImplemented(message string) *Error {
	return New(CodeNotImplemented, message, http.StatusNotImplemented)
}

// ServiceUnavailable creates a 503 Service Unavailable error.
func ServiceUnavailable(message string) *Error {
	return New(CodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

// Timeout creates a 504 Gateway Timeout error.
func Timeout(message string) *Error {
	return New(CodeTimeout, message, http.StatusGatewayTimeout)
}

// Validation creates a 400 Bad Request error with validation code.
func Validation(message string) *Error {
	return New(CodeValidationFailed, message, http.StatusBadRequest)
}

// AlreadyExists creates a 409 Conflict error with already_exists code.
func AlreadyExists(message string) *Error {
	return New(CodeAlreadyExists, message, http.StatusConflict)
}

// InvalidInput creates a 400 Bad Request error with invalid_input code.
func InvalidInput(message string) *Error {
	return New(CodeInvalidInput, message, http.StatusBadRequest)
}

// AuthenticationFailed creates a 401 Unauthorized error with authentication_failed code.
func AuthenticationFailed(message string) *Error {
	return New(CodeAuthenticationFailed, message, http.StatusUnauthorized)
}

// PermissionDenied creates a 403 Forbidden error with permission_denied code.
func PermissionDenied(message string) *Error {
	return New(CodePermissionDenied, message, http.StatusForbidden)
}

// ValidationErrors holds multiple field-level validation errors.
type ValidationErrors struct {
	Errors []FieldError `json:"errors"`
}

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Error implements the error interface.
func (v *ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s: %s", v.Errors[0].Field, v.Errors[0].Message)
}

// Add adds a field error.
func (v *ValidationErrors) Add(field, message string) *ValidationErrors {
	v.Errors = append(v.Errors, FieldError{Field: field, Message: message})
	return v
}

// AddWithCode adds a field error with a code.
func (v *ValidationErrors) AddWithCode(field, message, code string) *ValidationErrors {
	v.Errors = append(v.Errors, FieldError{Field: field, Message: message, Code: code})
	return v
}

// HasErrors returns true if there are validation errors.
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// ToError converts ValidationErrors to an *Error if there are errors.
func (v *ValidationErrors) ToError() *Error {
	if !v.HasErrors() {
		return nil
	}
	return Validation("validation failed").WithDetail("errors", v.Errors)
}

// NewValidationErrors creates a new ValidationErrors.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]FieldError, 0),
	}
}
