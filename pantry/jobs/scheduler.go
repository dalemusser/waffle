// jobs/scheduler.go
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ScheduledJob represents a recurring job.
type ScheduledJob struct {
	// Name identifies this scheduled job.
	Name string

	// Interval between job runs.
	Interval time.Duration

	// Handler is the function to execute.
	Handler func(ctx context.Context) error

	// RunImmediately executes the job immediately on start.
	RunImmediately bool

	// Timeout for each execution. Default: 5 minutes.
	Timeout time.Duration
}

// Scheduler manages recurring jobs.
type Scheduler struct {
	mu       sync.RWMutex
	jobs     map[string]*scheduledEntry
	cronJobs map[string]*cronScheduledEntry
	logger   *zap.Logger
	running  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Optional integrations
	locker   Locker
	history  HistoryStore
	workerID string
}

type scheduledEntry struct {
	job    *ScheduledJob
	ticker *time.Ticker
	stopCh chan struct{}
}

type cronScheduledEntry struct {
	job     *CronJob
	nextRun time.Time
	stopCh  chan struct{}
}

// SchedulerOption configures the scheduler.
type SchedulerOption func(*Scheduler)

// NewScheduler creates a new scheduler.
func NewScheduler(logger *zap.Logger, opts ...SchedulerOption) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Scheduler{
		jobs:     make(map[string]*scheduledEntry),
		cronJobs: make(map[string]*cronScheduledEntry),
		logger:   logger,
		stopCh:   make(chan struct{}),
		workerID: generateWorkerID(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithLocker sets a distributed locker for the scheduler.
// When set, jobs will acquire a lock before executing to prevent
// duplicate execution across multiple instances.
func WithLocker(locker Locker) SchedulerOption {
	return func(s *Scheduler) {
		s.locker = locker
	}
}

// WithHistory sets a history store for the scheduler.
// When set, job executions will be recorded for monitoring.
func WithHistory(history HistoryStore) SchedulerOption {
	return func(s *Scheduler) {
		s.history = history
	}
}

// WithWorkerID sets a custom worker ID.
func WithWorkerID(id string) SchedulerOption {
	return func(s *Scheduler) {
		s.workerID = id
	}
}

// generateWorkerID generates a unique worker ID.
func generateWorkerID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Add registers a scheduled job.
func (s *Scheduler) Add(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job.Name == "" {
		return fmt.Errorf("jobs: job name is required")
	}

	if _, exists := s.jobs[job.Name]; exists {
		return fmt.Errorf("jobs: job %q already exists", job.Name)
	}

	if job.Timeout == 0 {
		job.Timeout = 5 * time.Minute
	}

	entry := &scheduledEntry{
		job:    job,
		stopCh: make(chan struct{}),
	}
	s.jobs[job.Name] = entry

	s.logger.Info("scheduled job added",
		zap.String("name", job.Name),
		zap.Duration("interval", job.Interval),
	)

	// If already running, start this job immediately
	if s.running {
		s.startJob(entry)
	}

	return nil
}

// AddCron registers a cron-scheduled job.
func (s *Scheduler) AddCron(job *CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job.Name == "" {
		return fmt.Errorf("jobs: job name is required")
	}

	if _, exists := s.cronJobs[job.Name]; exists {
		return fmt.Errorf("jobs: cron job %q already exists", job.Name)
	}

	// Parse cron expression
	parsed, err := ParseCron(job.Cron)
	if err != nil {
		return fmt.Errorf("jobs: invalid cron expression: %w", err)
	}
	job.parsed = parsed

	if job.Timeout == 0 {
		job.Timeout = 5 * time.Minute
	}

	if job.Location == nil {
		job.Location = time.Local
	}

	entry := &cronScheduledEntry{
		job:    job,
		stopCh: make(chan struct{}),
	}
	s.cronJobs[job.Name] = entry

	s.logger.Info("cron job added",
		zap.String("name", job.Name),
		zap.String("cron", job.Cron),
	)

	// If already running, start this job
	if s.running {
		s.startCronJob(entry)
	}

	return nil
}

// Cron is a convenience method to add a cron job.
func (s *Scheduler) Cron(expr, name string, handler func(ctx context.Context) error) error {
	return s.AddCron(&CronJob{
		Name:    name,
		Cron:    expr,
		Handler: handler,
	})
}

// Every is a convenience method to add a simple recurring job.
func (s *Scheduler) Every(interval time.Duration, name string, handler func(ctx context.Context) error) error {
	return s.Add(&ScheduledJob{
		Name:     name,
		Interval: interval,
		Handler:  handler,
	})
}

// Start begins executing all scheduled jobs.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})

	s.logger.Info("starting scheduler",
		zap.Int("interval_jobs", len(s.jobs)),
		zap.Int("cron_jobs", len(s.cronJobs)),
		zap.String("worker_id", s.workerID),
	)

	for _, entry := range s.jobs {
		s.startJob(entry)
	}

	for _, entry := range s.cronJobs {
		s.startCronJob(entry)
	}
}

// startJob begins execution of a single scheduled job.
func (s *Scheduler) startJob(entry *scheduledEntry) {
	entry.ticker = time.NewTicker(entry.job.Interval)
	entry.stopCh = make(chan struct{})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer entry.ticker.Stop()

		// Run immediately if configured
		if entry.job.RunImmediately {
			s.executeJob(entry.job.Name, entry.job.Handler, entry.job.Timeout)
		}

		for {
			select {
			case <-entry.stopCh:
				return
			case <-s.stopCh:
				return
			case <-entry.ticker.C:
				s.executeJob(entry.job.Name, entry.job.Handler, entry.job.Timeout)
			}
		}
	}()
}

