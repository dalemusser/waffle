// websocket/websocket.go
package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Conn wraps a WebSocket connection with additional functionality.
type Conn struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool

	// OnClose is called when the connection is closed.
	OnClose func()
}

// AcceptOptions configures the WebSocket upgrade.
type AcceptOptions struct {
	// Subprotocols lists the server's supported subprotocols.
	// The first match with a client-requested subprotocol is selected.
	Subprotocols []string

	// InsecureSkipVerify disables origin verification.
	// Only use for development/testing.
	InsecureSkipVerify bool

	// OriginPatterns specifies allowed origin patterns.
	// Use "*" for any origin (not recommended for production).
	// Example: []string{"https://example.com", "https://*.example.com"}
	OriginPatterns []string

	// CompressionMode controls message compression.
	// Defaults to CompressionDisabled.
	CompressionMode CompressionMode

	// CompressionThreshold is the minimum message size to compress.
	// Messages smaller than this are sent uncompressed.
	// Defaults to 512 bytes.
	CompressionThreshold int
}

// CompressionMode specifies the compression mode for messages.
type CompressionMode int

const (
	// CompressionDisabled disables compression.
	CompressionDisabled CompressionMode = iota

	// CompressionContextTakeover enables compression with context takeover.
	// More efficient but uses more memory per connection.
	CompressionContextTakeover

	// CompressionNoContextTakeover enables compression without context takeover.
	// Less memory per connection but slightly less efficient.
	CompressionNoContextTakeover
)

// DefaultAcceptOptions returns sensible defaults for accepting connections.
func DefaultAcceptOptions() AcceptOptions {
	return AcceptOptions{
		CompressionMode:      CompressionDisabled,
		CompressionThreshold: 512,
	}
}

// Accept upgrades an HTTP connection to a WebSocket connection.
func Accept(w http.ResponseWriter, r *http.Request, opts *AcceptOptions) (*Conn, error) {
	var wsOpts *websocket.AcceptOptions
	if opts != nil {
		wsOpts = &websocket.AcceptOptions{
			Subprotocols:         opts.Subprotocols,
			InsecureSkipVerify:   opts.InsecureSkipVerify,
			OriginPatterns:       opts.OriginPatterns,
			CompressionMode:      toWSCompressionMode(opts.CompressionMode),
			CompressionThreshold: opts.CompressionThreshold,
		}
	}

	conn, err := websocket.Accept(w, r, wsOpts)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: conn}, nil
}

// toWSCompressionMode converts our CompressionMode to websocket.CompressionMode.
func toWSCompressionMode(mode CompressionMode) websocket.CompressionMode {
	switch mode {
	case CompressionContextTakeover:
		return websocket.CompressionContextTakeover
	case CompressionNoContextTakeover:
		return websocket.CompressionNoContextTakeover
	default:
		return websocket.CompressionDisabled
	}
}

// Handler creates an http.Handler that upgrades connections and calls the handler.
func Handler(opts *AcceptOptions, handler func(*Conn)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := Accept(w, r, opts)
		if err != nil {
			// Accept already wrote the error response
			return
		}
		handler(conn)
	})
}

// HandlerFunc creates an http.HandlerFunc that upgrades connections.
func HandlerFunc(opts *AcceptOptions, handler func(*Conn)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := Accept(w, r, opts)
		if err != nil {
			return
		}
		handler(conn)
	}
}

// Close closes the WebSocket connection with a normal closure.
func (c *Conn) Close() error {
	return c.CloseWithReason(StatusNormalClosure, "")
}

// CloseWithReason closes the WebSocket connection with a status code and reason.
func (c *Conn) CloseWithReason(code StatusCode, reason string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	onClose := c.OnClose
	c.mu.Unlock()

	if onClose != nil {
		onClose()
	}

	return c.conn.Close(websocket.StatusCode(code), reason)
}

// IsClosed returns true if the connection has been closed.
func (c *Conn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Read reads a message from the connection.
// It blocks until a message is received or the context is canceled.
func (c *Conn) Read(ctx context.Context) (MessageType, []byte, error) {
	msgType, data, err := c.conn.Read(ctx)
	if err != nil {
		return 0, nil, err
	}
	return MessageType(msgType), data, nil
}

// ReadText reads a text message from the connection.
// Returns an error if the message is not a text message.
func (c *Conn) ReadText(ctx context.Context) (string, error) {
	msgType, data, err := c.conn.Read(ctx)
	if err != nil {
		return "", err
	}
	if msgType != websocket.MessageText {
		return "", ErrExpectedTextMessage
	}
	return string(data), nil
}

// ReadBinary reads a binary message from the connection.
// Returns an error if the message is not a binary message.
func (c *Conn) ReadBinary(ctx context.Context) ([]byte, error) {
	msgType, data, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if msgType != websocket.MessageBinary {
		return nil, ErrExpectedBinaryMessage
	}
	return data, nil
}

// Write writes a message to the connection.
// It is safe to call concurrently.
func (c *Conn) Write(ctx context.Context, msgType MessageType, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConnectionClosed
	}

	return c.conn.Write(ctx, websocket.MessageType(msgType), data)
}

