// sse/sse.go
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Event represents a Server-Sent Event.
type Event struct {
	// ID is the event ID for client reconnection tracking.
	// The client sends this as Last-Event-ID header on reconnect.
	ID string

	// Event is the event type. Clients can listen for specific types.
	// If empty, the client receives it as a "message" event.
	Event string

	// Data is the event payload. Multiple data fields create multi-line data.
	Data string

	// Retry suggests a reconnection time in milliseconds to the client.
	// Only sent if non-zero.
	Retry int
}

// NewEvent creates a new event with the given data.
func NewEvent(data string) *Event {
	return &Event{Data: data}
}

// NewEventWithType creates a new event with type and data.
func NewEventWithType(eventType, data string) *Event {
	return &Event{Event: eventType, Data: data}
}

// NewJSONEvent creates an event with JSON-encoded data.
func NewJSONEvent(eventType string, v any) (*Event, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Event{Event: eventType, Data: string(data)}, nil
}

// MustJSONEvent creates an event with JSON-encoded data, panicking on error.
func MustJSONEvent(eventType string, v any) *Event {
	event, err := NewJSONEvent(eventType, v)
	if err != nil {
		panic(err)
	}
	return event
}

// WithID sets the event ID.
func (e *Event) WithID(id string) *Event {
	e.ID = id
	return e
}

// WithRetry sets the retry interval in milliseconds.
func (e *Event) WithRetry(ms int) *Event {
	e.Retry = ms
	return e
}

// Bytes serializes the event to SSE format.
func (e *Event) Bytes() []byte {
	var buf strings.Builder

	if e.ID != "" {
		buf.WriteString("id: ")
		buf.WriteString(e.ID)
		buf.WriteByte('\n')
	}

	if e.Event != "" {
		buf.WriteString("event: ")
		buf.WriteString(e.Event)
		buf.WriteByte('\n')
	}

	if e.Retry > 0 {
		buf.WriteString("retry: ")
		buf.WriteString(strconv.Itoa(e.Retry))
		buf.WriteByte('\n')
	}

	// Handle multi-line data
	if e.Data != "" {
		lines := strings.Split(e.Data, "\n")
		for _, line := range lines {
			buf.WriteString("data: ")
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}

	buf.WriteByte('\n')
	return []byte(buf.String())
}

// String returns the SSE-formatted event as a string.
func (e *Event) String() string {
	return string(e.Bytes())
}

// Stream represents an SSE connection to a single client.
type Stream struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	closed  bool
	done    chan struct{}

	// LastEventID is the ID sent by the client on reconnection.
	LastEventID string
}

// NewStream creates a new SSE stream for the given response writer.
// Returns an error if the response writer doesn't support flushing.
func NewStream(w http.ResponseWriter, r *http.Request) (*Stream, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrFlushNotSupported
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get last event ID from client (for reconnection)
	lastEventID := r.Header.Get("Last-Event-ID")

	return &Stream{
		w:           w,
		flusher:     flusher,
		done:        make(chan struct{}),
		LastEventID: lastEventID,
	}, nil
}

// Send sends an event to the client.
func (s *Stream) Send(event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	_, err := s.w.Write(event.Bytes())
	if err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

// SendData sends a simple data-only event.
func (s *Stream) SendData(data string) error {
	return s.Send(NewEvent(data))
}

// SendEvent sends an event with type and data.
func (s *Stream) SendEvent(eventType, data string) error {
	return s.Send(NewEventWithType(eventType, data))
}

// SendJSON sends a JSON-encoded event.
func (s *Stream) SendJSON(eventType string, v any) error {
	event, err := NewJSONEvent(eventType, v)
	if err != nil {
		return err
	}
	return s.Send(event)
}

// SendComment sends a comment (used as keep-alive).
// Comments start with a colon and are ignored by clients.
func (s *Stream) SendComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	_, err := fmt.Fprintf(s.w, ": %s\n\n", comment)
	if err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

// SendRetry sends a retry interval to the client.
func (s *Stream) SendRetry(ms int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	_, err := fmt.Fprintf(s.w, "retry: %d\n\n", ms)
	if err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

// Close closes the stream.
func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true
	close(s.done)
}

// Done returns a channel that's closed when the stream is closed.
func (s *Stream) Done() <-chan struct{} {
	return s.done
}

// IsClosed returns true if the stream has been closed.
func (s *Stream) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Handler creates an http.Handler for SSE connections.
func Handler(handler func(*Stream, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream, err := NewStream(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		handler(stream, r)
	})
}

// HandlerFunc creates an http.HandlerFunc for SSE connections.
func HandlerFunc(handler func(*Stream, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream, err := NewStream(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		handler(stream, r)
	}
}

// Config holds configuration for SSE streams.
type Config struct {
	// KeepAliveInterval is how often to send keep-alive comments.
	// Zero disables keep-alive. Default: 30 seconds.
	KeepAliveInterval time.Duration

	// RetryInterval is the reconnection interval to suggest to clients.
	// Zero means don't send retry. Default: 3 seconds.
	RetryInterval time.Duration

	// OnConnect is called when a client connects.
	OnConnect func(*Stream)

	// OnDisconnect is called when a client disconnects.
	OnDisconnect func(*Stream)
}

// DefaultConfig returns sensible defaults for SSE.
func DefaultConfig() Config {
	return Config{
		KeepAliveInterval: 30 * time.Second,
		RetryInterval:     3 * time.Second,
	}
}

// Serve runs an SSE stream with the given configuration.
// It handles keep-alive and listens for client disconnect.
// The events channel receives events to send to the client.
// Returns when the context is canceled or the client disconnects.
func Serve(ctx context.Context, stream *Stream, cfg Config, events <-chan *Event) error {
	// Send initial retry interval
	if cfg.RetryInterval > 0 {
		stream.SendRetry(int(cfg.RetryInterval.Milliseconds()))
	}

	if cfg.OnConnect != nil {
		cfg.OnConnect(stream)
	}

	defer func() {
		stream.Close()
		if cfg.OnDisconnect != nil {
			cfg.OnDisconnect(stream)
		}
	}()

	var keepAliveTicker *time.Ticker
	var keepAliveChan <-chan time.Time

	if cfg.KeepAliveInterval > 0 {
		keepAliveTicker = time.NewTicker(cfg.KeepAliveInterval)
		defer keepAliveTicker.Stop()
		keepAliveChan = keepAliveTicker.C
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-events:
			if !ok {
				return nil // Channel closed
			}
			if err := stream.Send(event); err != nil {
				return err
			}

		case <-keepAliveChan:
			if err := stream.SendComment("keep-alive"); err != nil {
				return err
			}
		}
	}
}

// ServeFunc is like Serve but takes a function that generates events.
// The function should send events to the provided channel and return when done.
func ServeFunc(ctx context.Context, stream *Stream, cfg Config, fn func(ctx context.Context, send func(*Event) error) error) error {
	// Send initial retry interval
	if cfg.RetryInterval > 0 {
		stream.SendRetry(int(cfg.RetryInterval.Milliseconds()))
	}

	if cfg.OnConnect != nil {
		cfg.OnConnect(stream)
	}

	defer func() {
		stream.Close()
		if cfg.OnDisconnect != nil {
			cfg.OnDisconnect(stream)
		}
	}()

	// Create context that cancels on client disconnect
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start keep-alive goroutine
	if cfg.KeepAliveInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.KeepAliveInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := stream.SendComment("keep-alive"); err != nil {
						cancel()
						return
					}
				}
			}
		}()
	}

	// Run user function
	send := func(event *Event) error {
		return stream.Send(event)
	}

	return fn(ctx, send)
}
