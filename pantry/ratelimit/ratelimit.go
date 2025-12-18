// ratelimit/ratelimit.go
package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// Limiter implements a token bucket rate limiter.
type Limiter struct {
	mu       sync.Mutex
	rate     float64   // tokens per second
	burst    int       // maximum bucket size
	tokens   float64   // current tokens
	lastTime time.Time // last token update
}

// New creates a new rate limiter.
// rate is the number of requests allowed per second.
// burst is the maximum number of requests allowed in a burst.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow reports whether a request is allowed.
// It consumes one token if available.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.lastTime = now

	// Add tokens based on elapsed time
	l.tokens += elapsed * l.rate
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	if l.tokens >= 1 {
		l.tokens--
		return true
	}

	return false
}

// AllowN reports whether n requests are allowed.
// It consumes n tokens if available.
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.lastTime = now

	l.tokens += elapsed * l.rate
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		return true
	}

	return false
}

// Tokens returns the current number of available tokens.
func (l *Limiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()

	tokens := l.tokens + elapsed*l.rate
	if tokens > float64(l.burst) {
		tokens = float64(l.burst)
	}

	return tokens
}

// KeyLimiter provides per-key rate limiting (e.g., per IP address).
type KeyLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*entry
	rate     float64
	burst    int
	ttl      time.Duration
}

type entry struct {
	limiter  *Limiter
	lastSeen time.Time
}

// NewKeyLimiter creates a rate limiter that tracks limits per key.
// rate is requests per second, burst is max burst size.
// ttl is how long to keep inactive keys before cleanup.
func NewKeyLimiter(rate float64, burst int, ttl time.Duration) *KeyLimiter {
	kl := &KeyLimiter{
		limiters: make(map[string]*entry),
		rate:     rate,
		burst:    burst,
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go kl.cleanup()

	return kl
}

// Allow checks if the request for the given key is allowed.
func (kl *KeyLimiter) Allow(key string) bool {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	e, exists := kl.limiters[key]
	if !exists {
		e = &entry{
			limiter:  New(kl.rate, kl.burst),
			lastSeen: time.Now(),
		}
		kl.limiters[key] = e
	} else {
		e.lastSeen = time.Now()
	}

	return e.limiter.Allow()
}

// AllowN checks if n requests for the given key are allowed.
func (kl *KeyLimiter) AllowN(key string, n int) bool {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	e, exists := kl.limiters[key]
	if !exists {
		e = &entry{
			limiter:  New(kl.rate, kl.burst),
			lastSeen: time.Now(),
		}
		kl.limiters[key] = e
	} else {
		e.lastSeen = time.Now()
	}

	return e.limiter.AllowN(n)
}

// cleanup removes stale entries periodically.
func (kl *KeyLimiter) cleanup() {
	ticker := time.NewTicker(kl.ttl)
	defer ticker.Stop()

	for range ticker.C {
		kl.mu.Lock()
		now := time.Now()
		for key, e := range kl.limiters {
			if now.Sub(e.lastSeen) > kl.ttl {
				delete(kl.limiters, key)
			}
		}
		kl.mu.Unlock()
	}
}

// Size returns the number of tracked keys.
func (kl *KeyLimiter) Size() int {
	kl.mu.RLock()
	defer kl.mu.RUnlock()
	return len(kl.limiters)
}

// KeyFunc extracts a key from an HTTP request for rate limiting.
type KeyFunc func(r *http.Request) string

// IPKeyFunc returns the client IP address as the rate limit key.
// It checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func IPKeyFunc(r *http.Request) string {
	// Check X-Forwarded-For first (may contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (strip port)
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// PathKeyFunc returns the request path as the rate limit key.
func PathKeyFunc(r *http.Request) string {
	return r.URL.Path
}

// MethodPathKeyFunc returns method + path as the rate limit key.
func MethodPathKeyFunc(r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// IPPathKeyFunc returns IP + path as the rate limit key.
func IPPathKeyFunc(r *http.Request) string {
	return IPKeyFunc(r) + " " + r.URL.Path
}

// Config configures the rate limit middleware.
type Config struct {
	// Rate is requests per second. Required.
	Rate float64

	// Burst is the maximum burst size. Required.
	Burst int

	// KeyFunc extracts the rate limit key from requests.
	// Defaults to IPKeyFunc.
	KeyFunc KeyFunc

	// TTL is how long to keep inactive keys.
	// Defaults to 1 hour.
	TTL time.Duration

	// StatusCode is the HTTP status when rate limited.
	// Defaults to 429 Too Many Requests.
	StatusCode int

	// Message is the response body when rate limited.
	// Defaults to "rate limit exceeded".
	Message string

	// OnLimited is called when a request is rate limited.
	// Can be used for logging or custom responses.
	OnLimited func(w http.ResponseWriter, r *http.Request)

	// Skip returns true to skip rate limiting for a request.
	// Useful for health checks, admin endpoints, etc.
	Skip func(r *http.Request) bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Rate:       10,
		Burst:      20,
		KeyFunc:    IPKeyFunc,
		TTL:        time.Hour,
		StatusCode: http.StatusTooManyRequests,
		Message:    "rate limit exceeded",
	}
}

// Middleware returns HTTP middleware that applies rate limiting.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = IPKeyFunc
	}
	if cfg.TTL == 0 {
		cfg.TTL = time.Hour
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusTooManyRequests
	}
	if cfg.Message == "" {
		cfg.Message = "rate limit exceeded"
	}

	limiter := NewKeyLimiter(cfg.Rate, cfg.Burst, cfg.TTL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip function
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := cfg.KeyFunc(r)

			if !limiter.Allow(key) {
				if cfg.OnLimited != nil {
					cfg.OnLimited(w, r)
					return
				}

				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(cfg.StatusCode)
				w.Write([]byte(cfg.Message))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MiddlewareWithLimiter returns middleware using a provided KeyLimiter.
// Useful when you need access to the limiter for metrics or management.
func MiddlewareWithLimiter(limiter *KeyLimiter, cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = IPKeyFunc
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusTooManyRequests
	}
	if cfg.Message == "" {
		cfg.Message = "rate limit exceeded"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := cfg.KeyFunc(r)

			if !limiter.Allow(key) {
				if cfg.OnLimited != nil {
					cfg.OnLimited(w, r)
					return
				}

				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(cfg.StatusCode)
				w.Write([]byte(cfg.Message))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Simple returns a simple rate limit middleware with the given rate and burst.
// Uses IP-based rate limiting with sensible defaults.
func Simple(rate float64, burst int) func(http.Handler) http.Handler {
	return Middleware(Config{
		Rate:  rate,
		Burst: burst,
	})
}
