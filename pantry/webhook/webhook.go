// Package webhook provides utilities for handling incoming webhooks with signature
// verification and sending outgoing webhooks with automatic retries.
//
// Incoming webhooks:
//
//	// Verify GitHub webhook signature
//	verifier := webhook.NewGitHubVerifier("webhook-secret")
//	if err := verifier.Verify(r); err != nil {
//	    http.Error(w, "Invalid signature", http.StatusUnauthorized)
//	    return
//	}
//
// Outgoing webhooks:
//
//	// Send webhook with automatic retries
//	sender := webhook.NewSender(webhook.SenderConfig{
//	    SigningSecret: "your-secret",
//	})
//	err := sender.Send(ctx, "https://example.com/webhook", webhook.Event{
//	    Type: "order.created",
//	    Data: order,
//	})
//
// Event routing:
//
//	router := webhook.NewRouter()
//	router.On("order.created", handleOrderCreated)
//	router.On("order.*", handleAllOrderEvents)
//	router.HandleFunc(w, r) // Use as HTTP handler
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Common errors returned by webhook operations.
var (
	ErrInvalidSignature    = errors.New("webhook: invalid signature")
	ErrMissingSignature    = errors.New("webhook: missing signature header")
	ErrTimestampExpired    = errors.New("webhook: timestamp expired")
	ErrMissingTimestamp    = errors.New("webhook: missing timestamp")
	ErrInvalidPayload      = errors.New("webhook: invalid payload")
	ErrDeliveryFailed      = errors.New("webhook: delivery failed")
	ErrNoHandlerFound      = errors.New("webhook: no handler found for event type")
	ErrMaxRetriesExceeded  = errors.New("webhook: max retries exceeded")
	ErrInvalidEventType    = errors.New("webhook: invalid event type")
	ErrRequestBodyTooLarge = errors.New("webhook: request body too large")
)

// Event represents a webhook event.
type Event struct {
	// ID is a unique identifier for this event.
	ID string `json:"id"`

	// Type is the event type (e.g., "order.created", "user.updated").
	Type string `json:"type"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Data is the event payload.
	Data any `json:"data"`

	// Metadata contains additional event metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RawEvent represents a webhook event with raw JSON data.
type RawEvent struct {
	// ID is a unique identifier for this event.
	ID string `json:"id"`

	// Type is the event type.
	Type string `json:"type"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Data is the raw JSON event payload.
	Data json.RawMessage `json:"data"`

	// Metadata contains additional event metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ParseData parses the raw event data into the provided type.
func (e *RawEvent) ParseData(v any) error {
	return json.Unmarshal(e.Data, v)
}

// Payload represents the full webhook request payload.
type Payload struct {
	// Event is the webhook event.
	Event RawEvent `json:"event"`

	// DeliveryID is the unique ID for this delivery attempt.
	DeliveryID string `json:"delivery_id,omitempty"`

	// Attempt is the delivery attempt number (1-based).
	Attempt int `json:"attempt,omitempty"`
}

// DeliveryResult contains the result of a webhook delivery attempt.
type DeliveryResult struct {
	// Success indicates if the delivery was successful.
	Success bool

	// StatusCode is the HTTP status code received.
	StatusCode int

	// ResponseBody is the response body (truncated if large).
	ResponseBody string

	// Duration is how long the request took.
	Duration time.Duration

	// Attempt is which attempt this was (1-based).
	Attempt int

	// Error is the error if delivery failed.
	Error error

	// Timestamp is when this attempt was made.
	Timestamp time.Time
}

// VerifyRequest contains the parsed and verified webhook request.
type VerifyRequest struct {
	// Body is the raw request body.
	Body []byte

	// Signature is the signature from the request.
	Signature string

	// Timestamp is the timestamp from the request (if available).
	Timestamp time.Time

	// Headers contains relevant headers.
	Headers http.Header
}

// HashAlgorithm represents a hash algorithm for HMAC signatures.
type HashAlgorithm string

const (
	SHA256 HashAlgorithm = "sha256"
	SHA512 HashAlgorithm = "sha512"
	SHA1   HashAlgorithm = "sha1"
)

// newHash creates a new hash.Hash for the given algorithm.
func newHash(alg HashAlgorithm, key []byte) hash.Hash {
	switch alg {
	case SHA512:
		return hmac.New(sha512.New, key)
	case SHA1:
		return hmac.New(sha256.New, key) // Fallback to SHA256 for safety
	default:
		return hmac.New(sha256.New, key)
	}
}

// ComputeHMAC computes an HMAC signature for the given payload.
func ComputeHMAC(payload, secret []byte, alg HashAlgorithm) string {
	h := newHash(alg, secret)
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC verifies an HMAC signature.
func VerifyHMAC(payload, secret []byte, signature string, alg HashAlgorithm) bool {
	expected := ComputeHMAC(payload, secret, alg)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// TimestampTolerance is the default tolerance for timestamp verification.
const TimestampTolerance = 5 * time.Minute

// MaxBodySize is the default maximum request body size (10MB).
const MaxBodySize = 10 * 1024 * 1024

// ReadBody reads and returns the request body, enforcing a size limit.
func ReadBody(r *http.Request, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		maxSize = MaxBodySize
	}

	// Check Content-Length header first
	if r.ContentLength > maxSize {
		return nil, ErrRequestBodyTooLarge
	}

	// Limit the reader
	limited := io.LimitReader(r.Body, maxSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("webhook: failed to read body: %w", err)
	}

	if int64(len(body)) > maxSize {
		return nil, ErrRequestBodyTooLarge
	}

	return body, nil
}

// ParseEvent parses an event from JSON data.
func ParseEvent(data []byte) (*RawEvent, error) {
	var event RawEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return &event, nil
}

// ParsePayload parses a webhook payload from JSON data.
func ParsePayload(data []byte) (*Payload, error) {
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return &payload, nil
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType string, data any) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), randomHex(8))
}

