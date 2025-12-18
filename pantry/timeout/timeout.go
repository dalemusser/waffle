// timeout/timeout.go
package timeout

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Config configures the timeout middleware.
type Config struct {
	// Timeout is the maximum duration for handling a request.
	// Default: 30 seconds.
	Timeout time.Duration

	// OnTimeout is called when a request times out.
	// If nil, a default 503 Service Unavailable response is sent.
	OnTimeout func(w http.ResponseWriter, r *http.Request)

	// Skipper determines whether to skip timeout for a request.
	// Useful for SSE, WebSocket upgrades, or long-polling endpoints.
	Skipper func(r *http.Request) bool

	// ErrorHandler handles panics that occur during request processing.
	// If nil, panics are re-raised after timeout cleanup.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err interface{})
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
	}
}

// Middleware returns timeout middleware with the given configuration.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip timeout for this request
			if cfg.Skipper != nil && cfg.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout)
			defer cancel()

			// Create wrapped response writer
			tw := &timeoutWriter{
				ResponseWriter: w,
				done:           make(chan struct{}),
			}

			// Channel for panic recovery
			panicChan := make(chan interface{}, 1)

			// Run handler in goroutine
			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
					close(tw.done)
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			// Wait for completion, timeout, or panic
			select {
			case <-tw.done:
				// Handler completed normally
				tw.mu.Lock()
				defer tw.mu.Unlock()
				// Response already written by handler

			case p := <-panicChan:
				// Handler panicked
				tw.mu.Lock()
				tw.timedOut = true
				tw.mu.Unlock()

				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(w, r, p)
				} else {
					// Re-panic to let recovery middleware handle it
					panic(p)
				}

			case <-ctx.Done():
				// Timeout occurred
				tw.mu.Lock()
				tw.timedOut = true
				tw.mu.Unlock()

				if cfg.OnTimeout != nil {
					cfg.OnTimeout(w, r)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte("Service Unavailable: request timeout"))
				}
			}
		})
	}
}

// Simple returns timeout middleware with default settings (30 seconds).
func Simple() func(http.Handler) http.Handler {
	return Middleware(DefaultConfig())
}

// WithTimeout returns timeout middleware with the specified duration.
func WithTimeout(d time.Duration) func(http.Handler) http.Handler {
	return Middleware(Config{Timeout: d})
}

// timeoutWriter wraps http.ResponseWriter to prevent writes after timeout.
type timeoutWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	timedOut   bool
	written    bool
	statusCode int
	done       chan struct{}
}

func (tw *timeoutWriter) Header() http.Header {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return http.Header{}
	}
	return tw.ResponseWriter.Header()
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, context.DeadlineExceeded
	}
	if !tw.written {
		tw.written = true
		if tw.statusCode == 0 {
			tw.statusCode = http.StatusOK
		}
	}
	return tw.ResponseWriter.Write(b)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.written {
		return
	}
	tw.written = true
	tw.statusCode = code
	tw.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the underlying ResponseWriter.
func (tw *timeoutWriter) Unwrap() http.ResponseWriter {
	return tw.ResponseWriter
}

// Flush implements http.Flusher.
func (tw *timeoutWriter) Flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	if f, ok := tw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Common skippers for timeout middleware.

// SkipWebSocket returns true for WebSocket upgrade requests.
func SkipWebSocket(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket"
}

// SkipSSE returns true for Server-Sent Events requests.
func SkipSSE(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/event-stream"
}

// SkipPaths returns a skipper that skips the given paths.
func SkipPaths(paths ...string) func(*http.Request) bool {
	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[p] = true
	}
	return func(r *http.Request) bool {
		return pathSet[r.URL.Path]
	}
}

// SkipMethods returns a skipper that skips the given HTTP methods.
func SkipMethods(methods ...string) func(*http.Request) bool {
	methodSet := make(map[string]bool, len(methods))
	for _, m := range methods {
		methodSet[m] = true
	}
	return func(r *http.Request) bool {
		return methodSet[r.Method]
	}
}

// CombineSkippers combines multiple skippers with OR logic.
func CombineSkippers(skippers ...func(*http.Request) bool) func(*http.Request) bool {
	return func(r *http.Request) bool {
		for _, skip := range skippers {
			if skip(r) {
				return true
			}
		}
		return false
	}
}
