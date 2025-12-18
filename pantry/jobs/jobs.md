# jobs

Background job processing for WAFFLE applications.

## Overview

The `jobs` package provides three components for background work:
- **Runner** — Queue-based job processing with retries and handlers
- **Scheduler** — Recurring jobs with intervals and cron expressions
- **Pool** — Simple worker pool for one-off async tasks

Plus advanced features:
- **Cron Expressions** — Standard cron scheduling ("0 0 * * *")
- **Distributed Locking** — Redis-based locking for multi-instance deployments
- **Job History** — Execution history and monitoring

## Import

```go
import "github.com/dalemusser/waffle/pantry/jobs"
```

---

## Runner

Queue-based job processing with typed handlers, retries, and exponential backoff.

### New

**Location:** `jobs.go`

```go
func New(cfg Config) *Runner
```

Creates a new job runner.

**Config:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Workers | int | 4 | Concurrent workers |
| QueueSize | int | 100 | Job queue capacity |
| Logger | *zap.Logger | no-op | Logger for job events |
| OnStart | func(*Job) | nil | Called when job starts |
| OnSuccess | func(*Job) | nil | Called on success |
| OnError | func(*Job, error) | nil | Called on permanent failure |
| OnRetry | func(*Job, error, int) | nil | Called on retry |

### Basic Usage

```go
// Create runner
runner := jobs.New(jobs.Config{
    Workers:   4,
    QueueSize: 100,
    Logger:    logger,
})

// Register handlers
runner.Register("send_email", func(ctx context.Context, job *jobs.Job) error {
    payload := job.Payload.(EmailPayload)
    return emailService.Send(ctx, payload.To, payload.Subject, payload.Body)
})

runner.Register("process_upload", func(ctx context.Context, job *jobs.Job) error {
    payload := job.Payload.(UploadPayload)
    return processFile(ctx, payload.Path)
})

// Start workers
runner.Start()

// Enqueue jobs
runner.Enqueue(&jobs.Job{
    ID:      "email-123",
    Type:    "send_email",
    Payload: EmailPayload{To: "user@example.com", Subject: "Welcome!"},
})

// Or use the simple form
runner.EnqueueFunc("send_email", EmailPayload{To: "user@example.com"})

// Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
runner.Stop(ctx)
```

### Job

```go
type Job struct {
    ID         string        // Unique identifier
    Type       string        // Job type (matches handler)
    Payload    any           // Job data
    Priority   int           // Higher = more urgent (default: 0)
    MaxRetries int           // Retry attempts (default: 3)
    RetryDelay time.Duration // Base retry delay (default: 1s)
    Timeout    time.Duration // Execution timeout (default: 30s)
    Attempts   int           // Current attempt count
    CreatedAt  time.Time     // Creation timestamp
    Error      error         // Last error (if failed)
}
```

### Runner Methods

```go
runner.Register(jobType string, handler Handler)  // Register handler
runner.Start()                                     // Start workers
runner.Stop(ctx context.Context) error            // Graceful shutdown
runner.Enqueue(job *Job) bool                     // Enqueue (returns false if full)
runner.EnqueueFunc(jobType string, payload any) bool // Simple enqueue
runner.MustEnqueue(job *Job)                      // Enqueue (blocks if full)
runner.QueueLen() int                             // Current queue length
```

### Retries and Backoff

Jobs automatically retry with exponential backoff:

```go
runner.Enqueue(&jobs.Job{
    Type:       "external_api",
    Payload:    data,
    MaxRetries: 5,              // Try up to 5 times
    RetryDelay: 2 * time.Second, // Base delay
    // Actual delays: 2s, 4s, 8s, 16s, 32s
})
```

### Hooks

```go
runner := jobs.New(jobs.Config{
    Logger: logger,
    OnStart: func(job *jobs.Job) {
        metrics.JobStarted(job.Type)
    },
    OnSuccess: func(job *jobs.Job) {
        metrics.JobCompleted(job.Type, job.Attempts)
    },
    OnError: func(job *jobs.Job, err error) {
        metrics.JobFailed(job.Type)
        alerting.NotifyFailure(job, err)
    },
    OnRetry: func(job *jobs.Job, err error, attempt int) {
        logger.Warn("job retry",
            zap.String("job_id", job.ID),
            zap.Int("attempt", attempt),
            zap.Error(err),
        )
    },
})
```

---

## Scheduler

Recurring jobs with intervals and cron expressions.

### NewScheduler

**Location:** `scheduler.go`

```go
func NewScheduler(logger *zap.Logger, opts ...SchedulerOption) *Scheduler
```

### Scheduler Options

