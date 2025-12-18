// notify/webpush.go
package notify

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

const (
	// Maximum payload size for Web Push (RFC 8291)
	maxPayloadSize = 4096

	// Default TTL for messages (24 hours)
	defaultTTL = 86400

	// Default token expiry for VAPID
	defaultTokenExpiry = 12 * time.Hour
)

// Client is a Web Push notification client.
type Client struct {
	vapidKeys  *VAPIDKeys
	httpClient *http.Client
	mu         sync.RWMutex
}

// Config configures the Web Push client.
type Config struct {
	// VAPIDKeys are the VAPID keys for authentication.
	VAPIDKeys *VAPIDKeys

	// HTTPClient is the HTTP client to use.
	// If nil, a default client with timeout is used.
	HTTPClient *http.Client

	// Timeout is the timeout for sending notifications.
	// Default: 30 seconds
	Timeout time.Duration
}

// NewClient creates a new Web Push client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.VAPIDKeys == nil {
		return nil, ErrVAPIDKeysRequired
	}

	if cfg.VAPIDKeys.Subject == "" {
		return nil, ErrSubjectRequired
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		vapidKeys:  cfg.VAPIDKeys,
		httpClient: httpClient,
	}, nil
}

// Message represents a Web Push notification message.
type Message struct {
	// Payload is the notification payload (typically JSON).
	Payload []byte

	// TTL is the time-to-live in seconds.
	// Default: 86400 (24 hours)
	TTL int

	// Urgency is the message urgency.
	// One of: "very-low", "low", "normal", "high"
	Urgency Urgency

	// Topic is an optional topic for message replacement.
	Topic string
}

// Urgency represents message urgency levels.
type Urgency string

const (
	UrgencyVeryLow Urgency = "very-low"
	UrgencyLow     Urgency = "low"
	UrgencyNormal  Urgency = "normal"
	UrgencyHigh    Urgency = "high"
)

// Notification is a standard notification payload.
type Notification struct {
	Title   string   `json:"title"`
	Body    string   `json:"body,omitempty"`
	Icon    string   `json:"icon,omitempty"`
	Badge   string   `json:"badge,omitempty"`
	Image   string   `json:"image,omitempty"`
	Tag     string   `json:"tag,omitempty"`
	Data    any      `json:"data,omitempty"`
	Actions []Action `json:"actions,omitempty"`

	// Behavior options
	Renotify           bool   `json:"renotify,omitempty"`
	RequireInteraction bool   `json:"requireInteraction,omitempty"`
	Silent             bool   `json:"silent,omitempty"`
	Timestamp          int64  `json:"timestamp,omitempty"`
	Dir                string `json:"dir,omitempty"`  // auto, ltr, rtl
	Lang               string `json:"lang,omitempty"` // BCP 47 language tag
	Vibrate            []int  `json:"vibrate,omitempty"`
}

// Action represents a notification action button.
type Action struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// SendResponse contains the result of sending a notification.
type SendResponse struct {
	// StatusCode is the HTTP status code from the push service.
	StatusCode int

	// Success indicates if the notification was accepted.
	Success bool

	// MessageID is the message ID if available.
	MessageID string
}

