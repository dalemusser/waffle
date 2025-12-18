package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// JobStatus represents the status of a job execution.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped" // Skipped due to lock contention
	JobStatusCancelled JobStatus = "cancelled"
)

// JobExecution represents a single execution of a job.
type JobExecution struct {
	// ID is the unique identifier for this execution.
	ID string `json:"id"`

	// JobName is the name of the job.
	JobName string `json:"job_name"`

	// Status is the current status.
	Status JobStatus `json:"status"`

	// StartedAt is when the execution started.
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the execution completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Duration is how long the execution took.
	Duration time.Duration `json:"duration,omitempty"`

	// Error is the error message if the job failed.
	Error string `json:"error,omitempty"`

	// WorkerID identifies which worker ran this job.
	WorkerID string `json:"worker_id,omitempty"`

	// Metadata contains additional execution metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// JobStats contains statistics for a job.
type JobStats struct {
	// JobName is the name of the job.
	JobName string `json:"job_name"`

	// TotalExecutions is the total number of executions.
	TotalExecutions int64 `json:"total_executions"`

	// SuccessfulExecutions is the number of successful executions.
	SuccessfulExecutions int64 `json:"successful_executions"`

	// FailedExecutions is the number of failed executions.
	FailedExecutions int64 `json:"failed_executions"`

	// SkippedExecutions is the number of skipped executions.
	SkippedExecutions int64 `json:"skipped_executions"`

	// LastExecution is the last execution time.
	LastExecution *time.Time `json:"last_execution,omitempty"`

	// LastSuccess is the last successful execution time.
	LastSuccess *time.Time `json:"last_success,omitempty"`

	// LastFailure is the last failed execution time.
	LastFailure *time.Time `json:"last_failure,omitempty"`

	// LastError is the last error message.
	LastError string `json:"last_error,omitempty"`

	// AverageDuration is the average execution duration.
	AverageDuration time.Duration `json:"average_duration"`

	// MinDuration is the minimum execution duration.
	MinDuration time.Duration `json:"min_duration"`

	// MaxDuration is the maximum execution duration.
	MaxDuration time.Duration `json:"max_duration"`
}

// HistoryStore is the interface for storing job execution history.
type HistoryStore interface {
	// RecordStart records the start of a job execution.
	RecordStart(ctx context.Context, exec *JobExecution) error

	// RecordComplete records the completion of a job execution.
	RecordComplete(ctx context.Context, exec *JobExecution) error

	// GetExecutions returns recent executions for a job.
	GetExecutions(ctx context.Context, jobName string, limit int) ([]*JobExecution, error)

	// GetExecution returns a specific execution by ID.
	GetExecution(ctx context.Context, id string) (*JobExecution, error)

	// GetStats returns statistics for a job.
	GetStats(ctx context.Context, jobName string) (*JobStats, error)

	// GetAllStats returns statistics for all jobs.
	GetAllStats(ctx context.Context) (map[string]*JobStats, error)

	// Cleanup removes old executions.
	Cleanup(ctx context.Context, maxAge time.Duration) error
}

// MemoryHistoryStore stores job history in memory.
type MemoryHistoryStore struct {
	mu         sync.RWMutex
	executions map[string][]*JobExecution // jobName -> executions
	byID       map[string]*JobExecution   // id -> execution
	stats      map[string]*JobStats       // jobName -> stats
	maxPerJob  int
}

// NewMemoryHistoryStore creates a new in-memory history store.
func NewMemoryHistoryStore(maxPerJob int) *MemoryHistoryStore {
	if maxPerJob <= 0 {
		maxPerJob = 100
	}
	return &MemoryHistoryStore{
		executions: make(map[string][]*JobExecution),
		byID:       make(map[string]*JobExecution),
		stats:      make(map[string]*JobStats),
		maxPerJob:  maxPerJob,
	}
}

// RecordStart records the start of a job execution.
func (s *MemoryHistoryStore) RecordStart(ctx context.Context, exec *JobExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store execution
	s.byID[exec.ID] = exec
	s.executions[exec.JobName] = append(s.executions[exec.JobName], exec)

	// Trim if too many
	if len(s.executions[exec.JobName]) > s.maxPerJob {
		old := s.executions[exec.JobName][0]
		delete(s.byID, old.ID)
		s.executions[exec.JobName] = s.executions[exec.JobName][1:]
	}

	// Update stats
	stats := s.getOrCreateStats(exec.JobName)
	stats.TotalExecutions++
	stats.LastExecution = &exec.StartedAt

	return nil
}

// RecordComplete records the completion of a job execution.
func (s *MemoryHistoryStore) RecordComplete(ctx context.Context, exec *JobExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update stored execution
	if stored, ok := s.byID[exec.ID]; ok {
		stored.Status = exec.Status
		stored.CompletedAt = exec.CompletedAt
		stored.Duration = exec.Duration
		stored.Error = exec.Error
	}

	// Update stats
	stats := s.getOrCreateStats(exec.JobName)

	switch exec.Status {
	case JobStatusCompleted:
		stats.SuccessfulExecutions++
		stats.LastSuccess = exec.CompletedAt
		s.updateDurationStats(stats, exec.Duration)
	case JobStatusFailed:
		stats.FailedExecutions++
		stats.LastFailure = exec.CompletedAt
		stats.LastError = exec.Error
		s.updateDurationStats(stats, exec.Duration)
	case JobStatusSkipped:
		stats.SkippedExecutions++
	}

	return nil
}

// updateDurationStats updates duration statistics.
func (s *MemoryHistoryStore) updateDurationStats(stats *JobStats, duration time.Duration) {
	if stats.MinDuration == 0 || duration < stats.MinDuration {
		stats.MinDuration = duration
	}
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	// Calculate running average
	total := stats.SuccessfulExecutions + stats.FailedExecutions
	if total > 0 {
		oldAvg := stats.AverageDuration
		stats.AverageDuration = oldAvg + (duration-oldAvg)/time.Duration(total)
	}
}

// getOrCreateStats gets or creates stats for a job.
func (s *MemoryHistoryStore) getOrCreateStats(jobName string) *JobStats {
	stats, ok := s.stats[jobName]
	if !ok {
		stats = &JobStats{JobName: jobName}
		s.stats[jobName] = stats
	}
	return stats
}

// GetExecutions returns recent executions for a job.
func (s *MemoryHistoryStore) GetExecutions(ctx context.Context, jobName string, limit int) ([]*JobExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	execs := s.executions[jobName]
	if len(execs) == 0 {
		return nil, nil
	}

	// Return most recent first
	result := make([]*JobExecution, 0, min(limit, len(execs)))
	for i := len(execs) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, execs[i])
	}

	return result, nil
}

