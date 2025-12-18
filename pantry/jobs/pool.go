// jobs/pool.go
package jobs

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Pool provides a simple worker pool for async tasks.
// Unlike Runner, Pool doesn't queue jobs - it runs them immediately
// with bounded concurrency.
type Pool struct {
	sem     chan struct{}
	wg      sync.WaitGroup
	logger  *zap.Logger
	running atomic.Int64
}

// NewPool creates a worker pool with the specified concurrency limit.
func NewPool(workers int, logger *zap.Logger) *Pool {
	if workers <= 0 {
		workers = 4
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Pool{
		sem:    make(chan struct{}, workers),
		logger: logger,
	}
}

// Go runs a task asynchronously, blocking if the pool is at capacity.
func (p *Pool) Go(task func()) {
	p.sem <- struct{}{} // Acquire
	p.wg.Add(1)
	p.running.Add(1)

	go func() {
		defer func() {
			<-p.sem // Release
			p.wg.Done()
			p.running.Add(-1)

			if r := recover(); r != nil {
				p.logger.Error("task panicked", zap.Any("panic", r))
			}
		}()
		task()
	}()
}

// GoWithContext runs a task with context support.
func (p *Pool) GoWithContext(ctx context.Context, task func(ctx context.Context)) {
	p.Go(func() {
		task(ctx)
	})
}

// TryGo attempts to run a task immediately.
// Returns false if the pool is at capacity.
func (p *Pool) TryGo(task func()) bool {
	select {
	case p.sem <- struct{}{}:
		p.wg.Add(1)
		p.running.Add(1)

		go func() {
			defer func() {
				<-p.sem
				p.wg.Done()
				p.running.Add(-1)

				if r := recover(); r != nil {
					p.logger.Error("task panicked", zap.Any("panic", r))
				}
			}()
			task()
		}()
		return true
	default:
		return false
	}
}

// Wait blocks until all running tasks complete.
func (p *Pool) Wait() {
	p.wg.Wait()
}

// WaitWithTimeout waits for tasks with a timeout.
// Returns true if all tasks completed, false if timeout.
func (p *Pool) WaitWithTimeout(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Running returns the number of currently executing tasks.
func (p *Pool) Running() int {
	return int(p.running.Load())
}

// Submit runs a task and returns a Future to wait for its result.
func Submit[T any](p *Pool, task func() (T, error)) *Future[T] {
	f := &Future[T]{
		done: make(chan struct{}),
	}

	p.Go(func() {
		f.value, f.err = task()
		close(f.done)
	})

	return f
}

// Future represents the result of an async task.
type Future[T any] struct {
	value T
	err   error
	done  chan struct{}
}

// Wait blocks until the task completes and returns the result.
func (f *Future[T]) Wait() (T, error) {
	<-f.done
	return f.value, f.err
}

// WaitWithTimeout waits with a timeout.
// Returns the zero value and context.DeadlineExceeded if timeout.
func (f *Future[T]) WaitWithTimeout(timeout time.Duration) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	case <-time.After(timeout):
		var zero T
		return zero, context.DeadlineExceeded
	}
}

// Done returns a channel that is closed when the task completes.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// Ready returns true if the task has completed.
func (f *Future[T]) Ready() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}
