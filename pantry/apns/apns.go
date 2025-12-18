// apns/apns.go
package apns

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/pkcs12"
	"golang.org/x/net/http2"
)

const (
	// Production APNs endpoint
	ProductionEndpoint = "https://api.push.apple.com"

	// Development/sandbox APNs endpoint
	DevelopmentEndpoint = "https://api.development.push.apple.com"

	// Default port for APNs
	defaultPort = 443

	// Token refresh interval (tokens are valid for 1 hour, refresh at 50 minutes)
	tokenRefreshInterval = 50 * time.Minute
)

// Client is an Apple Push Notification Service client.
type Client struct {
	endpoint   string
	httpClient *http.Client

	// Token-based authentication
	authKey   *ecdsa.PrivateKey
	keyID     string
	teamID    string
	token     string
	tokenTime time.Time
	tokenMu   sync.RWMutex

	// Certificate-based authentication
	certificate *tls.Certificate
}

// Config configures the APNs client.
type Config struct {
	// Endpoint is the APNs endpoint URL.
	// Use ProductionEndpoint or DevelopmentEndpoint.
	// Default: ProductionEndpoint
	Endpoint string

	// Token-based authentication (preferred)

	// AuthKeyFile is the path to the .p8 auth key file.
	AuthKeyFile string

	// AuthKeyBytes is the auth key data (alternative to AuthKeyFile).
	AuthKeyBytes []byte

	// KeyID is the key ID from Apple Developer.
	KeyID string

	// TeamID is the team ID from Apple Developer.
	TeamID string

	// Certificate-based authentication (legacy)

	// CertificateFile is the path to the .p12 certificate file.
	CertificateFile string

	// CertificateBytes is the certificate data (alternative to CertificateFile).
	CertificateBytes []byte

	// CertificatePassword is the password for the .p12 file.
	CertificatePassword string

	// HTTPClient is a custom HTTP client.
	// If nil, a default HTTP/2 client is created.
	HTTPClient *http.Client

	// Timeout for requests.
	// Default: 30 seconds
	Timeout time.Duration
}

// NewClient creates a new APNs client.
func NewClient(cfg Config) (*Client, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = ProductionEndpoint
	}

	client := &Client{
		endpoint: endpoint,
	}

	// Token-based authentication
	if cfg.AuthKeyFile != "" || len(cfg.AuthKeyBytes) > 0 {
		var keyData []byte
		var err error

		if len(cfg.AuthKeyBytes) > 0 {
			keyData = cfg.AuthKeyBytes
		} else {
			keyData, err = os.ReadFile(cfg.AuthKeyFile)
			if err != nil {
				return nil, fmt.Errorf("apns: failed to read auth key file: %w", err)
			}
		}

		key, err := parseAuthKey(keyData)
		if err != nil {
			return nil, err
		}

		if cfg.KeyID == "" {
			return nil, ErrKeyIDRequired
		}
		if cfg.TeamID == "" {
			return nil, ErrTeamIDRequired
		}

		client.authKey = key
		client.keyID = cfg.KeyID
		client.teamID = cfg.TeamID
	}

	// Certificate-based authentication
	if cfg.CertificateFile != "" || len(cfg.CertificateBytes) > 0 {
		var certData []byte
		var err error

		if len(cfg.CertificateBytes) > 0 {
			certData = cfg.CertificateBytes
		} else {
			certData, err = os.ReadFile(cfg.CertificateFile)
			if err != nil {
				return nil, fmt.Errorf("apns: failed to read certificate file: %w", err)
			}
		}

		cert, err := parseCertificate(certData, cfg.CertificatePassword)
		if err != nil {
			return nil, err
		}

		client.certificate = cert
	}

	// Require some form of authentication
	if client.authKey == nil && client.certificate == nil {
		return nil, ErrAuthRequired
	}

	// Set up HTTP client
	if cfg.HTTPClient != nil {
		client.httpClient = cfg.HTTPClient
	} else {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		var tlsConfig *tls.Config
		if client.certificate != nil {
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{*client.certificate},
			}
		}

		transport := &http2.Transport{
			TLSClientConfig: tlsConfig,
		}

		client.httpClient = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}

	return client, nil
}

// parseAuthKey parses a .p8 auth key file.
func parseAuthKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrInvalidAuthKey
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("apns: failed to parse auth key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, ErrInvalidAuthKey
	}

	return ecdsaKey, nil
}

