// retry/retry.go
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config configures retry behavior.
type Config struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	// Default: 3.
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	// Default: 100ms.
	InitialDelay time.Duration

	// MaxDelay caps the delay between retries.
	// Default: 30 seconds.
	MaxDelay time.Duration

	// Multiplier increases the delay after each retry.
	// Default: 2.0 (exponential backoff).
	Multiplier float64

	// Jitter adds randomness to delays (0.0 to 1.0).
	// Default: 0.1 (10% jitter).
	Jitter float64

	// RetryIf determines whether to retry based on the error.
	// Default: retry all non-nil errors.
	RetryIf func(error) bool

	// OnRetry is called before each retry attempt.
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultConfig returns sensible retry defaults.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// Do executes a function with retries using the given configuration.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	cfg = withDefaults(cfg)

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context before attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if cfg.RetryIf != nil && !cfg.RetryIf(err) {
			return err
		}

		// Don't wait after the last attempt
		if attempt >= cfg.MaxAttempts {
			break
		}

		// Calculate delay with jitter
		actualDelay := addJitter(delay, cfg.Jitter)

		// Notify callback
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, actualDelay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(actualDelay):
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return lastErr
}

// DoWithResult executes a function that returns a value with retries.
func DoWithResult[T any](ctx context.Context, cfg Config, fn func(ctx context.Context) (T, error)) (T, error) {
	cfg = withDefaults(cfg)

	var lastErr error
	var zero T
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context before attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return zero, lastErr
			}
			return zero, ctx.Err()
		}

		// Execute the function
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if cfg.RetryIf != nil && !cfg.RetryIf(err) {
			return zero, err
		}

		// Don't wait after the last attempt
		if attempt >= cfg.MaxAttempts {
			break
		}

		// Calculate delay with jitter
		actualDelay := addJitter(delay, cfg.Jitter)

		// Notify callback
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, actualDelay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return zero, lastErr
		case <-time.After(actualDelay):
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return zero, lastErr
}

// Simple executes a function with default retry settings.
func Simple(ctx context.Context, fn func(ctx context.Context) error) error {
	return Do(ctx, DefaultConfig(), fn)
}

// WithAttempts executes a function with the specified number of attempts.
func WithAttempts(ctx context.Context, attempts int, fn func(ctx context.Context) error) error {
	cfg := DefaultConfig()
	cfg.MaxAttempts = attempts
	return Do(ctx, cfg, fn)
}

// withDefaults applies default values to config.
func withDefaults(cfg Config) Config {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 100 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	return cfg
}

// addJitter adds randomness to a duration.
func addJitter(d time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return d
	}
	// Add +/- jitter%
	delta := float64(d) * jitter
	min := float64(d) - delta
	max := float64(d) + delta
	return time.Duration(min + rand.Float64()*(max-min))
}

// Permanent wraps an error to indicate it should not be retried.
type Permanent struct {
	Err error
}

func (p *Permanent) Error() string {
	if p.Err == nil {
		return "permanent error"
	}
	return p.Err.Error()
}

func (p *Permanent) Unwrap() error {
	return p.Err
}

// PermanentError wraps an error to prevent retries.
func PermanentError(err error) error {
	if err == nil {
		return nil
	}
	return &Permanent{Err: err}
}

// IsPermanent returns true if the error is marked as permanent.
func IsPermanent(err error) bool {
	var p *Permanent
	return errors.As(err, &p)
}

// SkipPermanent returns a RetryIf function that skips permanent errors.
func SkipPermanent(err error) bool {
	return !IsPermanent(err)
}

// Backoff strategies.

// ExponentialBackoff returns a config with exponential backoff.
func ExponentialBackoff(initial, max time.Duration, multiplier float64) Config {
	return Config{
		MaxAttempts:  5,
		InitialDelay: initial,
		MaxDelay:     max,
		Multiplier:   multiplier,
		Jitter:       0.1,
	}
}

// ConstantBackoff returns a config with constant delay between retries.
func ConstantBackoff(delay time.Duration, attempts int) Config {
	return Config{
		MaxAttempts:  attempts,
		InitialDelay: delay,
		MaxDelay:     delay,
		Multiplier:   1.0,
		Jitter:       0,
	}
}

// LinearBackoff returns a config with linearly increasing delays.
func LinearBackoff(initial, increment time.Duration, attempts int) Config {
	// Approximate linear with low multiplier
	return Config{
		MaxAttempts:  attempts,
		InitialDelay: initial,
		MaxDelay:     initial + increment*time.Duration(attempts),
		Multiplier:   1.0 + float64(increment)/float64(initial),
		Jitter:       0.1,
	}
}

// Fibonacci returns delays following the Fibonacci sequence.
func FibonacciBackoff(initial time.Duration, attempts int) Config {
	// Golden ratio approximates Fibonacci growth
	return Config{
		MaxAttempts:  attempts,
		InitialDelay: initial,
		MaxDelay:     initial * time.Duration(math.Pow(1.618, float64(attempts))),
		Multiplier:   1.618,
		Jitter:       0.1,
	}
}