// GetExecution returns a specific execution by ID.
func (s *MemoryHistoryStore) GetExecution(ctx context.Context, id string) (*JobExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.byID[id], nil
}

// GetStats returns statistics for a job.
func (s *MemoryHistoryStore) GetStats(ctx context.Context, jobName string) (*JobStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats[jobName]
	if stats == nil {
		return &JobStats{JobName: jobName}, nil
	}

	// Return a copy
	copy := *stats
	return &copy, nil
}

// GetAllStats returns statistics for all jobs.
func (s *MemoryHistoryStore) GetAllStats(ctx context.Context) (map[string]*JobStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*JobStats, len(s.stats))
	for name, stats := range s.stats {
		copy := *stats
		result[name] = &copy
	}

	return result, nil
}

// Cleanup removes old executions.
func (s *MemoryHistoryStore) Cleanup(ctx context.Context, maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for jobName, execs := range s.executions {
		var kept []*JobExecution
		for _, exec := range execs {
			if exec.StartedAt.After(cutoff) {
				kept = append(kept, exec)
			} else {
				delete(s.byID, exec.ID)
			}
		}
		s.executions[jobName] = kept
	}

	return nil
}

// RedisHistoryStore stores job history in Redis.
type RedisHistoryStore struct {
	client    RedisHistoryClient
	prefix    string
	maxPerJob int
	ttl       time.Duration
}

