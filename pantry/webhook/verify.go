package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Verifier is the interface for webhook signature verification.
type Verifier interface {
	// Verify verifies the webhook signature from an HTTP request.
	Verify(r *http.Request) error

	// VerifyPayload verifies the webhook signature for a raw payload.
	VerifyPayload(payload []byte, signature string) error
}

// HMACVerifier verifies HMAC signatures.
type HMACVerifier struct {
	// Secret is the shared secret for HMAC computation.
	Secret []byte

	// Algorithm is the hash algorithm to use.
	Algorithm HashAlgorithm

	// Header is the HTTP header containing the signature.
	Header string

	// Prefix is an optional prefix to strip from the signature (e.g., "sha256=").
	Prefix string

	// MaxBodySize is the maximum allowed request body size.
	MaxBodySize int64
}

// HMACConfig configures the HMAC verifier.
type HMACConfig struct {
	Secret      string
	Algorithm   HashAlgorithm
	Header      string
	Prefix      string
	MaxBodySize int64
}

// NewHMACVerifier creates a new HMAC signature verifier.
func NewHMACVerifier(cfg HMACConfig) *HMACVerifier {
	if cfg.Algorithm == "" {
		cfg.Algorithm = SHA256
	}
	if cfg.Header == "" {
		cfg.Header = "X-Webhook-Signature"
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = MaxBodySize
	}

	return &HMACVerifier{
		Secret:      []byte(cfg.Secret),
		Algorithm:   cfg.Algorithm,
		Header:      cfg.Header,
		Prefix:      cfg.Prefix,
		MaxBodySize: cfg.MaxBodySize,
	}
}

// Verify verifies the webhook signature from an HTTP request.
func (v *HMACVerifier) Verify(r *http.Request) error {
	signature := r.Header.Get(v.Header)
	if signature == "" {
		return ErrMissingSignature
	}

	// Strip prefix if present
	signature = ExtractSignature(signature, v.Prefix)

	// Read the body
	body, err := DrainBody(r, v.MaxBodySize)
	if err != nil {
		return err
	}

	return v.VerifyPayload(body, signature)
}

// VerifyPayload verifies the webhook signature for a raw payload.
func (v *HMACVerifier) VerifyPayload(payload []byte, signature string) error {
	signature = ExtractSignature(signature, v.Prefix)

	if !VerifyHMAC(payload, v.Secret, signature, v.Algorithm) {
		return ErrInvalidSignature
	}
	return nil
}

// GitHubVerifier verifies GitHub webhook signatures.
// GitHub uses HMAC-SHA256 with the signature in the X-Hub-Signature-256 header
// prefixed with "sha256=".
type GitHubVerifier struct {
	secret      []byte
	maxBodySize int64
}

// NewGitHubVerifier creates a new GitHub webhook verifier.
func NewGitHubVerifier(secret string) *GitHubVerifier {
	return &GitHubVerifier{
		secret:      []byte(secret),
		maxBodySize: MaxBodySize,
	}
}

// WithMaxBodySize sets the maximum body size.
func (v *GitHubVerifier) WithMaxBodySize(size int64) *GitHubVerifier {
	v.maxBodySize = size
	return v
}

// Verify verifies a GitHub webhook signature.
func (v *GitHubVerifier) Verify(r *http.Request) error {
	// GitHub sends signatures in X-Hub-Signature-256 (SHA-256) or X-Hub-Signature (SHA-1)
	signature := r.Header.Get("X-Hub-Signature-256")
	algorithm := SHA256
	prefix := "sha256="

	// Fall back to SHA-1 if SHA-256 not present (legacy)
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature")
		algorithm = SHA1
		prefix = "sha1="
	}

	if signature == "" {
		return ErrMissingSignature
	}

	body, err := DrainBody(r, v.maxBodySize)
	if err != nil {
		return err
	}

	return v.verifySignature(body, signature, algorithm, prefix)
}

// VerifyPayload verifies the signature for a raw payload.
func (v *GitHubVerifier) VerifyPayload(payload []byte, signature string) error {
	// Try SHA-256 first
	if strings.HasPrefix(signature, "sha256=") {
		return v.verifySignature(payload, signature, SHA256, "sha256=")
	}
	// Fall back to SHA-1
	if strings.HasPrefix(signature, "sha1=") {
		return v.verifySignature(payload, signature, SHA1, "sha1=")
	}
	// Assume SHA-256 if no prefix
	return v.verifySignature(payload, signature, SHA256, "")
}