// parseCertificate parses a .p12 certificate file.
func parseCertificate(data []byte, password string) (*tls.Certificate, error) {
	privateKey, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, fmt.Errorf("apns: failed to decode certificate: %w", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  privateKey,
		Leaf:        cert,
	}

	return tlsCert, nil
}

// Send sends a notification to a device.
func (c *Client) Send(ctx context.Context, notification *Notification) (*Response, error) {
	if notification == nil {
		return nil, ErrNotificationRequired
	}

	if notification.DeviceToken == "" {
		return nil, ErrDeviceTokenRequired
	}

	// Build payload
	payload, err := notification.MarshalPayload()
	if err != nil {
		return nil, fmt.Errorf("apns: failed to marshal payload: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/3/device/%s", c.endpoint, notification.DeviceToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("apns: failed to create request: %w", err)
	}

	// Set headers
	c.setHeaders(req, notification)

	// Set authorization
	if c.authKey != nil {
		token, err := c.getToken()
		if err != nil {
			return nil, fmt.Errorf("apns: failed to get auth token: %w", err)
		}
		req.Header.Set("authorization", "bearer "+token)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apns: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("apns: failed to read response: %w", err)
	}

	// Parse response
	response := &Response{
		StatusCode: resp.StatusCode,
		ApnsID:     resp.Header.Get("apns-id"),
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			response.Reason = errResp.Reason
			response.Timestamp = errResp.Timestamp
		}
		return response, parseError(resp.StatusCode, response.Reason)
	}

	return response, nil
}

// SendMultiple sends notifications to multiple devices.
func (c *Client) SendMultiple(ctx context.Context, notifications []*Notification) (*BatchResponse, error) {
	if len(notifications) == 0 {
		return nil, ErrNoNotifications
	}

	results := make([]SendResult, len(notifications))
	var successCount, failureCount int

	for i, notif := range notifications {
		resp, err := c.Send(ctx, notif)
		results[i] = SendResult{
			DeviceToken: notif.DeviceToken,
			Response:    resp,
			Error:       err,
		}

		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	return &BatchResponse{
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      results,
	}, nil
}

// setHeaders sets the APNs request headers.
func (c *Client) setHeaders(req *http.Request, n *Notification) {
	req.Header.Set("content-type", "application/json")

	if n.ApnsID != "" {
		req.Header.Set("apns-id", n.ApnsID)
	}

	if n.CollapseID != "" {
		req.Header.Set("apns-collapse-id", n.CollapseID)
	}

	if n.Expiration > 0 {
		req.Header.Set("apns-expiration", fmt.Sprintf("%d", n.Expiration))
	}

	if n.Priority > 0 {
		req.Header.Set("apns-priority", fmt.Sprintf("%d", n.Priority))
	}

	if n.Topic != "" {
		req.Header.Set("apns-topic", n.Topic)
	}

	if n.PushType != "" {
		req.Header.Set("apns-push-type", string(n.PushType))
	}
}

// getToken returns a valid JWT token, generating a new one if needed.
func (c *Client) getToken() (string, error) {
	c.tokenMu.RLock()
	if c.token != "" && time.Since(c.tokenTime) < tokenRefreshInterval {
		token := c.token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Since(c.tokenTime) < tokenRefreshInterval {
		return c.token, nil
	}

	token, err := c.generateToken()
	if err != nil {
		return "", err
	}

	c.token = token
	c.tokenTime = time.Now()

	return token, nil
}

// generateToken generates a new JWT token for APNs.
func (c *Client) generateToken() (string, error) {
	header := map[string]string{
		"alg": "ES256",
		"kid": c.keyID,
	}

	claims := map[string]any{
		"iss": c.teamID,
		"iat": time.Now().Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.New(rand.NewSource(time.Now().UnixNano())), c.authKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("apns: failed to sign token: %w", err)
	}

	// Convert to fixed-size signature
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):], sBytes)

	signatureB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + signatureB64, nil
}

// Response is the response from sending a notification.
type Response struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// ApnsID is the unique ID for the notification.
	ApnsID string

	// Reason is the error reason if the request failed.
	Reason string

	// Timestamp is when the device token became invalid (for Unregistered errors).
	Timestamp int64
}

