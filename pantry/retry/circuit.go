// retry/circuit.go
package retry

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Circuit breaker errors.
var (
	ErrCircuitOpen    = errors.New("retry: circuit breaker is open")
	ErrCircuitTimeout = errors.New("retry: circuit breaker timeout")
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitConfig configures a circuit breaker.
type CircuitConfig struct {
	// FailureThreshold is the number of failures before opening the circuit.
	// Default: 5.
	FailureThreshold int

	// SuccessThreshold is the number of successes needed to close from half-open.
	// Default: 2.
	SuccessThreshold int

	// Timeout is how long to wait before transitioning from open to half-open.
	// Default: 30 seconds.
	Timeout time.Duration

	// MaxHalfOpenRequests is the max concurrent requests in half-open state.
	// Default: 1.
	MaxHalfOpenRequests int

	// IsFailure determines if an error should count as a failure.
	// Default: all non-nil errors are failures.
	IsFailure func(error) bool

	// OnStateChange is called when the circuit state changes.
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitConfig returns sensible circuit breaker defaults.
func DefaultCircuitConfig() CircuitConfig {
	return CircuitConfig{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 1,
	}
}

// Circuit implements the circuit breaker pattern.
type Circuit struct {
	mu sync.Mutex

	cfg CircuitConfig

	state            CircuitState
	failures         int
	successes        int
	lastFailure      time.Time
	halfOpenRequests int
}

// NewCircuit creates a new circuit breaker.
func NewCircuit(cfg CircuitConfig) *Circuit {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxHalfOpenRequests <= 0 {
		cfg.MaxHalfOpenRequests = 1
	}

	return &Circuit{
		cfg:   cfg,
		state: CircuitClosed,
	}
}

// Do executes a function with circuit breaker protection.
func (c *Circuit) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := c.allowRequest(); err != nil {
		return err
	}

	// Track half-open requests
	c.mu.Lock()
	isHalfOpen := c.state == CircuitHalfOpen
	if isHalfOpen {
		c.halfOpenRequests++
	}
	c.mu.Unlock()

	// Execute function
	err := fn(ctx)

	// Record result
	c.recordResult(err, isHalfOpen)

	return err
}

// CircuitDoWithResult executes a function that returns a value with circuit breaker protection.
func CircuitDoWithResult[T any](c *Circuit, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	if err := c.allowRequest(); err != nil {
		return zero, err
	}

	// Track half-open requests
	c.mu.Lock()
	isHalfOpen := c.state == CircuitHalfOpen
	if isHalfOpen {
		c.halfOpenRequests++
	}
	c.mu.Unlock()

	// Execute function
	result, err := fn(ctx)

	// Record result
	c.recordResult(err, isHalfOpen)

	return result, err
}

// allowRequest checks if a request is allowed.
func (c *Circuit) allowRequest() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case CircuitClosed:
		return nil

	case CircuitOpen:
		// Check if timeout has elapsed
		if time.Since(c.lastFailure) >= c.cfg.Timeout {
			c.transitionTo(CircuitHalfOpen)
			return nil
		}
		return ErrCircuitOpen

	case CircuitHalfOpen:
		// Limit concurrent requests
		if c.halfOpenRequests >= c.cfg.MaxHalfOpenRequests {
			return ErrCircuitOpen
		}
		return nil
	}

	return nil
}

// recordResult records the result of a request.
func (c *Circuit) recordResult(err error, wasHalfOpen bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if wasHalfOpen {
		c.halfOpenRequests--
	}

	// Check if this error counts as a failure
	isFailure := err != nil
	if c.cfg.IsFailure != nil && err != nil {
		isFailure = c.cfg.IsFailure(err)
	}

	switch c.state {
	case CircuitClosed:
		if isFailure {
			c.failures++
			c.lastFailure = time.Now()
			if c.failures >= c.cfg.FailureThreshold {
				c.transitionTo(CircuitOpen)
			}
		} else {
			// Reset failures on success
			c.failures = 0
		}

	case CircuitHalfOpen:
		if isFailure {
			// Failure in half-open returns to open
			c.lastFailure = time.Now()
			c.transitionTo(CircuitOpen)
		} else {
			c.successes++
			if c.successes >= c.cfg.SuccessThreshold {
				c.transitionTo(CircuitClosed)
			}
		}
	}
}

// transitionTo changes the circuit state.
func (c *Circuit) transitionTo(state CircuitState) {
	if c.state == state {
		return
	}

	oldState := c.state
	c.state = state

	// Reset counters on state change
	switch state {
	case CircuitClosed:
		c.failures = 0
		c.successes = 0
		c.halfOpenRequests = 0
	case CircuitOpen:
		c.successes = 0
		c.halfOpenRequests = 0
	case CircuitHalfOpen:
		c.successes = 0
	}

	// Notify callback
	if c.cfg.OnStateChange != nil {
		go c.cfg.OnStateChange(oldState, state)
	}
}

// State returns the current circuit state.
func (c *Circuit) State() CircuitState {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for timeout transition
	if c.state == CircuitOpen && time.Since(c.lastFailure) >= c.cfg.Timeout {
		c.transitionTo(CircuitHalfOpen)
	}

	return c.state
}

// Reset resets the circuit breaker to closed state.
func (c *Circuit) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transitionTo(CircuitClosed)
}

// Trip manually opens the circuit.
func (c *Circuit) Trip() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFailure = time.Now()
	c.transitionTo(CircuitOpen)
}

// Failures returns the current failure count.
func (c *Circuit) Failures() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures
}

// CircuitBreaker combines retry logic with circuit breaker.
type CircuitBreaker struct {
	circuit *Circuit
	retry   Config
}

// NewCircuitBreaker creates a new circuit breaker with retry support.
func NewCircuitBreaker(circuitCfg CircuitConfig, retryCfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		circuit: NewCircuit(circuitCfg),
		retry:   retryCfg,
	}
}

// Do executes a function with retry and circuit breaker.
func (cb *CircuitBreaker) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return cb.circuit.Do(ctx, func(ctx context.Context) error {
		return Do(ctx, cb.retry, fn)
	})
}

// CircuitBreakerDoWithResult executes a function with retry and circuit breaker.
func CircuitBreakerDoWithResult[T any](cb *CircuitBreaker, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	return CircuitDoWithResult(cb.circuit, ctx, func(ctx context.Context) (T, error) {
		return DoWithResult(ctx, cb.retry, fn)
	})
}

// State returns the circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	return cb.circuit.State()
}

// Reset resets the circuit breaker.
func (cb *CircuitBreaker) Reset() {
	cb.circuit.Reset()
}