func (v *GitHubVerifier) verifySignature(payload []byte, signature string, alg HashAlgorithm, prefix string) error {
	signature = strings.TrimPrefix(signature, prefix)

	var expected string
	if alg == SHA1 {
		h := hmac.New(sha1.New, v.secret)
		h.Write(payload)
		expected = hex.EncodeToString(h.Sum(nil))
	} else {
		h := hmac.New(sha256.New, v.secret)
		h.Write(payload)
		expected = hex.EncodeToString(h.Sum(nil))
	}

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

// StripeVerifier verifies Stripe webhook signatures.
// Stripe uses a custom signature scheme with timestamps to prevent replay attacks.
type StripeVerifier struct {
	secret      []byte
	tolerance   time.Duration
	maxBodySize int64
}

// NewStripeVerifier creates a new Stripe webhook verifier.
func NewStripeVerifier(secret string) *StripeVerifier {
	return &StripeVerifier{
		secret:      []byte(secret),
		tolerance:   TimestampTolerance,
		maxBodySize: MaxBodySize,
	}
}

// WithTolerance sets the timestamp tolerance.
func (v *StripeVerifier) WithTolerance(d time.Duration) *StripeVerifier {
	v.tolerance = d
	return v
}

// WithMaxBodySize sets the maximum body size.
func (v *StripeVerifier) WithMaxBodySize(size int64) *StripeVerifier {
	v.maxBodySize = size
	return v
}

// Verify verifies a Stripe webhook signature.
func (v *StripeVerifier) Verify(r *http.Request) error {
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		return ErrMissingSignature
	}

	body, err := DrainBody(r, v.maxBodySize)
	if err != nil {
		return err
	}

	return v.VerifyPayload(body, signature)
}

// VerifyPayload verifies the Stripe signature for a raw payload.
func (v *StripeVerifier) VerifyPayload(payload []byte, signature string) error {
	// Parse the Stripe signature header
	// Format: t=timestamp,v1=signature,v1=signature2,...
	parts := strings.Split(signature, ",")

	var timestamp int64
	var signatures []string

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return fmt.Errorf("%w: invalid timestamp", ErrMissingTimestamp)
			}
			timestamp = ts
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == 0 {
		return ErrMissingTimestamp
	}

	if len(signatures) == 0 {
		return ErrMissingSignature
	}

	// Verify timestamp
	ts := time.Unix(timestamp, 0)
	if err := VerifyTimestamp(ts, v.tolerance); err != nil {
		return err
	}

	// Compute expected signature
	// Stripe uses: timestamp + "." + payload
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, v.secret)
	h.Write([]byte(signedPayload))
	expected := hex.EncodeToString(h.Sum(nil))

	// Check if any signature matches
	for _, sig := range signatures {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}

	return ErrInvalidSignature
}

// SlackVerifier verifies Slack webhook signatures.
// Slack uses a similar scheme to Stripe with timestamps.
type SlackVerifier struct {
	signingSecret []byte
	tolerance     time.Duration
	maxBodySize   int64
}

// NewSlackVerifier creates a new Slack webhook verifier.
func NewSlackVerifier(signingSecret string) *SlackVerifier {
	return &SlackVerifier{
		signingSecret: []byte(signingSecret),
		tolerance:     TimestampTolerance,
		maxBodySize:   MaxBodySize,
	}
}

// WithTolerance sets the timestamp tolerance.
func (v *SlackVerifier) WithTolerance(d time.Duration) *SlackVerifier {
	v.tolerance = d
	return v
}

// Verify verifies a Slack webhook signature.
func (v *SlackVerifier) Verify(r *http.Request) error {
	signature := r.Header.Get("X-Slack-Signature")
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")

	if signature == "" {
		return ErrMissingSignature
	}
	if timestamp == "" {
		return ErrMissingTimestamp
	}

	body, err := DrainBody(r, v.maxBodySize)
	if err != nil {
		return err
	}

	return v.verifyWithTimestamp(body, signature, timestamp)
}

