// timeout/context.go
package timeout

import (
	"context"
	"time"
)

// FromContext returns the deadline from the context, if any.
// Returns zero time and false if no deadline is set.
func FromContext(ctx context.Context) (time.Time, bool) {
	return ctx.Deadline()
}

// Remaining returns the time remaining until the context deadline.
// Returns 0 if no deadline is set or if the deadline has passed.
func Remaining(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	remaining := time.Until(deadline)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasDeadline returns true if the context has a deadline set.
func HasDeadline(ctx context.Context) bool {
	_, ok := ctx.Deadline()
	return ok
}

// IsExpired returns true if the context deadline has passed.
func IsExpired(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// WithShorter creates a context with a deadline that is the shorter of
// the existing deadline (if any) and the given duration.
func WithShorter(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	newDeadline := time.Now().Add(d)

	if ok && deadline.Before(newDeadline) {
		// Existing deadline is sooner, use it
		return context.WithDeadline(ctx, deadline)
	}

	return context.WithTimeout(ctx, d)
}

// WithExtended creates a context with a new deadline extended from now.
// Note: This creates a new context that may outlive the parent's deadline.
// Use carefully - the parent context's cancellation will still propagate.
func WithExtended(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

// ShrinkBy reduces the remaining timeout by the given duration.
// Useful for reserving time for cleanup operations.
func ShrinkBy(ctx context.Context, reserve time.Duration) (context.Context, context.CancelFunc) {
	remaining := Remaining(ctx)
	if remaining <= reserve {
		// Already expired or not enough time
		ctx, cancel := context.WithCancel(ctx)
		cancel() // Immediately cancel
		return ctx, cancel
	}
	return context.WithTimeout(ctx, remaining-reserve)
}

// PropagateTimeout creates a new context that inherits the parent's deadline
// but is separately cancellable. Useful for spawning child operations.
func PropagateTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	deadline, ok := parent.Deadline()
	if !ok {
		return context.WithCancel(parent)
	}
	return context.WithDeadline(parent, deadline)
}

// Run executes a function with the given timeout.
// Returns context.DeadlineExceeded if the function doesn't complete in time.
func Run(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunWithResult executes a function with the given timeout and returns its result.
func RunWithResult[T any](ctx context.Context, timeout time.Duration, fn func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}

	done := make(chan result, 1)
	go func() {
		v, err := fn(ctx)
		done <- result{value: v, err: err}
	}()

	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}