// Send sends a notification to a subscription.
func (c *Client) Send(ctx context.Context, sub *Subscription, msg *Message) (*SendResponse, error) {
	if sub == nil {
		return nil, ErrSubscriptionRequired
	}

	if err := sub.Validate(); err != nil {
		return nil, err
	}

	if msg == nil {
		msg = &Message{}
	}

	// Encrypt payload
	var body []byte
	var err error

	if len(msg.Payload) > 0 {
		if len(msg.Payload) > maxPayloadSize {
			return nil, ErrPayloadTooLarge
		}
		body, err = c.encrypt(sub, msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("notify: failed to encrypt payload: %w", err)
		}
	}

	// Parse endpoint URL for audience
	endpointURL, err := url.Parse(sub.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("notify: invalid endpoint URL: %w", err)
	}
	audience := endpointURL.Scheme + "://" + endpointURL.Host

	// Create VAPID authorization header
	authHeader, err := c.vapidKeys.authorizationHeader(audience, defaultTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to create VAPID header: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("notify: failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")

	ttl := msg.TTL
	if ttl == 0 {
		ttl = defaultTTL
	}
	req.Header.Set("TTL", strconv.Itoa(ttl))

	if msg.Urgency != "" {
		req.Header.Set("Urgency", string(msg.Urgency))
	}

	if msg.Topic != "" {
		req.Header.Set("Topic", msg.Topic)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	respBody, _ := io.ReadAll(resp.Body)

	result := &SendResponse{
		StatusCode: resp.StatusCode,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		MessageID:  resp.Header.Get("Location"),
	}

	if !result.Success {
		return result, parseHTTPError(resp.StatusCode, respBody)
	}

	return result, nil
}

// SendMultiple sends a notification to multiple subscriptions.
func (c *Client) SendMultiple(ctx context.Context, subs []*Subscription, msg *Message) (*BatchResult, error) {
	if len(subs) == 0 {
		return nil, ErrNoSubscriptions
	}

	results := make([]SubscriptionResult, len(subs))
	var successCount, failureCount int

	for i, sub := range subs {
		resp, err := c.Send(ctx, sub, msg)
		results[i] = SubscriptionResult{
			Subscription: sub,
			Response:     resp,
			Error:        err,
		}

		if err == nil && resp.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      results,
	}, nil
}

// BatchResult contains results from sending to multiple subscriptions.
type BatchResult struct {
	SuccessCount int
	FailureCount int
	Results      []SubscriptionResult
}

// SubscriptionResult contains the result for a single subscription.
type SubscriptionResult struct {
	Subscription *Subscription
	Response     *SendResponse
	Error        error
}

// FailedSubscriptions returns subscriptions that failed.
func (r *BatchResult) FailedSubscriptions() []*Subscription {
	var failed []*Subscription
	for _, result := range r.Results {
		if result.Error != nil || (result.Response != nil && !result.Response.Success) {
			failed = append(failed, result.Subscription)
		}
	}
	return failed
}

// ExpiredSubscriptions returns subscriptions that are no longer valid.
func (r *BatchResult) ExpiredSubscriptions() []*Subscription {
	var expired []*Subscription
	for _, result := range r.Results {
		if result.Response != nil && (result.Response.StatusCode == 404 || result.Response.StatusCode == 410) {
			expired = append(expired, result.Subscription)
		}
	}
	return expired
}

// encrypt encrypts the payload using the subscription keys.
// Implements RFC 8291 (Message Encryption for Web Push)
func (c *Client) encrypt(sub *Subscription, payload []byte) ([]byte, error) {
	// Decode subscription keys
	p256dh, err := base64.RawURLEncoding.DecodeString(sub.Keys.P256dh)
	if err != nil {
		return nil, fmt.Errorf("invalid p256dh key: %w", err)
	}

	auth, err := base64.RawURLEncoding.DecodeString(sub.Keys.Auth)
	if err != nil {
		return nil, fmt.Errorf("invalid auth key: %w", err)
	}

	// Generate ephemeral key pair
	curve := ecdh.P256()
	localPrivate, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	localPublic := localPrivate.PublicKey()

	// Parse subscriber's public key
	subscriberPublic, err := curve.NewPublicKey(p256dh)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriber public key: %w", err)
	}

	// ECDH shared secret
	sharedSecret, err := localPrivate.ECDH(subscriberPublic)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive keys using HKDF
	// IKM = HKDF-Extract(auth, ecdh_secret)
	// PRK = HKDF-Extract(salt, IKM)

	// First HKDF for IKM
	ikmInfo := buildInfo("WebPush: info", subscriberPublic.Bytes(), localPublic.Bytes())
	ikm := hkdfExtract(auth, sharedSecret, ikmInfo, 32)

	// Derive content encryption key
	cekInfo := []byte("Content-Encoding: aes128gcm\x00")
	cek := hkdfExtract(salt, ikm, cekInfo, 16)

	// Derive nonce
	nonceInfo := []byte("Content-Encoding: nonce\x00")
	nonce := hkdfExtract(salt, ikm, nonceInfo, 12)

	// Pad payload (add record size padding)
	// For simplicity, we use a single record with minimal padding
	padded := append(payload, 0x02) // Delimiter followed by padding

	// Encrypt with AES-128-GCM
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, padded, nil)

	// Build aes128gcm content encoding header
	// salt (16) + rs (4) + idlen (1) + keyid (65)
	localPublicBytes := localPublic.Bytes()

	recordSize := uint32(len(ciphertext) + 16 + 4 + 1 + len(localPublicBytes))

	header := make([]byte, 0, 16+4+1+len(localPublicBytes))
	header = append(header, salt...)
	header = binary.BigEndian.AppendUint32(header, recordSize)
	header = append(header, byte(len(localPublicBytes)))
	header = append(header, localPublicBytes...)

	return append(header, ciphertext...), nil
}