```go
// With distributed locking (prevents duplicate execution across instances)
scheduler := jobs.NewScheduler(logger,
    jobs.WithLocker(redisLocker),
)

// With execution history
scheduler := jobs.NewScheduler(logger,
    jobs.WithHistory(historyStore),
)

// With custom worker ID
scheduler := jobs.NewScheduler(logger,
    jobs.WithWorkerID("worker-1"),
)

// Combined
scheduler := jobs.NewScheduler(logger,
    jobs.WithLocker(redisLocker),
    jobs.WithHistory(historyStore),
)
```

### Interval-Based Jobs

```go
scheduler := jobs.NewScheduler(logger)

// Add jobs with intervals
scheduler.Add(&jobs.ScheduledJob{
    Name:     "cleanup_temp_files",
    Interval: time.Hour,
    Handler: func(ctx context.Context) error {
        return cleanupTempFiles(ctx)
    },
})

scheduler.Add(&jobs.ScheduledJob{
    Name:           "sync_external_data",
    Interval:       5 * time.Minute,
    RunImmediately: true, // Run once at startup
    Timeout:        2 * time.Minute,
    Handler: func(ctx context.Context) error {
        return syncData(ctx)
    },
})

// Simple form
scheduler.Every(time.Minute, "health_check", func(ctx context.Context) error {
    return checkHealth(ctx)
})

// Start and stop
scheduler.Start()
defer scheduler.Stop(ctx)
```

### Cron-Based Jobs

```go
scheduler := jobs.NewScheduler(logger)

// Add cron jobs
scheduler.AddCron(&jobs.CronJob{
    Name: "nightly_backup",
    Cron: "0 0 * * *", // Every day at midnight
    Handler: func(ctx context.Context) error {
        return runBackup(ctx)
    },
})

scheduler.AddCron(&jobs.CronJob{
    Name:     "weekly_report",
    Cron:     "0 9 * * 1", // Every Monday at 9 AM
    Timeout:  30 * time.Minute,
    Location: time.UTC,
    Handler: func(ctx context.Context) error {
        return generateWeeklyReport(ctx)
    },
})

// Simple form
scheduler.Cron("*/15 * * * *", "check_queue", func(ctx context.Context) error {
    return checkQueueHealth(ctx)
})

// Predefined expressions
scheduler.Cron("@daily", "daily_cleanup", cleanupHandler)
scheduler.Cron("@hourly", "hourly_sync", syncHandler)

scheduler.Start()
```

### Cron Expression Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12 or jan-dec)
│ │ │ │ ┌───────────── day of week (0-6 or sun-sat)
│ │ │ │ │
* * * * *
```

**Field Values:**
- `*` — Any value
- `*/n` — Every n (e.g., `*/15` = every 15)
- `n` — Specific value
- `n-m` — Range (e.g., `1-5` = Monday-Friday)
- `n,m,o` — List (e.g., `1,15` = 1st and 15th)

**Examples:**
```go
"0 0 * * *"       // Every day at midnight
"*/15 * * * *"    // Every 15 minutes
"0 9 * * 1-5"     // At 9 AM, Monday through Friday
"0 0 1 * *"       // At midnight on the 1st of every month
"30 4 * * sun"    // At 4:30 AM every Sunday
"0 */2 * * *"     // Every 2 hours
```

**Predefined Expressions:**
```go
"@yearly"   // 0 0 1 1 * (January 1st)
"@monthly"  // 0 0 1 * * (1st of each month)
"@weekly"   // 0 0 * * 0 (Sunday midnight)
"@daily"    // 0 0 * * * (midnight)
"@hourly"   // 0 * * * * (start of each hour)
```

### Named Job Management

```go
// Get job by name
if job, ok := scheduler.Get("nightly_backup"); ok {
    fmt.Printf("Job: %s, Type: %s, Next Run: %s\n",
        job.Name, job.Type, job.NextRun)
}

// List all jobs
for _, name := range scheduler.List() {
    fmt.Println(name)
}

// List with details
for _, job := range scheduler.ListJobs() {
    fmt.Printf("%s (%s): %s\n", job.Name, job.Type, job.Cron)
}

// Remove a job
scheduler.Remove("old_job")

