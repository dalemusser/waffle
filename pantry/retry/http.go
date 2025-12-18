// retry/http.go
package retry

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// HTTPConfig configures HTTP retry behavior.
type HTTPConfig struct {
	Config

	// RetryStatusCodes specifies which status codes to retry.
	// Default: 429, 500, 502, 503, 504.
	RetryStatusCodes []int

	// RetryOnConnectionError retries on connection errors.
	// Default: true.
	RetryOnConnectionError bool

	// RespectRetryAfter respects the Retry-After header.
	// Default: true.
	RespectRetryAfter bool

	// MaxRetryAfter caps the Retry-After delay.
	// Default: 5 minutes.
	MaxRetryAfter time.Duration
}

// DefaultHTTPConfig returns sensible HTTP retry defaults.
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Config:                 DefaultConfig(),
		RetryStatusCodes:       []int{429, 500, 502, 503, 504},
		RetryOnConnectionError: true,
		RespectRetryAfter:      true,
		MaxRetryAfter:          5 * time.Minute,
	}
}

// Transport wraps an http.RoundTripper with retry logic.
type Transport struct {
	// Base is the underlying transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// Config is the retry configuration.
	Config HTTPConfig
}

// RoundTrip implements http.RoundTripper with retries.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	cfg := t.Config
	if cfg.MaxAttempts <= 0 {
		cfg.Config = DefaultConfig()
	}
	if len(cfg.RetryStatusCodes) == 0 {
		cfg.RetryStatusCodes = []int{429, 500, 502, 503, 504}
	}
	if cfg.MaxRetryAfter <= 0 {
		cfg.MaxRetryAfter = 5 * time.Minute
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// Buffer the body if it exists and is retryable
	var bodyBytes []byte
	if req.Body != nil && req.GetBody == nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	delay := cfg.InitialDelay
	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context
		if req.Context().Err() != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, req.Context().Err()
		}

		// Reset body for retry
		if attempt > 1 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		// Make request
		resp, err := base.RoundTrip(req)

		// Success
		if err == nil && !t.shouldRetryStatus(resp.StatusCode, cfg.RetryStatusCodes) {
			return resp, nil
		}

		// Close previous response body
		if lastResp != nil && lastResp.Body != nil {
			lastResp.Body.Close()
		}

		lastResp = resp
		lastErr = err

		// Check if we should retry
		if err != nil && !cfg.RetryOnConnectionError {
			return nil, err
		}
		if resp != nil && !t.shouldRetryStatus(resp.StatusCode, cfg.RetryStatusCodes) {
			return resp, nil
		}

		// Last attempt - return result
		if attempt >= cfg.MaxAttempts {
			if resp != nil {
				return resp, nil
			}
			return nil, lastErr
		}

		// Calculate delay
		actualDelay := addJitter(delay, cfg.Jitter)

		// Check Retry-After header
		if resp != nil && cfg.RespectRetryAfter {
			if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > 0 {
				if retryAfter > cfg.MaxRetryAfter {
					retryAfter = cfg.MaxRetryAfter
				}
				actualDelay = retryAfter
			}
		}

		// Close response body before retry
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		// Notify callback
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, actualDelay)
		}

		// Wait before retry
		select {
		case <-req.Context().Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, req.Context().Err()
		case <-time.After(actualDelay):
		}

		// Increase delay
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

// shouldRetryStatus returns true if the status code should be retried.
func (t *Transport) shouldRetryStatus(status int, codes []int) bool {
	for _, code := range codes {
		if status == code {
			return true
		}
	}
	return false
}

// parseRetryAfter parses the Retry-After header value.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try parsing as seconds
	var seconds int
	if _, err := time.Parse(time.RFC1123, value); err != nil {
		// Not a date, try as seconds
		for _, c := range value {
			if c >= '0' && c <= '9' {
				seconds = seconds*10 + int(c-'0')
			} else {
				return 0
			}
		}
		return time.Duration(seconds) * time.Second
	}

	// Parse as HTTP date
	t, err := time.Parse(time.RFC1123, value)
	if err != nil {
		return 0
	}
	delay := time.Until(t)
	if delay < 0 {
		return 0
	}
	return delay
}

// Client returns an HTTP client with retry support.
func Client() *http.Client {
	return &http.Client{
		Transport: &Transport{Config: DefaultHTTPConfig()},
	}
}

// ClientWithConfig returns an HTTP client with custom retry configuration.
func ClientWithConfig(cfg HTTPConfig) *http.Client {
	return &http.Client{
		Transport: &Transport{Config: cfg},
	}
}

// ClientWithBase returns an HTTP client with a custom base transport.
func ClientWithBase(base http.RoundTripper, cfg HTTPConfig) *http.Client {
	return &http.Client{
		Transport: &Transport{Base: base, Config: cfg},
	}
}

// Do performs an HTTP request with retries.
func DoHTTP(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	if client == nil {
		client = Client()
	}
	return client.Do(req.WithContext(ctx))
}

// Get performs a GET request with retries.
func Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DoHTTP(ctx, nil, req)
}

// IsRetryableStatus returns true if the HTTP status code is typically retryable.
func IsRetryableStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// IsRetryableError returns true if the error is typically retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Connection errors are generally retryable
	// Context cancellation is not
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	return true
}
