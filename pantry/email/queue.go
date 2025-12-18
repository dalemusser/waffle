// pantry/email/queue.go
// Queue integration for async email sending.
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QueuedEmail represents an email queued for async delivery.
type QueuedEmail struct {
	// ID is a unique identifier for tracking.
	ID string `json:"id"`

	// Message is the email to send.
	Message Message `json:"message"`

	// Priority determines send order (higher = more urgent).
	Priority int `json:"priority,omitempty"`

	// ScheduledAt is when to send (nil = immediately).
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`

	// MaxRetries is the number of retry attempts on failure.
	MaxRetries int `json:"max_retries,omitempty"`

	// Metadata contains arbitrary key-value data for tracking.
	Metadata map[string]string `json:"metadata,omitempty"`

	// CreatedAt is when the email was queued.
	CreatedAt time.Time `json:"created_at"`

	// Attempts tracks delivery attempts.
	Attempts int `json:"attempts"`

	// LastError holds the most recent error.
	LastError string `json:"last_error,omitempty"`

	// Status is the current delivery status.
	Status EmailStatus `json:"status"`
}

// EmailStatus represents the delivery status of a queued email.
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusScheduled EmailStatus = "scheduled"
	EmailStatusSending   EmailStatus = "sending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusFailed    EmailStatus = "failed"
)

// Queue provides async email delivery.
type Queue struct {
	sender  *Sender
	store   QueueStore
	logger  *zap.Logger
	workers int

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Hooks
	onSent   func(*QueuedEmail)
	onFailed func(*QueuedEmail, error)
}

// QueueStore is the interface for persistent email queue storage.
type QueueStore interface {
	// Enqueue adds an email to the queue.
	Enqueue(ctx context.Context, email *QueuedEmail) error

	// Dequeue retrieves the next email ready to send.
	// Returns nil if no emails are ready.
	Dequeue(ctx context.Context) (*QueuedEmail, error)

	// Update updates an email's status.
	Update(ctx context.Context, email *QueuedEmail) error

	// Get retrieves an email by ID.
	Get(ctx context.Context, id string) (*QueuedEmail, error)

	// List retrieves emails matching the filter.
	List(ctx context.Context, filter QueueFilter) ([]*QueuedEmail, error)

	// Delete removes an email from the queue.
	Delete(ctx context.Context, id string) error

	// Stats returns queue statistics.
	Stats(ctx context.Context) (*QueueStats, error)
}

// QueueFilter filters queue listings.
type QueueFilter struct {
	Status    EmailStatus
	Limit     int
	Offset    int
	CreatedAt *TimeRange
}

// TimeRange specifies a time range.
type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// QueueStats contains queue statistics.
type QueueStats struct {
	Pending   int64
	Scheduled int64
	Sending   int64
	Sent      int64
	Failed    int64
	Total     int64
}

// QueueConfig configures the email queue.
type QueueConfig struct {
	// Sender is the email sender to use.
	Sender *Sender

	// Store is the queue storage backend.
	Store QueueStore

	// Logger for queue events.
	Logger *zap.Logger

	// Workers is the number of concurrent senders. Default: 2.
	Workers int

	// PollInterval is how often to check for pending emails. Default: 5s.
	PollInterval time.Duration

	// OnSent is called when an email is sent successfully.
	OnSent func(*QueuedEmail)

	// OnFailed is called when an email fails permanently.
	OnFailed func(*QueuedEmail, error)
}

// NewQueue creates a new email queue.
func NewQueue(cfg QueueConfig) *Queue {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}

	return &Queue{
		sender:   cfg.Sender,
		store:    cfg.Store,
		logger:   cfg.Logger,
		workers:  cfg.Workers,
		stopCh:   make(chan struct{}),
		onSent:   cfg.OnSent,
		onFailed: cfg.OnFailed,
	}
}

// Enqueue adds an email to the queue for async delivery.
func (q *Queue) Enqueue(ctx context.Context, email *QueuedEmail) error {
	if email.ID == "" {
		email.ID = generateEmailID()
	}
	if email.CreatedAt.IsZero() {
		email.CreatedAt = time.Now()
	}
	if email.MaxRetries == 0 {
		email.MaxRetries = 3
	}
	if email.Status == "" {
		if email.ScheduledAt != nil && email.ScheduledAt.After(time.Now()) {
			email.Status = EmailStatusScheduled
		} else {
			email.Status = EmailStatusPending
		}
	}

	if err := q.store.Enqueue(ctx, email); err != nil {
		return fmt.Errorf("email: failed to enqueue: %w", err)
	}

	q.logger.Debug("email queued",
		zap.String("id", email.ID),
		zap.Int("recipients", len(email.Message.To)),
	)

	return nil
}

// EnqueueMessage is a convenience method to queue a message directly.
func (q *Queue) EnqueueMessage(ctx context.Context, msg Message) (string, error) {
	email := &QueuedEmail{
		Message: msg,
	}
	if err := q.Enqueue(ctx, email); err != nil {
		return "", err
	}
	return email.ID, nil
}

// EnqueueSimple queues a simple text email.
func (q *Queue) EnqueueSimple(ctx context.Context, to, subject, body string) (string, error) {
	return q.EnqueueMessage(ctx, Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: body,
	})
}

// EnqueueHTML queues an HTML email with text fallback.
func (q *Queue) EnqueueHTML(ctx context.Context, to, subject, textBody, htmlBody string) (string, error) {
	return q.EnqueueMessage(ctx, Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	})
}

// Schedule queues an email for delivery at a specific time.
func (q *Queue) Schedule(ctx context.Context, email *QueuedEmail, at time.Time) error {
	email.ScheduledAt = &at
	email.Status = EmailStatusScheduled
	return q.Enqueue(ctx, email)
}

// Get retrieves a queued email by ID.
func (q *Queue) Get(ctx context.Context, id string) (*QueuedEmail, error) {
	return q.store.Get(ctx, id)
}

// Cancel cancels a queued email that hasn't been sent.
func (q *Queue) Cancel(ctx context.Context, id string) error {
	email, err := q.store.Get(ctx, id)
	if err != nil {
		return err
	}

	if email.Status == EmailStatusSent || email.Status == EmailStatusSending {
		return fmt.Errorf("email: cannot cancel email with status %s", email.Status)
	}

	return q.store.Delete(ctx, id)
}

// Stats returns queue statistics.
func (q *Queue) Stats(ctx context.Context) (*QueueStats, error) {
	return q.store.Stats(ctx)
}

// Start begins processing the email queue.
func (q *Queue) Start() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.stopCh = make(chan struct{})
	q.mu.Unlock()

	q.logger.Info("starting email queue", zap.Int("workers", q.workers))

	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop gracefully stops the queue processor.
func (q *Queue) Stop(ctx context.Context) error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = false
	close(q.stopCh)
	q.mu.Unlock()

	q.logger.Info("stopping email queue")

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		q.logger.Info("email queue stopped")
		return nil
	case <-ctx.Done():
		q.logger.Warn("email queue shutdown timed out")
		return ctx.Err()
	}
}

// worker processes emails from the queue.
func (q *Queue) worker(id int) {
	defer q.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			q.processNext()
		}
	}
}

// processNext dequeues and sends one email.
func (q *Queue) processNext() {
	ctx := context.Background()

	email, err := q.store.Dequeue(ctx)
	if err != nil {
		q.logger.Error("failed to dequeue email", zap.Error(err))
		return
	}
	if email == nil {
		return // No emails ready
	}

	// Mark as sending
	email.Status = EmailStatusSending
	email.Attempts++
	q.store.Update(ctx, email)

	q.logger.Debug("sending queued email",
		zap.String("id", email.ID),
		zap.Int("attempt", email.Attempts),
	)

	// Send the email
	err = q.sender.Send(ctx, email.Message)

	if err == nil {
		email.Status = EmailStatusSent
		q.store.Update(ctx, email)

		q.logger.Info("queued email sent",
			zap.String("id", email.ID),
			zap.Int("attempts", email.Attempts),
		)

		if q.onSent != nil {
			q.onSent(email)
		}
		return
	}

	// Handle failure
	email.LastError = err.Error()

	if email.Attempts >= email.MaxRetries {
		// Permanent failure
		email.Status = EmailStatusFailed
		q.store.Update(ctx, email)

		q.logger.Error("queued email failed permanently",
			zap.String("id", email.ID),
			zap.Int("attempts", email.Attempts),
			zap.Error(err),
		)

		if q.onFailed != nil {
			q.onFailed(email, err)
		}
		return
	}

	// Schedule retry
	email.Status = EmailStatusPending
	q.store.Update(ctx, email)

	q.logger.Warn("queued email failed, will retry",
		zap.String("id", email.ID),
		zap.Int("attempt", email.Attempts),
		zap.Int("max_retries", email.MaxRetries),
		zap.Error(err),
	)
}

// generateEmailID generates a unique email ID.
func generateEmailID() string {
	return fmt.Sprintf("email_%d", time.Now().UnixNano())
}

// MemoryQueueStore is an in-memory queue store for testing and development.
type MemoryQueueStore struct {
	mu     sync.RWMutex
	emails map[string]*QueuedEmail
}

// NewMemoryQueueStore creates a new in-memory queue store.
func NewMemoryQueueStore() *MemoryQueueStore {
	return &MemoryQueueStore{
		emails: make(map[string]*QueuedEmail),
	}
}

// Enqueue adds an email to the queue.
func (s *MemoryQueueStore) Enqueue(ctx context.Context, email *QueuedEmail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to prevent external modifications
	copy := *email
	copy.Message = email.Message
	if email.Metadata != nil {
		copy.Metadata = make(map[string]string)
		for k, v := range email.Metadata {
			copy.Metadata[k] = v
		}
	}

	s.emails[email.ID] = &copy
	return nil
}

// Dequeue retrieves the next email ready to send.
func (s *MemoryQueueStore) Dequeue(ctx context.Context) (*QueuedEmail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var oldest *QueuedEmail
	var oldestTime time.Time

	for _, email := range s.emails {
		// Skip if not ready
		if email.Status != EmailStatusPending && email.Status != EmailStatusScheduled {
			continue
		}

		// Check scheduled time
		if email.Status == EmailStatusScheduled {
			if email.ScheduledAt != nil && email.ScheduledAt.After(now) {
				continue
			}
		}

		// Find oldest (or highest priority)
		if oldest == nil ||
			email.Priority > oldest.Priority ||
			(email.Priority == oldest.Priority && email.CreatedAt.Before(oldestTime)) {
			oldest = email
			oldestTime = email.CreatedAt
		}
	}

	if oldest == nil {
		return nil, nil
	}

	// Return a copy
	copy := *oldest
	return &copy, nil
}

// Update updates an email's status.
func (s *MemoryQueueStore) Update(ctx context.Context, email *QueuedEmail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.emails[email.ID]; !exists {
		return fmt.Errorf("email: email %s not found", email.ID)
	}

	// Deep copy
	copy := *email
	if email.Metadata != nil {
		copy.Metadata = make(map[string]string)
		for k, v := range email.Metadata {
			copy.Metadata[k] = v
		}
	}

	s.emails[email.ID] = &copy
	return nil
}

// Get retrieves an email by ID.
func (s *MemoryQueueStore) Get(ctx context.Context, id string) (*QueuedEmail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email, exists := s.emails[id]
	if !exists {
		return nil, fmt.Errorf("email: email %s not found", id)
	}

	// Return a copy
	copy := *email
	return &copy, nil
}

// List retrieves emails matching the filter.
func (s *MemoryQueueStore) List(ctx context.Context, filter QueueFilter) ([]*QueuedEmail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*QueuedEmail
	skipped := 0

	for _, email := range s.emails {
		// Apply status filter
		if filter.Status != "" && email.Status != filter.Status {
			continue
		}

		// Apply time range filter
		if filter.CreatedAt != nil {
			if filter.CreatedAt.From != nil && email.CreatedAt.Before(*filter.CreatedAt.From) {
				continue
			}
			if filter.CreatedAt.To != nil && email.CreatedAt.After(*filter.CreatedAt.To) {
				continue
			}
		}

		// Apply offset
		if filter.Offset > 0 && skipped < filter.Offset {
			skipped++
			continue
		}

		// Apply limit
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}

		copy := *email
		result = append(result, &copy)
	}

	return result, nil
}

// Delete removes an email from the queue.
func (s *MemoryQueueStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.emails[id]; !exists {
		return fmt.Errorf("email: email %s not found", id)
	}

	delete(s.emails, id)
	return nil
}

// Stats returns queue statistics.
func (s *MemoryQueueStore) Stats(ctx context.Context) (*QueueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &QueueStats{}
	for _, email := range s.emails {
		switch email.Status {
		case EmailStatusPending:
			stats.Pending++
		case EmailStatusScheduled:
			stats.Scheduled++
		case EmailStatusSending:
			stats.Sending++
		case EmailStatusSent:
			stats.Sent++
		case EmailStatusFailed:
			stats.Failed++
		}
		stats.Total++
	}

	return stats, nil
}

// Cleanup removes old sent and failed emails.
func (s *MemoryQueueStore) Cleanup(ctx context.Context, maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, email := range s.emails {
		if (email.Status == EmailStatusSent || email.Status == EmailStatusFailed) &&
			email.CreatedAt.Before(cutoff) {
			delete(s.emails, id)
			removed++
		}
	}

	return removed
}

// RedisQueueStore is a Redis-backed queue store for production use.
type RedisQueueStore struct {
	client RedisQueueClient
	prefix string
}

// RedisQueueClient is the interface for Redis operations.
type RedisQueueClient interface {
	// Set sets a key with optional TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Get gets a value by key.
	Get(ctx context.Context, key string) (string, error)

	// Del deletes keys.
	Del(ctx context.Context, keys ...string) error

	// Keys returns keys matching a pattern.
	Keys(ctx context.Context, pattern string) ([]string, error)

	// ZAdd adds to a sorted set.
	ZAdd(ctx context.Context, key string, score float64, member string) error

	// ZRangeByScore gets members by score range.
	ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error)

	// ZRem removes from a sorted set.
	ZRem(ctx context.Context, key string, members ...string) error
}

// RedisQueueConfig configures the Redis queue store.
type RedisQueueConfig struct {
	Client RedisQueueClient
	Prefix string // Key prefix (default: "email_queue:")
}

// NewRedisQueueStore creates a new Redis-backed queue store.
func NewRedisQueueStore(cfg RedisQueueConfig) *RedisQueueStore {
	if cfg.Prefix == "" {
		cfg.Prefix = "email_queue:"
	}
	return &RedisQueueStore{
		client: cfg.Client,
		prefix: cfg.Prefix,
	}
}

// Enqueue adds an email to the queue.
func (s *RedisQueueStore) Enqueue(ctx context.Context, email *QueuedEmail) error {
	data, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("email: failed to marshal email: %w", err)
	}

	// Store email data
	key := s.prefix + "data:" + email.ID
	if err := s.client.Set(ctx, key, string(data), 0); err != nil {
		return fmt.Errorf("email: failed to store email: %w", err)
	}

	// Add to appropriate queue based on status
	queueKey := s.queueKey(email.Status)
	score := s.calculateScore(email)
	if err := s.client.ZAdd(ctx, queueKey, score, email.ID); err != nil {
		return fmt.Errorf("email: failed to add to queue: %w", err)
	}

	return nil
}

// Dequeue retrieves the next email ready to send.
func (s *RedisQueueStore) Dequeue(ctx context.Context) (*QueuedEmail, error) {
	now := float64(time.Now().Unix())

	// Check pending queue first
	ids, err := s.client.ZRangeByScore(ctx, s.prefix+"queue:pending", 0, now, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("email: failed to dequeue: %w", err)
	}

	if len(ids) == 0 {
		// Check scheduled queue
		ids, err = s.client.ZRangeByScore(ctx, s.prefix+"queue:scheduled", 0, now, 0, 1)
		if err != nil {
			return nil, fmt.Errorf("email: failed to dequeue scheduled: %w", err)
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	id := ids[0]

	// Get email data
	data, err := s.client.Get(ctx, s.prefix+"data:"+id)
	if err != nil {
		return nil, fmt.Errorf("email: failed to get email data: %w", err)
	}

	var email QueuedEmail
	if err := json.Unmarshal([]byte(data), &email); err != nil {
		return nil, fmt.Errorf("email: failed to unmarshal email: %w", err)
	}

	// Remove from current queue
	s.client.ZRem(ctx, s.queueKey(email.Status), id)

	return &email, nil
}

// Update updates an email's status.
func (s *RedisQueueStore) Update(ctx context.Context, email *QueuedEmail) error {
	// Get old email to determine queue changes
	oldData, err := s.client.Get(ctx, s.prefix+"data:"+email.ID)
	if err == nil {
		var oldEmail QueuedEmail
		if json.Unmarshal([]byte(oldData), &oldEmail) == nil && oldEmail.Status != email.Status {
			// Remove from old queue
			s.client.ZRem(ctx, s.queueKey(oldEmail.Status), email.ID)
		}
	}

	// Store updated email
	data, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("email: failed to marshal email: %w", err)
	}

	if err := s.client.Set(ctx, s.prefix+"data:"+email.ID, string(data), 0); err != nil {
		return fmt.Errorf("email: failed to update email: %w", err)
	}

	// Add to new queue
	queueKey := s.queueKey(email.Status)
	score := s.calculateScore(email)
	if err := s.client.ZAdd(ctx, queueKey, score, email.ID); err != nil {
		return fmt.Errorf("email: failed to update queue: %w", err)
	}

	return nil
}

// Get retrieves an email by ID.
func (s *RedisQueueStore) Get(ctx context.Context, id string) (*QueuedEmail, error) {
	data, err := s.client.Get(ctx, s.prefix+"data:"+id)
	if err != nil {
		return nil, fmt.Errorf("email: email %s not found", id)
	}

	var email QueuedEmail
	if err := json.Unmarshal([]byte(data), &email); err != nil {
		return nil, fmt.Errorf("email: failed to unmarshal email: %w", err)
	}

	return &email, nil
}

// List retrieves emails matching the filter.
func (s *RedisQueueStore) List(ctx context.Context, filter QueueFilter) ([]*QueuedEmail, error) {
	var keys []string
	var err error

	if filter.Status != "" {
		// Get from specific queue
		queueKey := s.queueKey(filter.Status)
		ids, err := s.client.ZRangeByScore(ctx, queueKey, 0, float64(time.Now().Add(365*24*time.Hour).Unix()), int64(filter.Offset), int64(filter.Limit))
		if err != nil {
			return nil, fmt.Errorf("email: failed to list: %w", err)
		}
		for _, id := range ids {
			keys = append(keys, s.prefix+"data:"+id)
		}
	} else {
		// Get all email keys
		keys, err = s.client.Keys(ctx, s.prefix+"data:*")
		if err != nil {
			return nil, fmt.Errorf("email: failed to list keys: %w", err)
		}
	}

	var emails []*QueuedEmail
	for _, key := range keys {
		data, err := s.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var email QueuedEmail
		if err := json.Unmarshal([]byte(data), &email); err != nil {
			continue
		}

		// Apply time filter
		if filter.CreatedAt != nil {
			if filter.CreatedAt.From != nil && email.CreatedAt.Before(*filter.CreatedAt.From) {
				continue
			}
			if filter.CreatedAt.To != nil && email.CreatedAt.After(*filter.CreatedAt.To) {
				continue
			}
		}

		emails = append(emails, &email)
	}

	return emails, nil
}

// Delete removes an email from the queue.
func (s *RedisQueueStore) Delete(ctx context.Context, id string) error {
	// Get email to determine queue
	email, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove from queue
	s.client.ZRem(ctx, s.queueKey(email.Status), id)

	// Delete data
	return s.client.Del(ctx, s.prefix+"data:"+id)
}

// Stats returns queue statistics.
func (s *RedisQueueStore) Stats(ctx context.Context) (*QueueStats, error) {
	stats := &QueueStats{}

	// Count each status
	for _, status := range []EmailStatus{EmailStatusPending, EmailStatusScheduled, EmailStatusSending, EmailStatusSent, EmailStatusFailed} {
		ids, err := s.client.ZRangeByScore(ctx, s.queueKey(status), 0, float64(time.Now().Add(365*24*time.Hour).Unix()), 0, -1)
		if err != nil {
			continue
		}
		count := int64(len(ids))
		switch status {
		case EmailStatusPending:
			stats.Pending = count
		case EmailStatusScheduled:
			stats.Scheduled = count
		case EmailStatusSending:
			stats.Sending = count
		case EmailStatusSent:
			stats.Sent = count
		case EmailStatusFailed:
			stats.Failed = count
		}
		stats.Total += count
	}

	return stats, nil
}

// queueKey returns the Redis key for a status queue.
func (s *RedisQueueStore) queueKey(status EmailStatus) string {
	return s.prefix + "queue:" + string(status)
}

// calculateScore calculates the sort score for an email.
// Lower score = higher priority.
func (s *RedisQueueStore) calculateScore(email *QueuedEmail) float64 {
	// Base score is timestamp (older = lower score = processed first)
	var timestamp int64
	if email.ScheduledAt != nil {
		timestamp = email.ScheduledAt.Unix()
	} else {
		timestamp = email.CreatedAt.Unix()
	}

	// Subtract priority to put higher priority emails first
	// Priority 0 = no change, Priority 10 = 10 seconds earlier
	return float64(timestamp - int64(email.Priority))
}