// Run a job immediately
scheduler.RunNow(ctx, "nightly_backup")
```

### ScheduledJob

```go
type ScheduledJob struct {
    Name           string                            // Job identifier
    Interval       time.Duration                     // Run interval
    Handler        func(ctx context.Context) error   // Job function
    RunImmediately bool                              // Run on start
    Timeout        time.Duration                     // Per-run timeout (default: 5m)
}
```

### CronJob

```go
type CronJob struct {
    Name     string                            // Job identifier
    Cron     string                            // Cron expression
    Handler  func(ctx context.Context) error   // Job function
    Timeout  time.Duration                     // Per-run timeout (default: 5m)
    Location *time.Location                    // Timezone (default: Local)
}
```

### Scheduler Methods

```go
scheduler.Add(job *ScheduledJob) error          // Add interval job
scheduler.AddCron(job *CronJob) error           // Add cron job
scheduler.Every(interval, name, handler) error  // Simple interval add
scheduler.Cron(expr, name, handler) error       // Simple cron add
scheduler.Start()                               // Start all jobs
scheduler.Stop(ctx context.Context) error       // Graceful shutdown
scheduler.Remove(name string) bool              // Remove job by name
scheduler.Get(name string) (*JobInfo, bool)     // Get job info
scheduler.List() []string                       // List job names
scheduler.ListJobs() []*JobInfo                 // List all job info
scheduler.RunNow(ctx, name) error               // Run job immediately
scheduler.IsRunning() bool                      // Check if running
scheduler.WorkerID() string                     // Get worker ID
```

---

## Distributed Locking

Prevent duplicate job execution across multiple instances.

### Redis Locker

```go
// Create Redis locker
locker := jobs.NewRedisLocker(jobs.RedisLockerConfig{
    Client: redisClient, // Implements RedisClient interface
    Prefix: "myapp:lock:",
})

// Use with scheduler
scheduler := jobs.NewScheduler(logger, jobs.WithLocker(locker))
scheduler.Cron("0 * * * *", "hourly_job", handler)
scheduler.Start()

// When the job runs, it will:
// 1. Try to acquire lock "myapp:lock:scheduler:hourly_job"
// 2. Skip if another instance holds the lock
// 3. Execute if lock acquired
// 4. Release lock when done
```

### RedisClient Interface

Implement this interface with your Redis client:

```go
type RedisClient interface {
    SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
    Get(ctx context.Context, key string) (string, error)
    Del(ctx context.Context, key string) error
    Expire(ctx context.Context, key string, ttl time.Duration) error
    Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
}
```

### Memory Locker (Single Instance)

```go
// For single-instance or testing
locker := jobs.NewMemoryLocker()
scheduler := jobs.NewScheduler(logger, jobs.WithLocker(locker))
```

### Manual Lock Usage

```go
locker := jobs.NewRedisLocker(cfg)

// Simple lock
acquired, err := locker.Acquire(ctx, "my-task", 5*time.Minute)
if acquired {
    defer locker.Release(ctx, "my-task")
    // Do work...
}

// With helper
lock := jobs.NewLock(locker, "my-task", 5*time.Minute)
if ok, _ := lock.Acquire(ctx); ok {
    defer lock.Release(ctx)
    // Do work...
}

// Wait for lock
if err := lock.AcquireWait(ctx, 100*time.Millisecond); err == nil {
    defer lock.Release(ctx)
    // Do work...
}

// Execute with lock
err := jobs.WithLock(ctx, locker, "my-task", 5*time.Minute, func(ctx context.Context) error {
    return doWork(ctx)
})
```

---

## Job History & Monitoring

Track job executions and monitor health.

### Memory History Store

```go
// Create history store (keeps last 100 executions per job)
history := jobs.NewMemoryHistoryStore(100)

// Use with scheduler
scheduler := jobs.NewScheduler(logger, jobs.WithHistory(history))
```

### Redis History Store

```go
history := jobs.NewRedisHistoryStore(jobs.RedisHistoryStoreConfig{
    Client:    redisClient,
    Prefix:    "myapp:job_history:",
    MaxPerJob: 100,
    TTL:       7 * 24 * time.Hour, // Keep 7 days
})

scheduler := jobs.NewScheduler(logger, jobs.WithHistory(history))
```

### Querying History

```go
// Get recent executions
execs, _ := history.GetExecutions(ctx, "nightly_backup", 10)
for _, exec := range execs {
    fmt.Printf("%s: %s (%s)\n", exec.ID, exec.Status, exec.Duration)
}

// Get specific execution
exec, _ := history.GetExecution(ctx, "exec_123456")

// Get job statistics
stats, _ := history.GetStats(ctx, "nightly_backup")
fmt.Printf("Total: %d, Success: %d, Failed: %d\n",
    stats.TotalExecutions,
    stats.SuccessfulExecutions,
    stats.FailedExecutions,
)
fmt.Printf("Avg Duration: %s\n", stats.AverageDuration)

// Get all job stats
allStats, _ := history.GetAllStats(ctx)
```

### Monitoring

```go
monitor := jobs.NewMonitor(history)

// Health check all jobs
health, _ := monitor.HealthCheck(ctx)
for name, healthy := range health {
    if !healthy {
        alert(fmt.Sprintf("Job %s is unhealthy", name))
    }
}

// Failure rate
rate, _ := monitor.FailureRate(ctx, "api_sync")
if rate > 0.1 { // >10% failure rate
    alert("api_sync failure rate too high")
}