// RedisHistoryClient is the interface for Redis operations needed by the history store.
type RedisHistoryClient interface {
	// LPush pushes a value to the left of a list.
	LPush(ctx context.Context, key string, value string) error

	// LRange returns a range of elements from a list.
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// LTrim trims a list to the specified range.
	LTrim(ctx context.Context, key string, start, stop int64) error

	// Set sets a key with TTL.
	Set(ctx context.Context, key, value string, ttl time.Duration) error

	// Get gets a value by key.
	Get(ctx context.Context, key string) (string, error)

	// HSet sets a hash field.
	HSet(ctx context.Context, key, field, value string) error

	// HGetAll gets all fields and values in a hash.
	HGetAll(ctx context.Context, key string) (map[string]string, error)

	// HIncrBy increments a hash field.
	HIncrBy(ctx context.Context, key, field string, incr int64) error

	// Expire sets a TTL on a key.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Keys returns keys matching a pattern.
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// RedisHistoryStoreConfig configures the Redis history store.
type RedisHistoryStoreConfig struct {
	// Client is the Redis client.
	Client RedisHistoryClient

	// Prefix is the key prefix.
	// Default: "job_history:"
	Prefix string

	// MaxPerJob is the maximum executions to keep per job.
	// Default: 100
	MaxPerJob int

	// TTL is how long to keep execution records.
	// Default: 7 days
	TTL time.Duration
}

// NewRedisHistoryStore creates a new Redis-based history store.
func NewRedisHistoryStore(cfg RedisHistoryStoreConfig) *RedisHistoryStore {
	if cfg.Prefix == "" {
		cfg.Prefix = "job_history:"
	}
	if cfg.MaxPerJob <= 0 {
		cfg.MaxPerJob = 100
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 7 * 24 * time.Hour
	}

	return &RedisHistoryStore{
		client:    cfg.Client,
		prefix:    cfg.Prefix,
		maxPerJob: cfg.MaxPerJob,
		ttl:       cfg.TTL,
	}
}

// RecordStart records the start of a job execution.
func (s *RedisHistoryStore) RecordStart(ctx context.Context, exec *JobExecution) error {
	// Store execution details
	data, err := json.Marshal(exec)
	if err != nil {
		return err
	}

	execKey := s.prefix + "exec:" + exec.ID
	if err := s.client.Set(ctx, execKey, string(data), s.ttl); err != nil {
		return err
	}

	// Add to job's execution list
	listKey := s.prefix + "list:" + exec.JobName
	if err := s.client.LPush(ctx, listKey, exec.ID); err != nil {
		return err
	}

	// Trim list
	if err := s.client.LTrim(ctx, listKey, 0, int64(s.maxPerJob-1)); err != nil {
		return err
	}

	// Update stats
	statsKey := s.prefix + "stats:" + exec.JobName
	s.client.HIncrBy(ctx, statsKey, "total", 1)
	s.client.HSet(ctx, statsKey, "last_execution", exec.StartedAt.Format(time.RFC3339))
	s.client.Expire(ctx, statsKey, s.ttl)

	return nil
}

// RecordComplete records the completion of a job execution.
func (s *RedisHistoryStore) RecordComplete(ctx context.Context, exec *JobExecution) error {
	// Update execution details
	data, err := json.Marshal(exec)
	if err != nil {
		return err
	}

	execKey := s.prefix + "exec:" + exec.ID
	if err := s.client.Set(ctx, execKey, string(data), s.ttl); err != nil {
		return err
	}

	// Update stats
	statsKey := s.prefix + "stats:" + exec.JobName

	switch exec.Status {
	case JobStatusCompleted:
		s.client.HIncrBy(ctx, statsKey, "successful", 1)
		if exec.CompletedAt != nil {
			s.client.HSet(ctx, statsKey, "last_success", exec.CompletedAt.Format(time.RFC3339))
		}
	case JobStatusFailed:
		s.client.HIncrBy(ctx, statsKey, "failed", 1)
		if exec.CompletedAt != nil {
			s.client.HSet(ctx, statsKey, "last_failure", exec.CompletedAt.Format(time.RFC3339))
		}
		if exec.Error != "" {
			s.client.HSet(ctx, statsKey, "last_error", exec.Error)
		}
	case JobStatusSkipped:
		s.client.HIncrBy(ctx, statsKey, "skipped", 1)
	}

	return nil
}

// GetExecutions returns recent executions for a job.
func (s *RedisHistoryStore) GetExecutions(ctx context.Context, jobName string, limit int) ([]*JobExecution, error) {
	listKey := s.prefix + "list:" + jobName

	ids, err := s.client.LRange(ctx, listKey, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}

	var result []*JobExecution
	for _, id := range ids {
		exec, err := s.GetExecution(ctx, id)
		if err != nil {
			continue
		}
		if exec != nil {
			result = append(result, exec)
		}
	}

	return result, nil
}

// GetExecution returns a specific execution by ID.
func (s *RedisHistoryStore) GetExecution(ctx context.Context, id string) (*JobExecution, error) {
	execKey := s.prefix + "exec:" + id

	data, err := s.client.Get(ctx, execKey)
	if err != nil {
		return nil, nil // Not found
	}

	var exec JobExecution
	if err := json.Unmarshal([]byte(data), &exec); err != nil {
		return nil, err
	}

	return &exec, nil
}

// GetStats returns statistics for a job.
func (s *RedisHistoryStore) GetStats(ctx context.Context, jobName string) (*JobStats, error) {
	statsKey := s.prefix + "stats:" + jobName

	data, err := s.client.HGetAll(ctx, statsKey)
	if err != nil {
		return &JobStats{JobName: jobName}, nil
	}

	stats := &JobStats{JobName: jobName}

	if v, ok := data["total"]; ok {
		stats.TotalExecutions, _ = parseInt64(v)
	}
	if v, ok := data["successful"]; ok {
		stats.SuccessfulExecutions, _ = parseInt64(v)
	}
	if v, ok := data["failed"]; ok {
		stats.FailedExecutions, _ = parseInt64(v)
	}
	if v, ok := data["skipped"]; ok {
		stats.SkippedExecutions, _ = parseInt64(v)
	}
	if v, ok := data["last_execution"]; ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			stats.LastExecution = &t
		}
	}
	if v, ok := data["last_success"]; ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			stats.LastSuccess = &t
		}
	}
	if v, ok := data["last_failure"]; ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			stats.LastFailure = &t
		}
	}
	if v, ok := data["last_error"]; ok {
		stats.LastError = v
	}

	return stats, nil
}

