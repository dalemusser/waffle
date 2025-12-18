package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SenderConfig configures the webhook sender.
type SenderConfig struct {
	// SigningSecret is the secret used to sign outgoing webhooks.
	// If empty, webhooks are sent without signatures.
	SigningSecret string

	// SignatureHeader is the header name for the signature.
	// Default: "X-Webhook-Signature"
	SignatureHeader string

	// TimestampHeader is the header name for the timestamp.
	// Default: "X-Webhook-Timestamp"
	TimestampHeader string

	// HTTPClient is the HTTP client to use.
	// If nil, a default client with reasonable timeouts is used.
	HTTPClient *http.Client

	// Timeout is the timeout for each delivery attempt.
	// Default: 30 seconds
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries int

	// RetryBackoff is the initial backoff duration between retries.
	// Default: 1 second
	RetryBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	// Default: 1 minute
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	// Default: 2.0
	BackoffMultiplier float64

	// Headers are additional headers to include in all requests.
	Headers map[string]string

	// UserAgent is the User-Agent header value.
	// Default: "Waffle-Webhook/1.0"
	UserAgent string

	// OnDelivery is called after each delivery attempt.
	// Can be used for logging or metrics.
	OnDelivery func(url string, result DeliveryResult)
}

// Sender sends outgoing webhooks with automatic retries.
type Sender struct {
	signingSecret     string
	signatureHeader   string
	timestampHeader   string
	httpClient        *http.Client
	timeout           time.Duration
	maxRetries        int
	retryBackoff      time.Duration
	maxBackoff        time.Duration
	backoffMultiplier float64
	headers           map[string]string
	userAgent         string
	onDelivery        func(url string, result DeliveryResult)
}

// NewSender creates a new webhook sender.
func NewSender(cfg SenderConfig) *Sender {
	if cfg.SignatureHeader == "" {
		cfg.SignatureHeader = "X-Webhook-Signature"
	}
	if cfg.TimestampHeader == "" {
		cfg.TimestampHeader = "X-Webhook-Timestamp"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 1 * time.Minute
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2.0
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "Waffle-Webhook/1.0"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Sender{
		signingSecret:     cfg.SigningSecret,
		signatureHeader:   cfg.SignatureHeader,
		timestampHeader:   cfg.TimestampHeader,
		httpClient:        httpClient,
		timeout:           cfg.Timeout,
		maxRetries:        cfg.MaxRetries,
		retryBackoff:      cfg.RetryBackoff,
		maxBackoff:        cfg.MaxBackoff,
		backoffMultiplier: cfg.BackoffMultiplier,
		headers:           cfg.Headers,
		userAgent:         cfg.UserAgent,
		onDelivery:        cfg.OnDelivery,
	}
}

// Send sends a webhook event to the specified URL.
func (s *Sender) Send(ctx context.Context, url string, event Event) error {
	results, err := s.SendWithResults(ctx, url, event)
	if err != nil {
		return err
	}

	// Return the last error if all attempts failed
	if len(results) > 0 && !results[len(results)-1].Success {
		return results[len(results)-1].Error
	}

	return nil
}

// SendWithResults sends a webhook event and returns all delivery attempt results.
func (s *Sender) SendWithResults(ctx context.Context, url string, event Event) ([]DeliveryResult, error) {
	// Build payload
	payload := Payload{
		Event:      s.eventToRaw(event),
		DeliveryID: generateDeliveryID(),
		Attempt:    1,
	}

	return s.deliver(ctx, url, payload)
}

// SendRaw sends a raw payload to the specified URL.
func (s *Sender) SendRaw(ctx context.Context, url string, payload []byte) error {
	results, err := s.deliverRaw(ctx, url, payload)
	if err != nil {
		return err
	}

	if len(results) > 0 && !results[len(results)-1].Success {
		return results[len(results)-1].Error
	}

	return nil
}

// eventToRaw converts an Event to a RawEvent.
func (s *Sender) eventToRaw(event Event) RawEvent {
	data, _ := json.Marshal(event.Data)
	return RawEvent{
		ID:        event.ID,
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      data,
		Metadata:  event.Metadata,
	}
}

// deliver sends the payload with retries.
func (s *Sender) deliver(ctx context.Context, url string, payload Payload) ([]DeliveryResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("webhook: failed to marshal payload: %w", err)
	}

	return s.deliverRaw(ctx, url, data)
}