// Average latency
latency, _ := monitor.AverageLatency(ctx, "report_generator")
```

### JobExecution

```go
type JobExecution struct {
    ID          string        // Unique execution ID
    JobName     string        // Job name
    Status      JobStatus     // pending, running, completed, failed, skipped
    StartedAt   time.Time     // Start time
    CompletedAt *time.Time    // End time
    Duration    time.Duration // Execution duration
    Error       string        // Error message (if failed)
    WorkerID    string        // Which worker ran this
    Metadata    map[string]string
}
```

### JobStats

```go
type JobStats struct {
    JobName              string
    TotalExecutions      int64
    SuccessfulExecutions int64
    FailedExecutions     int64
    SkippedExecutions    int64
    LastExecution        *time.Time
    LastSuccess          *time.Time
    LastFailure          *time.Time
    LastError            string
    AverageDuration      time.Duration
    MinDuration          time.Duration
    MaxDuration          time.Duration
}
```

---

## Pool

Simple worker pool for bounded-concurrency async tasks.

### NewPool

**Location:** `pool.go`

```go
func NewPool(workers int, logger *zap.Logger) *Pool
```

### Basic Usage

```go
pool := jobs.NewPool(10, logger)

// Run async task (blocks if pool is full)
pool.Go(func() {
    sendNotification(userID)
})

// Run with context
pool.GoWithContext(ctx, func(ctx context.Context) {
    processItem(ctx, item)
})

// Try to run (returns false if pool is full)
if !pool.TryGo(func() { doWork() }) {
    log.Warn("pool at capacity")
}

// Wait for all tasks
pool.Wait()

// Wait with timeout
if !pool.WaitWithTimeout(30 * time.Second) {
    log.Warn("some tasks still running")
}
```

### Futures

Get results from async tasks:

```go
pool := jobs.NewPool(10, logger)

// Submit task that returns a value
future := jobs.Submit(pool, func() (User, error) {
    return fetchUser(ctx, userID)
})

// Do other work...

// Get result (blocks until ready)
user, err := future.Wait()

// Or with timeout
user, err := future.WaitWithTimeout(5 * time.Second)

// Check if ready without blocking
if future.Ready() {
    user, err := future.Wait()
}

// Use the done channel
select {
case <-future.Done():
    user, err := future.Wait()
case <-ctx.Done():
    // Cancelled
}
```

### Pool Methods

```go
pool.Go(task func())                               // Run async (blocks if full)
pool.GoWithContext(ctx, task func(ctx))            // Run with context
pool.TryGo(task func()) bool                       // Try run (false if full)
pool.Wait()                                        // Wait for all tasks
pool.WaitWithTimeout(timeout) bool                 // Wait with timeout
pool.Running() int                                 // Current running tasks
```

---

## Complete Example

```go
package main

import (
    "context"
    "time"

    "github.com/dalemusser/waffle/pantry/jobs"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Create distributed locker (for multi-instance)
    locker := jobs.NewRedisLocker(jobs.RedisLockerConfig{
        Client: redisClient,
    })

    // Create history store
    history := jobs.NewMemoryHistoryStore(100)

    // Create scheduler with options
    scheduler := jobs.NewScheduler(logger,
        jobs.WithLocker(locker),
        jobs.WithHistory(history),
    )

    // Add interval jobs
    scheduler.Every(time.Minute, "health_check", func(ctx context.Context) error {
        return checkHealth(ctx)
    })

    // Add cron jobs
    scheduler.Cron("0 0 * * *", "nightly_backup", func(ctx context.Context) error {
        return runBackup(ctx)
    })

    scheduler.Cron("0 9 * * 1", "weekly_report", func(ctx context.Context) error {
        return generateReport(ctx)
    })

    // Start scheduler
    scheduler.Start()

    // Create monitor
    monitor := jobs.NewMonitor(history)

    // Periodic health check
    go func() {
        for {
            health, _ := monitor.HealthCheck(context.Background())
            for name, ok := range health {
                if !ok {
                    logger.Warn("unhealthy job", zap.String("job", name))
                }
            }
            time.Sleep(time.Minute)
        }
    }()

    // Wait for shutdown signal...

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    scheduler.Stop(ctx)
}
```

---

## Choosing the Right Tool

| Use Case | Component |
|----------|-----------|
| Deferred work (email, notifications) | Runner |
| Work with retries | Runner |
| Typed job handlers | Runner |
| Recurring tasks (cleanup, sync) | Scheduler |
| Cron-like scheduling | Scheduler + Cron |
| Multi-instance coordination | Scheduler + Locker |
| Execution tracking | Scheduler + History |
| One-off async tasks | Pool |
| Bounded concurrency | Pool |
| Fan-out processing | Pool |
| Getting results from async work | Pool + Future |

---

## See Also

- [mq/rabbitmq](../mq/rabbitmq/rabbitmq.md) — Distributed job queues
- [mq/sqs](../mq/sqs/sqs.md) — AWS-based job queues
- [app](../app/app.md) — Application lifecycle