// GetAllStats returns statistics for all jobs.
func (s *RedisHistoryStore) GetAllStats(ctx context.Context) (map[string]*JobStats, error) {
	pattern := s.prefix + "stats:*"
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*JobStats)
	prefixLen := len(s.prefix + "stats:")

	for _, key := range keys {
		jobName := key[prefixLen:]
		stats, err := s.GetStats(ctx, jobName)
		if err == nil && stats != nil {
			result[jobName] = stats
		}
	}

	return result, nil
}

// Cleanup removes old executions.
func (s *RedisHistoryStore) Cleanup(ctx context.Context, maxAge time.Duration) error {
	// Redis TTL handles cleanup automatically
	return nil
}

func parseInt64(s string) (int64, error) {
	var result int64
	err := json.Unmarshal([]byte(s), &result)
	return result, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Monitor provides a simple interface for monitoring job health.
type Monitor struct {
	store HistoryStore
}

// NewMonitor creates a new monitor.
func NewMonitor(store HistoryStore) *Monitor {
	return &Monitor{store: store}
}

// HealthCheck returns the health status of all jobs.
func (m *Monitor) HealthCheck(ctx context.Context) (map[string]bool, error) {
	allStats, err := m.store.GetAllStats(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for name, stats := range allStats {
		// Job is healthy if last execution was successful
		healthy := stats.LastSuccess != nil &&
			(stats.LastFailure == nil || stats.LastSuccess.After(*stats.LastFailure))
		result[name] = healthy
	}

	return result, nil
}

// FailureRate returns the failure rate for a job (0.0 to 1.0).
func (m *Monitor) FailureRate(ctx context.Context, jobName string) (float64, error) {
	stats, err := m.store.GetStats(ctx, jobName)
	if err != nil {
		return 0, err
	}

	total := stats.SuccessfulExecutions + stats.FailedExecutions
	if total == 0 {
		return 0, nil
	}

	return float64(stats.FailedExecutions) / float64(total), nil
}

// AverageLatency returns the average execution duration for a job.
func (m *Monitor) AverageLatency(ctx context.Context, jobName string) (time.Duration, error) {
	stats, err := m.store.GetStats(ctx, jobName)
	if err != nil {
		return 0, err
	}

	return stats.AverageDuration, nil
}
