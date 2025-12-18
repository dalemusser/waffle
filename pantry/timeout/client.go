// timeout/client.go
package timeout

import (
	"context"
	"net"
	"net/http"
	"time"
)

// ClientConfig configures an HTTP client with various timeouts.
type ClientConfig struct {
	// Total timeout for the entire request (including redirects).
	// Default: 30 seconds.
	Timeout time.Duration

	// DialTimeout is the maximum time to establish a connection.
	// Default: 10 seconds.
	DialTimeout time.Duration

	// TLSHandshakeTimeout is the maximum time for TLS handshake.
	// Default: 10 seconds.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the maximum time to wait for response headers.
	// Default: 10 seconds.
	ResponseHeaderTimeout time.Duration

	// IdleConnTimeout is the maximum time an idle connection will remain open.
	// Default: 90 seconds.
	IdleConnTimeout time.Duration

	// MaxIdleConns controls the maximum number of idle connections.
	// Default: 100.
	MaxIdleConns int

	// MaxIdleConnsPerHost controls idle connections per host.
	// Default: 10.
	MaxIdleConnsPerHost int

	// MaxConnsPerHost limits total connections per host.
	// Default: 0 (no limit).
	MaxConnsPerHost int

	// ExpectContinueTimeout is the time to wait for 100-continue response.
	// Default: 1 second.
	ExpectContinueTimeout time.Duration
}

// DefaultClientConfig returns sensible client timeout defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:               30 * time.Second,
		DialTimeout:           10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewClient creates an HTTP client with the given timeout configuration.
func NewClient(cfg ClientConfig) *http.Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.TLSHandshakeTimeout <= 0 {
		cfg.TLSHandshakeTimeout = 10 * time.Second
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = 10 * time.Second
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 100
	}
	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = 10
	}
	if cfg.ExpectContinueTimeout <= 0 {
		cfg.ExpectContinueTimeout = 1 * time.Second
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}
}

// Client returns an HTTP client with default timeout settings.
func Client() *http.Client {
	return NewClient(DefaultClientConfig())
}

// QuickClient returns an HTTP client optimized for quick requests.
// Uses shorter timeouts suitable for internal service calls.
func QuickClient() *http.Client {
	return NewClient(ClientConfig{
		Timeout:               5 * time.Second,
		DialTimeout:           2 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		ExpectContinueTimeout: 500 * time.Millisecond,
	})
}

// LongClient returns an HTTP client for long-running requests.
// Uses longer timeouts suitable for file uploads, reports, etc.
func LongClient() *http.Client {
	return NewClient(ClientConfig{
		Timeout:               5 * time.Minute,
		DialTimeout:           30 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       120 * time.Second,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
		ExpectContinueTimeout: 2 * time.Second,
	})
}

// Transport wraps an http.RoundTripper to enforce per-request timeouts.
type Transport struct {
	// Base is the underlying transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// Timeout is the per-request timeout.
	Timeout time.Duration
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	if t.Timeout <= 0 {
		return base.RoundTrip(req)
	}

	// Only add timeout if context doesn't already have one
	if _, ok := req.Context().Deadline(); !ok {
		ctx, cancel := context.WithTimeout(req.Context(), t.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	return base.RoundTrip(req)
}

// WithTransportTimeout wraps an existing transport with per-request timeout.
func WithTransportTimeout(base http.RoundTripper, timeout time.Duration) *Transport {
	return &Transport{
		Base:    base,
		Timeout: timeout,
	}
}

// Do performs an HTTP request with the given timeout.
// If client is nil, uses the default client.
func Do(ctx context.Context, client *http.Client, req *http.Request, timeout time.Duration) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return client.Do(req.WithContext(ctx))
}

// Get performs a GET request with the given timeout.
func Get(ctx context.Context, url string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return Do(ctx, nil, req, timeout)
}