// WriteText writes a text message to the connection.
func (c *Conn) WriteText(ctx context.Context, msg string) error {
	return c.Write(ctx, MessageText, []byte(msg))
}

// WriteBinary writes a binary message to the connection.
func (c *Conn) WriteBinary(ctx context.Context, data []byte) error {
	return c.Write(ctx, MessageBinary, data)
}

// Ping sends a ping to the peer and waits for a pong.
// This is useful for keeping the connection alive and detecting dead connections.
func (c *Conn) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// SetReadLimit sets the maximum message size that can be read.
// The default is 32KB.
func (c *Conn) SetReadLimit(limit int64) {
	c.conn.SetReadLimit(limit)
}

// Subprotocol returns the negotiated subprotocol.
// Returns an empty string if no subprotocol was negotiated.
func (c *Conn) Subprotocol() string {
	return c.conn.Subprotocol()
}

// RemoteAddr returns the remote address (from the original HTTP request).
// Note: This requires storing it during Accept, which we don't currently do.
// Use the request's RemoteAddr in your handler instead.

// MessageType represents the type of WebSocket message.
type MessageType int

const (
	// MessageText is a text message (UTF-8 encoded).
	MessageText MessageType = MessageType(websocket.MessageText)

	// MessageBinary is a binary message.
	MessageBinary MessageType = MessageType(websocket.MessageBinary)
)

// StatusCode represents a WebSocket close status code.
type StatusCode int

const (
	StatusNormalClosure   StatusCode = StatusCode(websocket.StatusNormalClosure)
	StatusGoingAway       StatusCode = StatusCode(websocket.StatusGoingAway)
	StatusProtocolError   StatusCode = StatusCode(websocket.StatusProtocolError)
	StatusUnsupportedData StatusCode = StatusCode(websocket.StatusUnsupportedData)
	StatusNoStatusRcvd    StatusCode = StatusCode(websocket.StatusNoStatusRcvd)
	StatusAbnormalClosure StatusCode = StatusCode(websocket.StatusAbnormalClosure)
	StatusInvalidPayload  StatusCode = StatusCode(websocket.StatusInvalidFramePayloadData)
	StatusPolicyViolation StatusCode = StatusCode(websocket.StatusPolicyViolation)
	StatusMessageTooBig   StatusCode = StatusCode(websocket.StatusMessageTooBig)
	StatusMandatoryExt    StatusCode = StatusCode(websocket.StatusMandatoryExtension)
	StatusInternalError   StatusCode = StatusCode(websocket.StatusInternalError)
	StatusServiceRestart  StatusCode = StatusCode(websocket.StatusServiceRestart)
	StatusTryAgainLater   StatusCode = StatusCode(websocket.StatusTryAgainLater)
	StatusBadGateway      StatusCode = StatusCode(websocket.StatusBadGateway)
)

// Config holds configuration for WebSocket connections.
type Config struct {
	// ReadTimeout is the maximum time to wait for a message.
	// Zero means no timeout.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum time to wait when writing a message.
	// Zero means no timeout.
	WriteTimeout time.Duration

	// PingInterval is how often to send pings to keep the connection alive.
	// Zero disables automatic pings.
	PingInterval time.Duration

	// PongTimeout is how long to wait for a pong response.
	// Defaults to PingInterval if not set.
	PongTimeout time.Duration

	// MaxMessageSize is the maximum message size to accept.
	// Defaults to 32KB.
	MaxMessageSize int64
}

// DefaultConfig returns sensible defaults for WebSocket configuration.
func DefaultConfig() Config {
	return Config{
		ReadTimeout:    0,
		WriteTimeout:   10 * time.Second,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 32 * 1024, // 32KB
	}
}

// RunWithConfig runs a connection handler with the given configuration.
// It handles ping/pong automatically and respects timeouts.
func RunWithConfig(ctx context.Context, conn *Conn, cfg Config, handler func(ctx context.Context, msgType MessageType, data []byte) error) error {
	if cfg.MaxMessageSize > 0 {
		conn.SetReadLimit(cfg.MaxMessageSize)
	}

	// Start ping goroutine if enabled
	if cfg.PingInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.PingInterval)
			defer ticker.Stop()

			pongTimeout := cfg.PongTimeout
			if pongTimeout == 0 {
				pongTimeout = cfg.PingInterval
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pingCtx, cancel := context.WithTimeout(ctx, pongTimeout)
					err := conn.Ping(pingCtx)
					cancel()
					if err != nil {
						conn.CloseWithReason(StatusGoingAway, "ping timeout")
						return
					}
				}
			}
		}()
	}

	// Read loop
	for {
		readCtx := ctx
		if cfg.ReadTimeout > 0 {
			var cancel context.CancelFunc
			readCtx, cancel = context.WithTimeout(ctx, cfg.ReadTimeout)
			defer cancel()
		}

		msgType, data, err := conn.Read(readCtx)
		if err != nil {
			return err
		}

		if err := handler(ctx, msgType, data); err != nil {
			return err
		}
	}
}