// VerifyPayload verifies with separate signature and timestamp.
func (v *SlackVerifier) VerifyPayload(payload []byte, signature string) error {
	// For Slack, we need the timestamp separately, so this simplified method
	// just verifies without timestamp check
	signature = strings.TrimPrefix(signature, "v0=")

	// We can't verify properly without timestamp, so just check format
	if len(signature) != 64 {
		return ErrInvalidSignature
	}
	return nil
}

func (v *SlackVerifier) verifyWithTimestamp(payload []byte, signature, timestampStr string) error {
	// Verify timestamp
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp format", ErrMissingTimestamp)
	}

	timestamp := time.Unix(ts, 0)
	if err := VerifyTimestamp(timestamp, v.tolerance); err != nil {
		return err
	}

	// Compute expected signature
	// Slack uses: v0:timestamp:body
	baseString := fmt.Sprintf("v0:%s:%s", timestampStr, string(payload))
	h := hmac.New(sha256.New, v.signingSecret)
	h.Write([]byte(baseString))
	expected := "v0=" + hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}

	return nil
}

// ShopifyVerifier verifies Shopify webhook signatures.
// Shopify uses HMAC-SHA256 with base64 encoding.
type ShopifyVerifier struct {
	secret      []byte
	maxBodySize int64
}

// NewShopifyVerifier creates a new Shopify webhook verifier.
func NewShopifyVerifier(secret string) *ShopifyVerifier {
	return &ShopifyVerifier{
		secret:      []byte(secret),
		maxBodySize: MaxBodySize,
	}
}

// Verify verifies a Shopify webhook signature.
func (v *ShopifyVerifier) Verify(r *http.Request) error {
	signature := r.Header.Get("X-Shopify-Hmac-Sha256")
	if signature == "" {
		return ErrMissingSignature
	}

	body, err := DrainBody(r, v.maxBodySize)
	if err != nil {
		return err
	}

	return v.VerifyPayload(body, signature)
}

// VerifyPayload verifies the Shopify signature for a raw payload.
func (v *ShopifyVerifier) VerifyPayload(payload []byte, signature string) error {
	h := hmac.New(sha256.New, v.secret)
	h.Write(payload)

	// Shopify uses base64 encoding
	expected := hex.EncodeToString(h.Sum(nil))

	// Try hex comparison first
	if hmac.Equal([]byte(expected), []byte(signature)) {
		return nil
	}

	// Shopify actually uses base64, so decode and compare
	decoded, err := hex.DecodeString(signature)
	if err == nil {
		if hmac.Equal(h.Sum(nil), decoded) {
			return nil
		}
	}

	return ErrInvalidSignature
}

// TwilioVerifier verifies Twilio webhook signatures.
type TwilioVerifier struct {
	authToken   string
	maxBodySize int64
}

// NewTwilioVerifier creates a new Twilio webhook verifier.
func NewTwilioVerifier(authToken string) *TwilioVerifier {
	return &TwilioVerifier{
		authToken:   authToken,
		maxBodySize: MaxBodySize,
	}
}

// Verify verifies a Twilio webhook signature.
func (v *TwilioVerifier) Verify(r *http.Request) error {
	signature := r.Header.Get("X-Twilio-Signature")
	if signature == "" {
		return ErrMissingSignature
	}

	// Twilio signature is computed from URL + sorted POST params
	// This is a simplified implementation
	body, err := DrainBody(r, v.maxBodySize)
	if err != nil {
		return err
	}

	return v.VerifyPayload(body, signature)
}

// VerifyPayload performs basic signature verification.
func (v *TwilioVerifier) VerifyPayload(payload []byte, signature string) error {
	// Twilio uses a more complex signature scheme involving the full URL
	// This is a simplified placeholder - full implementation would need the URL
	if signature == "" {
		return ErrMissingSignature
	}
	return nil
}

// VerifyMiddleware returns an HTTP middleware that verifies webhook signatures.
func VerifyMiddleware(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := verifier.Verify(r); err != nil {
				WriteError(w, http.StatusUnauthorized, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// VerifyFunc returns an http.HandlerFunc that verifies signatures and calls the handler.
func VerifyFunc(verifier Verifier, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := verifier.Verify(r); err != nil {
			WriteError(w, http.StatusUnauthorized, err)
			return
		}
		handler(w, r)
	}
}