// startCronJob begins execution of a cron-scheduled job.
func (s *Scheduler) startCronJob(entry *cronScheduledEntry) {
	entry.stopCh = make(chan struct{})

	// Calculate first run time
	now := time.Now()
	if entry.job.Location != nil {
		now = now.In(entry.job.Location)
	}
	entry.nextRun = entry.job.parsed.Next(now)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			// Calculate time until next run
			now := time.Now()
			if entry.job.Location != nil {
				now = now.In(entry.job.Location)
			}

			waitDuration := entry.nextRun.Sub(now)
			if waitDuration < 0 {
				// Missed execution, calculate next
				entry.nextRun = entry.job.parsed.Next(now)
				continue
			}

			timer := time.NewTimer(waitDuration)

			select {
			case <-entry.stopCh:
				timer.Stop()
				return
			case <-s.stopCh:
				timer.Stop()
				return
			case <-timer.C:
				s.executeJob(entry.job.Name, entry.job.Handler, entry.job.Timeout)

				// Calculate next run
				now = time.Now()
				if entry.job.Location != nil {
					now = now.In(entry.job.Location)
				}
				entry.nextRun = entry.job.parsed.Next(now)
			}
		}
	}()
}

// executeJob runs a job with optional locking and history recording.
func (s *Scheduler) executeJob(name string, handler func(ctx context.Context) error, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	execID := generateExecutionID()
	start := time.Now()

	// Record start if history is enabled
	if s.history != nil {
		exec := &JobExecution{
			ID:        execID,
			JobName:   name,
			Status:    JobStatusRunning,
			StartedAt: start,
			WorkerID:  s.workerID,
		}
		s.history.RecordStart(ctx, exec)
	}

	s.logger.Debug("executing scheduled job",
		zap.String("name", name),
		zap.String("execution_id", execID),
	)

	// Try to acquire lock if locker is configured
	if s.locker != nil {
		lockKey := "scheduler:" + name
		acquired, err := s.locker.Acquire(ctx, lockKey, timeout)
		if err != nil {
			s.logger.Error("failed to acquire lock",
				zap.String("name", name),
				zap.Error(err),
			)
			s.recordComplete(ctx, execID, name, start, JobStatusFailed, err)
			return
		}
		if !acquired {
			s.logger.Debug("skipping job, lock held by another instance",
				zap.String("name", name),
			)
			s.recordComplete(ctx, execID, name, start, JobStatusSkipped, nil)
			return
		}
		defer s.locker.Release(ctx, lockKey)
	}

	// Execute the job
	err := handler(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("scheduled job failed",
			zap.String("name", name),
			zap.String("execution_id", execID),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		s.recordComplete(ctx, execID, name, start, JobStatusFailed, err)
		return
	}

	s.logger.Debug("scheduled job completed",
		zap.String("name", name),
		zap.String("execution_id", execID),
		zap.Duration("duration", duration),
	)
	s.recordComplete(ctx, execID, name, start, JobStatusCompleted, nil)
}

// recordComplete records job completion to history.
func (s *Scheduler) recordComplete(ctx context.Context, execID, name string, start time.Time, status JobStatus, err error) {
	if s.history == nil {
		return
	}

	now := time.Now()
	exec := &JobExecution{
		ID:          execID,
		JobName:     name,
		Status:      status,
		StartedAt:   start,
		CompletedAt: &now,
		Duration:    now.Sub(start),
		WorkerID:    s.workerID,
	}
	if err != nil {
		exec.Error = err.Error()
	}

	s.history.RecordComplete(ctx, exec)
}

// generateExecutionID generates a unique execution ID.
func generateExecutionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("exec_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// Stop gracefully stops all scheduled jobs.
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.logger.Info("stopping scheduler")

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("scheduler stopped")
		return nil
	case <-ctx.Done():
		s.logger.Warn("scheduler shutdown timed out")
		return ctx.Err()
	}
}