// Success returns true if the notification was sent successfully.
func (r *Response) Success() bool {
	return r.StatusCode == http.StatusOK
}

// BatchResponse is the response from sending multiple notifications.
type BatchResponse struct {
	SuccessCount int
	FailureCount int
	Results      []SendResult
}

// SendResult is the result of sending a single notification.
type SendResult struct {
	DeviceToken string
	Response    *Response
	Error       error
}

// FailedTokens returns tokens that failed to send.
func (r *BatchResponse) FailedTokens() []string {
	var tokens []string
	for _, result := range r.Results {
		if result.Error != nil {
			tokens = append(tokens, result.DeviceToken)
		}
	}
	return tokens
}

// InvalidTokens returns tokens that are no longer valid.
func (r *BatchResponse) InvalidTokens() []string {
	var tokens []string
	for _, result := range r.Results {
		if result.Response != nil && (result.Response.Reason == "BadDeviceToken" ||
			result.Response.Reason == "Unregistered" ||
			result.Response.Reason == "DeviceTokenNotForTopic") {
			tokens = append(tokens, result.DeviceToken)
		}
	}
	return tokens
}

// errorResponse is the error response from APNs.
type errorResponse struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// parseError parses an APNs error response.
func parseError(statusCode int, reason string) error {
	switch reason {
	case "BadDeviceToken":
		return ErrBadDeviceToken
	case "Unregistered":
		return ErrUnregistered
	case "DeviceTokenNotForTopic":
		return ErrDeviceTokenNotForTopic
	case "BadCollapseId":
		return ErrBadCollapseID
	case "BadExpirationDate":
		return ErrBadExpirationDate
	case "BadMessageId":
		return ErrBadMessageID
	case "BadPriority":
		return ErrBadPriority
	case "BadTopic":
		return ErrBadTopic
	case "DuplicateHeaders":
		return ErrDuplicateHeaders
	case "IdleTimeout":
		return ErrIdleTimeout
	case "InvalidPushType":
		return ErrInvalidPushType
	case "MissingDeviceToken":
		return ErrMissingDeviceToken
	case "MissingTopic":
		return ErrMissingTopic
	case "PayloadEmpty":
		return ErrPayloadEmpty
	case "TopicDisallowed":
		return ErrTopicDisallowed
	case "BadCertificate":
		return ErrBadCertificate
	case "BadCertificateEnvironment":
		return ErrBadCertificateEnvironment
	case "ExpiredProviderToken":
		return ErrExpiredProviderToken
	case "Forbidden":
		return ErrForbidden
	case "InvalidProviderToken":
		return ErrInvalidProviderToken
	case "MissingProviderToken":
		return ErrMissingProviderToken
	case "BadPath":
		return ErrBadPath
	case "MethodNotAllowed":
		return ErrMethodNotAllowed
	case "ExpiredToken":
		return ErrExpiredToken
	case "TooManyProviderTokenUpdates":
		return ErrTooManyProviderTokenUpdates
	case "TooManyRequests":
		return ErrTooManyRequests
	case "InternalServerError":
		return ErrInternalServerError
	case "ServiceUnavailable":
		return ErrServiceUnavailable
	case "Shutdown":
		return ErrShutdown
	default:
		return &Error{
			StatusCode: statusCode,
			Reason:     reason,
		}
	}
}

// Global client for convenience
var (
	defaultClient   *Client
	defaultClientMu sync.RWMutex
)

// SetDefaultClient sets the default APNs client.
func SetDefaultClient(client *Client) {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = client
}

// DefaultClient returns the default APNs client.
func DefaultClient() *Client {
	defaultClientMu.RLock()
	defer defaultClientMu.RUnlock()
	return defaultClient
}

// Send sends a notification using the default client.
func Send(ctx context.Context, notification *Notification) (*Response, error) {
	client := DefaultClient()
	if client == nil {
		return nil, ErrNoDefaultClient
	}
	return client.Send(ctx, notification)
}

// Development returns a new client configured for the development environment.
func Development(cfg Config) (*Client, error) {
	cfg.Endpoint = DevelopmentEndpoint
	return NewClient(cfg)
}

// Production returns a new client configured for the production environment.
func Production(cfg Config) (*Client, error) {
	cfg.Endpoint = ProductionEndpoint
	return NewClient(cfg)
}
