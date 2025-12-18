// jobs/jobs.go
package jobs

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Job represents a unit of work to be processed.
type Job struct {
	// ID is a unique identifier for the job.
	ID string

	// Type categorizes the job (e.g., "send_email", "process_upload").
	Type string

	// Payload contains the job data.
	Payload any

	// Priority determines processing order (higher = more urgent).
	// Default is 0.
	Priority int

	// MaxRetries is the maximum number of retry attempts.
	// Default is 3.
	MaxRetries int

	// RetryDelay is the base delay between retries (exponential backoff applied).
	// Default is 1 second.
	RetryDelay time.Duration

	// Timeout is the maximum time allowed for job execution.
	// Default is 30 seconds.
	Timeout time.Duration

	// Attempts tracks how many times the job has been attempted.
	Attempts int

	// CreatedAt is when the job was created.
	CreatedAt time.Time

	// Error holds the last error if the job failed.
	Error error
}

// Handler processes jobs of a specific type.
type Handler func(ctx context.Context, job *Job) error

// Runner manages background job processing.
type Runner struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	queue    chan *Job
	wg       sync.WaitGroup
	logger   *zap.Logger

	workers    int
	queueSize  int
	running    atomic.Bool
	stopCh     chan struct{}
	shutdownWg sync.WaitGroup

	// Hooks
	onStart   func(*Job)
	onSuccess func(*Job)
	onError   func(*Job, error)
	onRetry   func(*Job, error, int)
}

// Config configures the job runner.
type Config struct {
	// Workers is the number of concurrent workers. Default: 4.
	Workers int

	// QueueSize is the size of the job queue. Default: 100.
	QueueSize int

	// Logger for job processing. Default: no-op logger.
	Logger *zap.Logger

	// OnStart is called when a job starts processing.
	OnStart func(*Job)

	// OnSuccess is called when a job completes successfully.
	OnSuccess func(*Job)

	// OnError is called when a job fails permanently.
	OnError func(*Job, error)

	// OnRetry is called when a job is retried.
	OnRetry func(*Job, error, int)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Workers:   4,
		QueueSize: 100,
		Logger:    zap.NewNop(),
	}
}

// New creates a new job runner with the given configuration.
func New(cfg Config) *Runner {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	return &Runner{
		handlers:  make(map[string]Handler),
		queue:     make(chan *Job, cfg.QueueSize),
		logger:    cfg.Logger,
		workers:   cfg.Workers,
		queueSize: cfg.QueueSize,
		stopCh:    make(chan struct{}),
		onStart:   cfg.OnStart,
		onSuccess: cfg.OnSuccess,
		onError:   cfg.OnError,
		onRetry:   cfg.OnRetry,
	}
}

// Register adds a handler for a job type.
func (r *Runner) Register(jobType string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = handler
}

// Start begins processing jobs with the configured number of workers.
func (r *Runner) Start() {
	if r.running.Swap(true) {
		return // Already running
	}

	r.logger.Info("starting job runner", zap.Int("workers", r.workers))

	for i := 0; i < r.workers; i++ {
		r.shutdownWg.Add(1)
		go r.worker(i)
	}
}

// Stop gracefully stops the runner, waiting for in-flight jobs to complete.
func (r *Runner) Stop(ctx context.Context) error {
	if !r.running.Swap(false) {
		return nil // Not running
	}

	r.logger.Info("stopping job runner")
	close(r.stopCh)

	done := make(chan struct{})
	go func() {
		r.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("job runner stopped")
		return nil
	case <-ctx.Done():
		r.logger.Warn("job runner shutdown timed out")
		return ctx.Err()
	}
}