// Remove stops and removes a scheduled job by name.
func (s *Scheduler) Remove(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check interval jobs
	if entry, exists := s.jobs[name]; exists {
		close(entry.stopCh)
		delete(s.jobs, name)
		s.logger.Info("scheduled job removed", zap.String("name", name))
		return true
	}

	// Check cron jobs
	if entry, exists := s.cronJobs[name]; exists {
		close(entry.stopCh)
		delete(s.cronJobs, name)
		s.logger.Info("cron job removed", zap.String("name", name))
		return true
	}

	return false
}

// Get returns a job by name.
func (s *Scheduler) Get(name string) (*JobInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check interval jobs
	if entry, exists := s.jobs[name]; exists {
		return &JobInfo{
			Name:           entry.job.Name,
			Type:           "interval",
			Interval:       entry.job.Interval,
			Timeout:        entry.job.Timeout,
			RunImmediately: entry.job.RunImmediately,
		}, true
	}

	// Check cron jobs
	if entry, exists := s.cronJobs[name]; exists {
		return &JobInfo{
			Name:     entry.job.Name,
			Type:     "cron",
			Cron:     entry.job.Cron,
			Timeout:  entry.job.Timeout,
			NextRun:  entry.nextRun,
			Location: entry.job.Location,
		}, true
	}

	return nil, false
}

// List returns the names of all scheduled jobs.
func (s *Scheduler) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.jobs)+len(s.cronJobs))
	for name := range s.jobs {
		names = append(names, name)
	}
	for name := range s.cronJobs {
		names = append(names, name)
	}
	return names
}

// ListJobs returns information about all scheduled jobs.
func (s *Scheduler) ListJobs() []*JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*JobInfo, 0, len(s.jobs)+len(s.cronJobs))

	for _, entry := range s.jobs {
		jobs = append(jobs, &JobInfo{
			Name:           entry.job.Name,
			Type:           "interval",
			Interval:       entry.job.Interval,
			Timeout:        entry.job.Timeout,
			RunImmediately: entry.job.RunImmediately,
		})
	}

	for _, entry := range s.cronJobs {
		jobs = append(jobs, &JobInfo{
			Name:     entry.job.Name,
			Type:     "cron",
			Cron:     entry.job.Cron,
			Timeout:  entry.job.Timeout,
			NextRun:  entry.nextRun,
			Location: entry.job.Location,
		})
	}

	return jobs
}

// RunNow immediately executes a job by name.
func (s *Scheduler) RunNow(ctx context.Context, name string) error {
	s.mu.RLock()

	// Check interval jobs
	if entry, exists := s.jobs[name]; exists {
		s.mu.RUnlock()
		s.executeJob(entry.job.Name, entry.job.Handler, entry.job.Timeout)
		return nil
	}

	// Check cron jobs
	if entry, exists := s.cronJobs[name]; exists {
		s.mu.RUnlock()
		s.executeJob(entry.job.Name, entry.job.Handler, entry.job.Timeout)
		return nil
	}

	s.mu.RUnlock()
	return fmt.Errorf("jobs: job %q not found", name)
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// WorkerID returns this scheduler's worker ID.
func (s *Scheduler) WorkerID() string {
	return s.workerID
}

// JobInfo contains information about a scheduled job.
type JobInfo struct {
	Name           string
	Type           string // "interval" or "cron"
	Interval       time.Duration
	Cron           string
	Timeout        time.Duration
	RunImmediately bool
	NextRun        time.Time
	Location       *time.Location
}

// History returns the history store if configured.
func (s *Scheduler) History() HistoryStore {
	return s.history
}

// Locker returns the locker if configured.
func (s *Scheduler) Locker() Locker {
	return s.locker
}