// buildInfo builds the info parameter for HKDF.
func buildInfo(prefix string, subscriberKey, localKey []byte) []byte {
	info := make([]byte, 0, len(prefix)+1+5+1+2+len(subscriberKey)+2+len(localKey))
	info = append(info, prefix...)
	info = append(info, 0x00)

	// Subscriber public key length (2 bytes) + key
	info = append(info, byte(len(subscriberKey)>>8), byte(len(subscriberKey)))
	info = append(info, subscriberKey...)

	// Local public key length (2 bytes) + key
	info = append(info, byte(len(localKey)>>8), byte(len(localKey)))
	info = append(info, localKey...)

	return info
}

// hkdfExtract performs HKDF-Extract and Expand.
func hkdfExtract(salt, secret, info []byte, length int) []byte {
	reader := hkdf.New(sha256.New, secret, salt, info)
	key := make([]byte, length)
	if _, err := io.ReadFull(reader, key); err != nil {
		panic("hkdf failed: " + err.Error())
	}
	return key
}

// parseHTTPError parses an HTTP error response.
func parseHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case 400:
		return &HTTPError{StatusCode: statusCode, Message: "bad request", Body: string(body)}
	case 401:
		return &HTTPError{StatusCode: statusCode, Message: "unauthorized (invalid VAPID)", Body: string(body)}
	case 403:
		return &HTTPError{StatusCode: statusCode, Message: "forbidden", Body: string(body)}
	case 404:
		return ErrSubscriptionExpired
	case 410:
		return ErrSubscriptionExpired
	case 413:
		return ErrPayloadTooLarge
	case 429:
		return ErrRateLimited
	default:
		if statusCode >= 500 {
			return &HTTPError{StatusCode: statusCode, Message: "server error", Body: string(body)}
		}
		return &HTTPError{StatusCode: statusCode, Message: "unknown error", Body: string(body)}
	}
}

// Global client for convenience
var (
	defaultClient   *Client
	defaultClientMu sync.RWMutex
)

// SetDefaultClient sets the default Web Push client.
func SetDefaultClient(client *Client) {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = client
}

// DefaultClient returns the default Web Push client.
func DefaultClient() *Client {
	defaultClientMu.RLock()
	defer defaultClientMu.RUnlock()
	return defaultClient
}

// Send sends a notification using the default client.
func Send(ctx context.Context, sub *Subscription, msg *Message) (*SendResponse, error) {
	client := DefaultClient()
	if client == nil {
		return nil, ErrNoDefaultClient
	}
	return client.Send(ctx, sub, msg)
}

// SendMultiple sends to multiple subscriptions using the default client.
func SendMultiple(ctx context.Context, subs []*Subscription, msg *Message) (*BatchResult, error) {
	client := DefaultClient()
	if client == nil {
		return nil, ErrNoDefaultClient
	}
	return client.SendMultiple(ctx, subs, msg)
}