// deliverRaw sends raw data with retries.
func (s *Sender) deliverRaw(ctx context.Context, url string, data []byte) ([]DeliveryResult, error) {
	var results []DeliveryResult
	backoff := s.retryBackoff

	for attempt := 1; attempt <= s.maxRetries+1; attempt++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := s.attemptDelivery(ctx, url, data, attempt)
		results = append(results, result)

		// Call delivery callback
		if s.onDelivery != nil {
			s.onDelivery(url, result)
		}

		if result.Success {
			return results, nil
		}

		// Check if we should retry
		if attempt > s.maxRetries {
			break
		}

		// Only retry on retryable errors
		if !s.isRetryable(result) {
			break
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(backoff):
		}

		// Exponential backoff
		backoff = time.Duration(float64(backoff) * s.backoffMultiplier)
		if backoff > s.maxBackoff {
			backoff = s.maxBackoff
		}
	}

	return results, ErrDeliveryFailed
}

// attemptDelivery makes a single delivery attempt.
func (s *Sender) attemptDelivery(ctx context.Context, url string, data []byte, attempt int) DeliveryResult {
	start := time.Now()

	result := DeliveryResult{
		Attempt:   attempt,
		Timestamp: start,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		result.Error = fmt.Errorf("webhook: failed to create request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	// Sign the payload
	if s.signingSecret != "" {
		timestamp := time.Now().Unix()
		signature := SignPayloadWithTimestamp(timestamp, data, s.signingSecret)
		req.Header.Set(s.signatureHeader, signature)
		req.Header.Set(s.timestampHeader, fmt.Sprintf("%d", timestamp))
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("webhook: request failed: %w", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)

	// Read response body (truncated)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	result.ResponseBody = truncateString(string(body), 500)

	// Check status
	if isSuccessStatus(resp.StatusCode) {
		result.Success = true
	} else {
		result.Error = fmt.Errorf("webhook: received status %d", resp.StatusCode)
	}

	return result
}

// isRetryable determines if a delivery should be retried.
func (s *Sender) isRetryable(result DeliveryResult) bool {
	// Network errors are retryable
	if result.StatusCode == 0 {
		return true
	}

	return isRetryableStatus(result.StatusCode)
}

// BatchSender sends webhooks to multiple endpoints.
type BatchSender struct {
	sender      *Sender
	concurrency int
}

// NewBatchSender creates a new batch webhook sender.
func NewBatchSender(sender *Sender, concurrency int) *BatchSender {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &BatchSender{
		sender:      sender,
		concurrency: concurrency,
	}
}

// BatchResult contains results for a batch send operation.
type BatchResult struct {
	URL     string
	Results []DeliveryResult
	Error   error
}

// SendToAll sends an event to multiple URLs concurrently.
func (b *BatchSender) SendToAll(ctx context.Context, urls []string, event Event) []BatchResult {
	results := make([]BatchResult, len(urls))
	var wg sync.WaitGroup

	// Semaphore for concurrency control
	sem := make(chan struct{}, b.concurrency)

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = BatchResult{
					URL:   targetURL,
					Error: ctx.Err(),
				}
				return
			}

			deliveryResults, err := b.sender.SendWithResults(ctx, targetURL, event)
			results[idx] = BatchResult{
				URL:     targetURL,
				Results: deliveryResults,
				Error:   err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// Subscription represents a webhook subscription.
type Subscription struct {
	// ID is a unique identifier for this subscription.
	ID string `json:"id"`

	// URL is the endpoint to deliver webhooks to.
	URL string `json:"url"`

	// Events is a list of event types to subscribe to.
	// Use "*" to subscribe to all events.
	Events []string `json:"events"`

	// Secret is the signing secret for this subscription.
	Secret string `json:"secret,omitempty"`

	// Active indicates if the subscription is active.
	Active bool `json:"active"`

	// Metadata contains additional subscription metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// CreatedAt is when the subscription was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the subscription was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// Matches returns true if the subscription matches the given event type.
func (s *Subscription) Matches(eventType string) bool {
	if !s.Active {
		return false
	}

	for _, pattern := range s.Events {
		if pattern == "*" || pattern == eventType {
			return true
		}
		// Simple wildcard matching: "order.*" matches "order.created"
		if len(pattern) > 1 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
				return true
			}
		}
	}

	return false
}

// Dispatcher manages webhook subscriptions and dispatches events.
type Dispatcher struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	sender        *Sender
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(sender *Sender) *Dispatcher {
	return &Dispatcher{
		subscriptions: make(map[string]*Subscription),
		sender:        sender,
	}
}

// Subscribe adds a new subscription.
func (d *Dispatcher) Subscribe(sub *Subscription) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if sub.ID == "" {
		sub.ID = generateEventID()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	sub.UpdatedAt = time.Now()

	d.subscriptions[sub.ID] = sub
}

// Unsubscribe removes a subscription.
func (d *Dispatcher) Unsubscribe(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.subscriptions[id]; exists {
		delete(d.subscriptions, id)
		return true
	}
	return false
}

// GetSubscription returns a subscription by ID.
func (d *Dispatcher) GetSubscription(id string) (*Subscription, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sub, exists := d.subscriptions[id]
	return sub, exists
}

// ListSubscriptions returns all subscriptions.
func (d *Dispatcher) ListSubscriptions() []*Subscription {
	d.mu.RLock()
	defer d.mu.RUnlock()

	subs := make([]*Subscription, 0, len(d.subscriptions))
	for _, sub := range d.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// Dispatch sends an event to all matching subscriptions.
func (d *Dispatcher) Dispatch(ctx context.Context, event Event) []BatchResult {
	d.mu.RLock()
	var matching []*Subscription
	for _, sub := range d.subscriptions {
		if sub.Matches(event.Type) {
			matching = append(matching, sub)
		}
	}
	d.mu.RUnlock()

	if len(matching) == 0 {
		return nil
	}

	results := make([]BatchResult, len(matching))
	var wg sync.WaitGroup

	for i, sub := range matching {
		wg.Add(1)
		go func(idx int, subscription *Subscription) {
			defer wg.Done()

			// Create a sender with the subscription's secret if provided
			sender := d.sender
			if subscription.Secret != "" {
				sender = NewSender(SenderConfig{
					SigningSecret:     subscription.Secret,
					HTTPClient:        d.sender.httpClient,
					Timeout:           d.sender.timeout,
					MaxRetries:        d.sender.maxRetries,
					RetryBackoff:      d.sender.retryBackoff,
					MaxBackoff:        d.sender.maxBackoff,
					BackoffMultiplier: d.sender.backoffMultiplier,
					Headers:           d.sender.headers,
					UserAgent:         d.sender.userAgent,
					OnDelivery:        d.sender.onDelivery,
				})
			}

			deliveryResults, err := sender.SendWithResults(ctx, subscription.URL, event)
			results[idx] = BatchResult{
				URL:     subscription.URL,
				Results: deliveryResults,
				Error:   err,
			}
		}(i, sub)
	}

	wg.Wait()
	return results
}

// DispatchAsync sends an event to all matching subscriptions asynchronously.
// Results are sent to the provided channel.
func (d *Dispatcher) DispatchAsync(ctx context.Context, event Event, results chan<- BatchResult) {
	go func() {
		defer close(results)

		d.mu.RLock()
		var matching []*Subscription
		for _, sub := range d.subscriptions {
			if sub.Matches(event.Type) {
				matching = append(matching, sub)
			}
		}
		d.mu.RUnlock()

		var wg sync.WaitGroup
		for _, sub := range matching {
			wg.Add(1)
			go func(subscription *Subscription) {
				defer wg.Done()

				sender := d.sender
				if subscription.Secret != "" {
					sender = NewSender(SenderConfig{
						SigningSecret:     subscription.Secret,
						HTTPClient:        d.sender.httpClient,
						Timeout:           d.sender.timeout,
						MaxRetries:        d.sender.maxRetries,
						RetryBackoff:      d.sender.retryBackoff,
						MaxBackoff:        d.sender.maxBackoff,
						BackoffMultiplier: d.sender.backoffMultiplier,
					})
				}

				deliveryResults, err := sender.SendWithResults(ctx, subscription.URL, event)
				select {
				case results <- BatchResult{
					URL:     subscription.URL,
					Results: deliveryResults,
					Error:   err,
				}:
				case <-ctx.Done():
				}
			}(sub)
		}

		wg.Wait()
	}()
}
