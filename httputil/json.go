// httputil/json.go
package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
)

// ErrorResponse is a standard JSON error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// jsonLogger is a package-level logger for encoding errors. Use SetJSONLogger to configure.
var jsonLogger JSONLogger

// JSONLogger is a minimal interface for logging JSON encoding errors.
type JSONLogger interface {
	Error(msg string, args ...any)
}

// SetJSONLogger configures the logger used for JSON encoding errors.
// This should be called once during application startup.
func SetJSONLogger(logger JSONLogger) {
	jsonLogger = logger
}

// WriteJSON writes a JSON response with the given status code.
// If encoding fails, the error is logged (if a logger is configured via
// SetJSONLogger) because headers and status have already been sent and
// we can't send another response.
//
// Invalid status codes (outside 100-599) are clamped to 500 Internal Server Error
// to prevent undefined behavior in net/http.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	// Clamp invalid status codes to prevent undefined behavior.
	// Valid HTTP status codes are 100-599.
	if status < 100 || status > 599 {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Can't send another response after headers are written.
		// Log the error if a logger is configured, including the type for debugging.
		// Wrap in recover to prevent logger panics from crashing the server.
		if jsonLogger != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Logger panicked - write to stderr as last resort
						fmt.Fprintf(os.Stderr, "httputil: logger panic while reporting json error: %v\n", r)
					}
				}()
				typeName := "nil"
				if v != nil {
					typeName = reflect.TypeOf(v).String()
				}
				jsonLogger.Error(fmt.Sprintf("json encoding failed after headers sent (type: %s): %v", typeName, err))
			}()
		}
	}
}

// JSONError writes a structured JSON error with an error code and message.
func JSONError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Error:   code,
		Message: message,
	}
	WriteJSON(w, status, resp)
}

// JSONErrorSimple is a shorthand for errors where the message itself is the code.
func JSONErrorSimple(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{
		Error: message,
	}
	WriteJSON(w, status, resp)
}

// BindJSON decodes the request body as JSON into v.
//
// It returns a user-friendly error if the body is empty, malformed, or contains
// unknown fields. The error messages are safe to return to clients.
//
// Example:
//
//	var req CreateUserRequest
//	if err := httputil.BindJSON(r, &req); err != nil {
//	    httputil.JSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
//	    return
//	}
func BindJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	// ContentLength semantics:
	//   0  = explicitly empty body (Content-Length: 0) → reject early
	//  -1  = chunked/unknown length → must attempt decode; empty chunked body
	//        will fail with EOF, converted to "request body is empty" by parseJSONError
	//  >0  = known content length → proceed to decode
	if r.ContentLength == 0 {
		return errors.New("request body is empty")
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return parseJSONError(err)
	}

	// Check for extraneous data after the JSON object
	if dec.More() {
		return errors.New("request body contains multiple JSON values")
	}

	return nil
}

// BindJSONAllowUnknown is like BindJSON but permits unknown fields in the JSON.
// Use this when you want to be lenient about extra fields in the request.
// Like BindJSON, it rejects request bodies containing multiple JSON values.
func BindJSONAllowUnknown(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	// ContentLength is 0 for explicitly empty bodies, -1 for chunked/unknown.
	// We reject 0 early; chunked requests with empty content will fail at decode
	// with EOF, which parseJSONError converts to "request body is empty".
	if r.ContentLength == 0 {
		return errors.New("request body is empty")
	}

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(v); err != nil {
		return parseJSONError(err)
	}

	// Check for extraneous data after the JSON object
	if dec.More() {
		return errors.New("request body contains multiple JSON values")
	}

	return nil
}

// parseJSONError converts json decoding errors into user-friendly messages.
func parseJSONError(err error) error {
	if err == nil {
		return nil
	}

	// Empty body
	if errors.Is(err, io.EOF) {
		return errors.New("request body is empty")
	}

	// Syntax error
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)
	}

	// Type mismatch
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Errorf("invalid value for field %q: expected %s", typeErr.Field, typeErr.Type.String())
	}

	// Unknown field (when DisallowUnknownFields is set)
	// Error format: "json: unknown field \"fieldname\""
	if strings.HasPrefix(err.Error(), "json: unknown field") {
		// Extract field name and strip surrounding quotes for cleaner output
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		field = strings.Trim(field, "\"")
		return fmt.Errorf("unknown field %q", field)
	}

	// Body too large (from http.MaxBytesReader)
	if err.Error() == "http: request body too large" {
		return errors.New("request body too large")
	}

	// Generic fallback
	return errors.New("invalid JSON in request body")
}
