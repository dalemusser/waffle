// httputil/json.go
package httputil

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is a standard JSON error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// WriteJSON writes a JSON response with the given status code.
// If encoding fails, it silently drops the error because at that point
// headers and status have already been sent.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Best-effort: can't safely send another response here.
		_ = err
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