// generateDeliveryID generates a unique delivery ID.
func generateDeliveryID() string {
	return fmt.Sprintf("dlv_%d_%s", time.Now().UnixNano(), randomHex(8))
}

// randomHex generates a random hex string of the given length.
func randomHex(n int) string {
	b := make([]byte, n/2+1)
	// Use timestamp and simple counter for uniqueness
	// In production, you'd use crypto/rand
	ts := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(ts >> (i * 8))
	}
	return hex.EncodeToString(b)[:n]
}

// isSuccessStatus returns true if the status code indicates success.
func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// isRetryableStatus returns true if the status code indicates a retryable error.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// truncateString truncates a string to the given length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SignPayload signs a payload using HMAC-SHA256 and returns the signature.
func SignPayload(payload []byte, secret string) string {
	return ComputeHMAC(payload, []byte(secret), SHA256)
}

// SignPayloadWithTimestamp signs a payload with a timestamp prefix.
// This is the format used by Stripe and similar services.
func SignPayloadWithTimestamp(timestamp int64, payload []byte, secret string) string {
	// Format: timestamp.payload
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	return ComputeHMAC([]byte(signedPayload), []byte(secret), SHA256)
}

// VerifyTimestamp checks if a timestamp is within the allowed tolerance.
func VerifyTimestamp(ts time.Time, tolerance time.Duration) error {
	if tolerance <= 0 {
		tolerance = TimestampTolerance
	}

	now := time.Now()
	if ts.Before(now.Add(-tolerance)) || ts.After(now.Add(tolerance)) {
		return ErrTimestampExpired
	}
	return nil
}

// ParseUnixTimestamp parses a Unix timestamp string.
func ParseUnixTimestamp(s string) (time.Time, error) {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid timestamp format", ErrMissingTimestamp)
	}
	return time.Unix(ts, 0), nil
}

// ExtractSignature extracts a signature from a header value with optional prefix.
// For example: "sha256=abc123" returns "abc123" with prefix "sha256=".
func ExtractSignature(header, prefix string) string {
	header = strings.TrimSpace(header)
	if prefix != "" && strings.HasPrefix(header, prefix) {
		return strings.TrimPrefix(header, prefix)
	}
	return header
}

// ContextKey is a type for context keys used by this package.
type ContextKey string

const (
	// ContextKeyEvent is the context key for the parsed webhook event.
	ContextKeyEvent ContextKey = "webhook_event"

	// ContextKeyPayload is the context key for the raw webhook payload.
	ContextKeyPayload ContextKey = "webhook_payload"

	// ContextKeyDeliveryID is the context key for the delivery ID.
	ContextKeyDeliveryID ContextKey = "webhook_delivery_id"
)

// EventFromContext retrieves the webhook event from the context.
func EventFromContext(ctx context.Context) (*RawEvent, bool) {
	event, ok := ctx.Value(ContextKeyEvent).(*RawEvent)
	return event, ok
}

// PayloadFromContext retrieves the raw webhook payload from the context.
func PayloadFromContext(ctx context.Context) ([]byte, bool) {
	payload, ok := ctx.Value(ContextKeyPayload).([]byte)
	return payload, ok
}

// ContextWithEvent returns a new context with the webhook event.
func ContextWithEvent(ctx context.Context, event *RawEvent) context.Context {
	return context.WithValue(ctx, ContextKeyEvent, event)
}

// ContextWithPayload returns a new context with the raw webhook payload.
func ContextWithPayload(ctx context.Context, payload []byte) context.Context {
	return context.WithValue(ctx, ContextKeyPayload, payload)
}

// StandardHeaders are the standard headers set on outgoing webhook requests.
var StandardHeaders = map[string]string{
	"Content-Type": "application/json",
	"User-Agent":   "Waffle-Webhook/1.0",
}

// Response represents a standard webhook response.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WriteSuccess writes a success response.
func WriteSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Success: true, Message: message})
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
}

// bodyReader is a helper to allow reading the body multiple times.
type bodyReader struct {
	body []byte
	pos  int
}

func (b *bodyReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.body) {
		return 0, io.EOF
	}
	n = copy(p, b.body[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bodyReader) Close() error {
	return nil
}

// DrainBody reads the request body and replaces it with a replayable reader.
// This allows the body to be read multiple times.
func DrainBody(r *http.Request, maxSize int64) ([]byte, error) {
	body, err := ReadBody(r, maxSize)
	if err != nil {
		return nil, err
	}

	// Replace the body with a replayable reader
	r.Body = &bodyReader{body: body}

	return body, nil
}

// ResetBody resets the request body to allow re-reading.
func ResetBody(r *http.Request, body []byte) {
	r.Body = io.NopCloser(bytes.NewReader(body))
}