// Enqueue adds a job to the queue for processing.
// Returns false if the queue is full.
func (r *Runner) Enqueue(job *Job) bool {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	if job.RetryDelay == 0 {
		job.RetryDelay = time.Second
	}
	if job.Timeout == 0 {
		job.Timeout = 30 * time.Second
	}

	select {
	case r.queue <- job:
		r.logger.Debug("job enqueued",
			zap.String("id", job.ID),
			zap.String("type", job.Type),
		)
		return true
	default:
		r.logger.Warn("job queue full, dropping job",
			zap.String("id", job.ID),
			zap.String("type", job.Type),
		)
		return false
	}
}

// EnqueueFunc creates and enqueues a simple job.
func (r *Runner) EnqueueFunc(jobType string, payload any) bool {
	return r.Enqueue(&Job{
		Type:    jobType,
		Payload: payload,
	})
}

// MustEnqueue adds a job to the queue, blocking if full.
func (r *Runner) MustEnqueue(job *Job) {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	if job.RetryDelay == 0 {
		job.RetryDelay = time.Second
	}
	if job.Timeout == 0 {
		job.Timeout = 30 * time.Second
	}

	r.queue <- job
	r.logger.Debug("job enqueued",
		zap.String("id", job.ID),
		zap.String("type", job.Type),
	)
}

// QueueLen returns the current number of jobs in the queue.
func (r *Runner) QueueLen() int {
	return len(r.queue)
}

// worker processes jobs from the queue.
func (r *Runner) worker(id int) {
	defer r.shutdownWg.Done()

	r.logger.Debug("worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-r.stopCh:
			r.logger.Debug("worker stopping", zap.Int("worker_id", id))
			return
		case job := <-r.queue:
			r.process(job)
		}
	}
}

// process handles a single job with retries.
func (r *Runner) process(job *Job) {
	r.mu.RLock()
	handler, exists := r.handlers[job.Type]
	r.mu.RUnlock()

	if !exists {
		r.logger.Error("no handler for job type",
			zap.String("id", job.ID),
			zap.String("type", job.Type),
		)
		if r.onError != nil {
			r.onError(job, ErrNoHandler)
		}
		return
	}

	job.Attempts++

	if r.onStart != nil {
		r.onStart(job)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
	defer cancel()

	// Execute handler
	err := handler(ctx, job)

	if err == nil {
		r.logger.Debug("job completed",
			zap.String("id", job.ID),
			zap.String("type", job.Type),
			zap.Int("attempts", job.Attempts),
		)
		if r.onSuccess != nil {
			r.onSuccess(job)
		}
		return
	}

	job.Error = err

	// Check if we should retry
	if job.Attempts < job.MaxRetries {
		delay := r.calculateRetryDelay(job)

		r.logger.Warn("job failed, retrying",
			zap.String("id", job.ID),
			zap.String("type", job.Type),
			zap.Int("attempt", job.Attempts),
			zap.Int("max_retries", job.MaxRetries),
			zap.Duration("retry_delay", delay),
			zap.Error(err),
		)

		if r.onRetry != nil {
			r.onRetry(job, err, job.Attempts)
		}

		// Schedule retry
		time.AfterFunc(delay, func() {
			r.Enqueue(job)
		})
		return
	}

	// Permanent failure
	r.logger.Error("job failed permanently",
		zap.String("id", job.ID),
		zap.String("type", job.Type),
		zap.Int("attempts", job.Attempts),
		zap.Error(err),
	)

	if r.onError != nil {
		r.onError(job, err)
	}
}

// calculateRetryDelay applies exponential backoff.
func (r *Runner) calculateRetryDelay(job *Job) time.Duration {
	// Exponential backoff: delay * 2^(attempt-1)
	multiplier := 1 << (job.Attempts - 1)
	return job.RetryDelay * time.Duration(multiplier)
}

// Errors
var (
	ErrNoHandler = &jobError{code: "no_handler", message: "no handler registered for job type"}
	ErrQueueFull = &jobError{code: "queue_full", message: "job queue is full"}
)

type jobError struct {
	code    string
	message string
}

func (e *jobError) Error() string {
	return e.message
}

func (e *jobError) Code() string {
	return e.code
}
