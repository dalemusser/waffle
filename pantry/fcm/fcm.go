// fcm/fcm.go
package fcm

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// FCM HTTP v1 API endpoint
	fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

	// Token endpoint for OAuth2
	tokenEndpoint = "https://oauth2.googleapis.com/token"

	// Default timeout for API calls
	defaultTimeout = 30 * time.Second
)

// Client is a Firebase Cloud Messaging client.
type Client struct {
	projectID    string
	credentials  *ServiceAccountCredentials
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
	tokenMu      sync.RWMutex
}

// ServiceAccountCredentials holds Firebase service account credentials.
type ServiceAccountCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`

	// Parsed private key
	privateKey *rsa.PrivateKey
}

// Config configures the FCM client.
type Config struct {
	// ProjectID is the Firebase project ID.
	// If empty, extracted from credentials.
	ProjectID string

	// Credentials is the service account credentials JSON.
	Credentials []byte

	// CredentialsFile is the path to the service account JSON file.
	// Used if Credentials is empty.
	CredentialsFile string

	// HTTPClient is the HTTP client to use.
	// If nil, a default client with timeout is used.
	HTTPClient *http.Client

	// Timeout is the timeout for API calls.
	// Default: 30 seconds
	Timeout time.Duration
}

// NewClient creates a new FCM client.
func NewClient(cfg Config) (*Client, error) {
	// Load credentials
	var credJSON []byte
	var err error

	if len(cfg.Credentials) > 0 {
		credJSON = cfg.Credentials
	} else if cfg.CredentialsFile != "" {
		credJSON, err = readFile(cfg.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("fcm: failed to read credentials file: %w", err)
		}
	} else {
		return nil, ErrCredentialsRequired
	}

	// Parse credentials
	var creds ServiceAccountCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, fmt.Errorf("fcm: failed to parse credentials: %w", err)
	}

	// Parse private key
	block, _ := pem.Decode([]byte(creds.PrivateKey))
	if block == nil {
		return nil, ErrInvalidPrivateKey
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to parse private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrInvalidPrivateKey
	}
	creds.privateKey = rsaKey

	// Determine project ID
	projectID := cfg.ProjectID
	if projectID == "" {
		projectID = creds.ProjectID
	}
	if projectID == "" {
		return nil, ErrProjectIDRequired
	}

	// Set up HTTP client
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		projectID:   projectID,
		credentials: &creds,
		httpClient:  httpClient,
	}, nil
}

// Send sends a message to FCM.
func (c *Client) Send(ctx context.Context, msg *Message) (*SendResponse, error) {
	return c.send(ctx, msg, false)
}

// SendDryRun validates a message without actually sending it.
func (c *Client) SendDryRun(ctx context.Context, msg *Message) (*SendResponse, error) {
	return c.send(ctx, msg, true)
}

func (c *Client) send(ctx context.Context, msg *Message, dryRun bool) (*SendResponse, error) {
	if msg == nil {
		return nil, ErrMessageRequired
	}

	// Validate message has a target
	if msg.Token == "" && msg.Topic == "" && msg.Condition == "" {
		return nil, ErrTargetRequired
	}

	// Get access token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to get access token: %w", err)
	}

	// Build request body
	reqBody := sendRequest{
		ValidateOnly: dryRun,
		Message:      msg,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf(fcmEndpoint, c.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.StatusCode, respBody)
	}

	// Parse success response
	var sendResp sendResponse
	if err := json.Unmarshal(respBody, &sendResp); err != nil {
		return nil, fmt.Errorf("fcm: failed to parse response: %w", err)
	}

	return &SendResponse{
		MessageID: sendResp.Name,
	}, nil
}

// SendMulticast sends a message to multiple tokens.
// Returns results for each token.
func (c *Client) SendMulticast(ctx context.Context, msg *MulticastMessage) (*MulticastResponse, error) {
	if msg == nil || len(msg.Tokens) == 0 {
		return nil, ErrTokensRequired
	}

	results := make([]SendResult, len(msg.Tokens))
	var successCount, failureCount int

	// Send to each token
	for i, token := range msg.Tokens {
		singleMsg := &Message{
			Token:        token,
			Notification: msg.Notification,
			Data:         msg.Data,
			Android:      msg.Android,
			Webpush:      msg.Webpush,
			APNS:         msg.APNS,
			FCMOptions:   msg.FCMOptions,
		}

		resp, err := c.Send(ctx, singleMsg)
		if err != nil {
			results[i] = SendResult{
				Error: err,
			}
			failureCount++
		} else {
			results[i] = SendResult{
				MessageID: resp.MessageID,
				Success:   true,
			}
			successCount++
		}
	}

	return &MulticastResponse{
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      results,
	}, nil
}

// SendAll sends multiple messages in batch.
func (c *Client) SendAll(ctx context.Context, messages []*Message) (*BatchResponse, error) {
	if len(messages) == 0 {
		return nil, ErrMessagesRequired
	}

	results := make([]SendResult, len(messages))
	var successCount, failureCount int

	for i, msg := range messages {
		resp, err := c.Send(ctx, msg)
		if err != nil {
			results[i] = SendResult{
				Error: err,
			}
			failureCount++
		} else {
			results[i] = SendResult{
				MessageID: resp.MessageID,
				Success:   true,
			}
			successCount++
		}
	}

	return &BatchResponse{
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      results,
	}, nil
}

// SubscribeToTopic subscribes tokens to a topic.
func (c *Client) SubscribeToTopic(ctx context.Context, tokens []string, topic string) (*TopicResponse, error) {
	return c.manageTopicSubscription(ctx, tokens, topic, "batchAdd")
}

// UnsubscribeFromTopic unsubscribes tokens from a topic.
func (c *Client) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) (*TopicResponse, error) {
	return c.manageTopicSubscription(ctx, tokens, topic, "batchRemove")
}

func (c *Client) manageTopicSubscription(ctx context.Context, tokens []string, topic string, action string) (*TopicResponse, error) {
	if len(tokens) == 0 {
		return nil, ErrTokensRequired
	}
	if topic == "" {
		return nil, ErrTopicRequired
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://iid.googleapis.com/iid/v1:%s", action)

	reqBody := topicRequest{
		To:                  "/topics/" + topic,
		RegistrationTokens: tokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token_auth", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.StatusCode, respBody)
	}

	var topicResp topicResponse
	if err := json.Unmarshal(respBody, &topicResp); err != nil {
		return nil, fmt.Errorf("fcm: failed to parse response: %w", err)
	}

	result := &TopicResponse{
		SuccessCount: 0,
		FailureCount: 0,
		Errors:       make([]TopicError, 0),
	}

	for i, r := range topicResp.Results {
		if r.Error != "" {
			result.FailureCount++
			result.Errors = append(result.Errors, TopicError{
				Index: i,
				Error: r.Error,
			})
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// getAccessToken returns a valid access token, refreshing if necessary.
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-time.Minute)) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-time.Minute)) {
		return c.accessToken, nil
	}

	// Create JWT
	jwt, err := c.createJWT()
	if err != nil {
		return "", err
	}

	// Exchange JWT for access token
	token, expiry, err := c.exchangeToken(ctx, jwt)
	if err != nil {
		return "", err
	}

	c.accessToken = token
	c.tokenExpiry = expiry

	return token, nil
}

// createJWT creates a signed JWT for authentication.
func (c *Client) createJWT() (string, error) {
	now := time.Now()
	exp := now.Add(time.Hour)

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}

	claims := map[string]any{
		"iss":   c.credentials.ClientEmail,
		"sub":   c.credentials.ClientEmail,
		"aud":   tokenEndpoint,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	signature, err := signRS256([]byte(signingInput), c.credentials.privateKey)
	if err != nil {
		return "", err
	}

	signatureB64 := base64URLEncode(signature)

	return signingInput + "." + signatureB64, nil
}

// exchangeToken exchanges a JWT for an access token.
func (c *Client) exchangeToken(ctx context.Context, jwt string) (string, time.Time, error) {
	data := "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer&assertion=" + jwt

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, bytes.NewBufferString(data))
	if err != nil {
		return "", time.Time{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return tokenResp.AccessToken, expiry, nil
}

// SendResponse is the response from sending a message.
type SendResponse struct {
	// MessageID is the unique identifier for the sent message.
	MessageID string
}

// MulticastResponse is the response from sending a multicast message.
type MulticastResponse struct {
	SuccessCount int
	FailureCount int
	Results      []SendResult
}

// BatchResponse is the response from sending multiple messages.
type BatchResponse struct {
	SuccessCount int
	FailureCount int
	Results      []SendResult
}

// SendResult is the result of sending a single message.
type SendResult struct {
	MessageID string
	Success   bool
	Error     error
}

// TopicResponse is the response from topic subscription operations.
type TopicResponse struct {
	SuccessCount int
	FailureCount int
	Errors       []TopicError
}

// TopicError represents an error for a specific token in topic operations.
type TopicError struct {
	Index int
	Error string
}

// Request/response types for API calls.
type sendRequest struct {
	ValidateOnly bool     `json:"validate_only,omitempty"`
	Message      *Message `json:"message"`
}

type sendResponse struct {
	Name string `json:"name"`
}

type topicRequest struct {
	To                  string   `json:"to"`
	RegistrationTokens []string `json:"registration_tokens"`
}

type topicResponse struct {
	Results []struct {
		Error string `json:"error,omitempty"`
	} `json:"results"`
}

// parseError parses an FCM error response.
func parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
			Details []struct {
				Type      string `json:"@type"`
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return &Error{
			Code:    statusCode,
			Message: string(body),
		}
	}

	fcmErr := &Error{
		Code:    statusCode,
		Message: errResp.Error.Message,
		Status:  errResp.Error.Status,
	}

	// Extract FCM-specific error code
	for _, detail := range errResp.Error.Details {
		if detail.ErrorCode != "" {
			fcmErr.FCMCode = detail.ErrorCode
			break
		}
	}

	return fcmErr
}

// Error represents an FCM API error.
type Error struct {
	Code    int
	Message string
	Status  string
	FCMCode string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.FCMCode != "" {
		return fmt.Sprintf("fcm: %s (code: %s, status: %d)", e.Message, e.FCMCode, e.Code)
	}
	return fmt.Sprintf("fcm: %s (status: %d)", e.Message, e.Code)
}

// Is checks if this error matches the target.
func (e *Error) Is(target error) bool {
	switch e.FCMCode {
	case "UNREGISTERED":
		return errors.Is(target, ErrTokenNotRegistered)
	case "INVALID_ARGUMENT":
		return errors.Is(target, ErrInvalidToken)
	case "QUOTA_EXCEEDED":
		return errors.Is(target, ErrQuotaExceeded)
	case "UNAVAILABLE":
		return errors.Is(target, ErrUnavailable)
	case "INTERNAL":
		return errors.Is(target, ErrServerError)
	case "SENDER_ID_MISMATCH":
		return errors.Is(target, ErrSenderMismatch)
	}
	return false
}

// IsRetryable returns true if the error is retryable.
func (e *Error) IsRetryable() bool {
	switch e.FCMCode {
	case "UNAVAILABLE", "INTERNAL":
		return true
	}
	return e.Code >= 500
}

// readFile reads a file from disk.
func readFile(path string) ([]byte, error) {
	return readFileImpl(path)
}

var readFileImpl = func(path string) ([]byte, error) {
	// Default implementation - will be overridden by init in a separate file
	return nil, errors.New("use Credentials instead of CredentialsFile")
}
