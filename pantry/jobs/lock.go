package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common lock errors.
var (
	ErrLockNotAcquired = errors.New("jobs: lock not acquired")
	ErrLockNotHeld     = errors.New("jobs: lock not held")
	ErrLockExpired     = errors.New("jobs: lock expired")
)

// Locker is the interface for distributed locking.
type Locker interface {
	// Acquire attempts to acquire a lock with the given key.
	// Returns true if the lock was acquired, false otherwise.
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Release releases a lock with the given key.
	// Returns true if the lock was released, false if it wasn't held.
	Release(ctx context.Context, key string) (bool, error)

	// Extend extends the TTL of a held lock.
	// Returns an error if the lock is not held.
	Extend(ctx context.Context, key string, ttl time.Duration) error

	// IsHeld returns true if we currently hold the lock.
	IsHeld(ctx context.Context, key string) (bool, error)
}

// RedisLocker implements distributed locking using Redis.
type RedisLocker struct {
	client   RedisClient
	prefix   string
	ownerID  string
	mu       sync.Mutex
	held     map[string]bool
}

// RedisClient is the interface for Redis operations needed by the locker.
type RedisClient interface {
	// SetNX sets a key if it doesn't exist, with TTL.
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)

	// Get gets a value by key.
	Get(ctx context.Context, key string) (string, error)

	// Del deletes a key.
	Del(ctx context.Context, key string) error

	// Expire sets a TTL on a key.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Eval runs a Lua script.
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
}

// RedisLockerConfig configures the Redis locker.
type RedisLockerConfig struct {
	// Client is the Redis client.
	Client RedisClient

	// Prefix is the key prefix for locks.
	// Default: "lock:"
	Prefix string

	// OwnerID is the unique identifier for this instance.
	// Default: random UUID
	OwnerID string
}

// NewRedisLocker creates a new Redis-based distributed locker.
func NewRedisLocker(cfg RedisLockerConfig) *RedisLocker {
	if cfg.Prefix == "" {
		cfg.Prefix = "lock:"
	}
	if cfg.OwnerID == "" {
		cfg.OwnerID = generateOwnerID()
	}

	return &RedisLocker{
		client:  cfg.Client,
		prefix:  cfg.Prefix,
		ownerID: cfg.OwnerID,
		held:    make(map[string]bool),
	}
}

// generateOwnerID generates a unique owner ID.
func generateOwnerID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Acquire attempts to acquire a lock.
func (l *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fullKey := l.prefix + key

	ok, err := l.client.SetNX(ctx, fullKey, l.ownerID, ttl)
	if err != nil {
		return false, fmt.Errorf("jobs: failed to acquire lock: %w", err)
	}

	if ok {
		l.mu.Lock()
		l.held[key] = true
		l.mu.Unlock()
	}

	return ok, nil
}

// Release releases a lock.
func (l *RedisLocker) Release(ctx context.Context, key string) (bool, error) {
	fullKey := l.prefix + key

	// Use Lua script to ensure we only delete our own lock
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, l.ownerID)
	if err != nil {
		return false, fmt.Errorf("jobs: failed to release lock: %w", err)
	}

	released := result == int64(1) || result == "1"

	l.mu.Lock()
	delete(l.held, key)
	l.mu.Unlock()

	return released, nil
}

// Extend extends the TTL of a held lock.
func (l *RedisLocker) Extend(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := l.prefix + key

	// Use Lua script to ensure we only extend our own lock
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, l.ownerID, int64(ttl/time.Millisecond))
	if err != nil {
		return fmt.Errorf("jobs: failed to extend lock: %w", err)
	}

	if result == int64(0) || result == "0" {
		return ErrLockNotHeld
	}

	return nil
}

// IsHeld returns true if we currently hold the lock.
func (l *RedisLocker) IsHeld(ctx context.Context, key string) (bool, error) {
	fullKey := l.prefix + key

	value, err := l.client.Get(ctx, fullKey)
	if err != nil {
		// Key doesn't exist
		return false, nil
	}

	return value == l.ownerID, nil
}

// OwnerID returns this locker's owner ID.
func (l *RedisLocker) OwnerID() string {
	return l.ownerID
}

// MemoryLocker implements in-memory locking for single-instance deployments.
type MemoryLocker struct {
	mu      sync.Mutex
	locks   map[string]*memoryLock
	ownerID string
}

type memoryLock struct {
	owner   string
	expires time.Time
}

// NewMemoryLocker creates a new in-memory locker.
func NewMemoryLocker() *MemoryLocker {
	return &MemoryLocker{
		locks:   make(map[string]*memoryLock),
		ownerID: generateOwnerID(),
	}
}

// Acquire attempts to acquire a lock.
func (l *MemoryLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Check if lock exists and is still valid
	if existing, ok := l.locks[key]; ok {
		if existing.expires.After(now) {
			// Lock is held by someone else
			if existing.owner != l.ownerID {
				return false, nil
			}
			// We already hold the lock, extend it
			existing.expires = now.Add(ttl)
			return true, nil
		}
	}

	// Acquire lock
	l.locks[key] = &memoryLock{
		owner:   l.ownerID,
		expires: now.Add(ttl),
	}

	return true, nil
}

// Release releases a lock.
func (l *MemoryLocker) Release(ctx context.Context, key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	existing, ok := l.locks[key]
	if !ok {
		return false, nil
	}

	if existing.owner != l.ownerID {
		return false, nil
	}

	delete(l.locks, key)
	return true, nil
}

// Extend extends the TTL of a held lock.
func (l *MemoryLocker) Extend(ctx context.Context, key string, ttl time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	existing, ok := l.locks[key]
	if !ok || existing.owner != l.ownerID {
		return ErrLockNotHeld
	}

	existing.expires = time.Now().Add(ttl)
	return nil
}

// IsHeld returns true if we currently hold the lock.
func (l *MemoryLocker) IsHeld(ctx context.Context, key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	existing, ok := l.locks[key]
	if !ok {
		return false, nil
	}

	if existing.expires.Before(time.Now()) {
		delete(l.locks, key)
		return false, nil
	}

	return existing.owner == l.ownerID, nil
}

// Cleanup removes expired locks.
func (l *MemoryLocker) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, lock := range l.locks {
		if lock.expires.Before(now) {
			delete(l.locks, key)
		}
	}
}

// Lock is a helper that holds a lock and provides methods to work with it.
type Lock struct {
	locker Locker
	key    string
	ttl    time.Duration
	held   bool
	mu     sync.Mutex
}

// NewLock creates a new lock helper.
func NewLock(locker Locker, key string, ttl time.Duration) *Lock {
	return &Lock{
		locker: locker,
		key:    key,
		ttl:    ttl,
	}
}

// Acquire attempts to acquire the lock.
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ok, err := l.locker.Acquire(ctx, l.key, l.ttl)
	if err != nil {
		return false, err
	}

	l.held = ok
	return ok, nil
}

// AcquireWait attempts to acquire the lock, waiting until acquired or context is cancelled.
func (l *Lock) AcquireWait(ctx context.Context, retryInterval time.Duration) error {
	for {
		ok, err := l.Acquire(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

// Release releases the lock.
func (l *Lock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.held {
		return nil
	}

	_, err := l.locker.Release(ctx, l.key)
	l.held = false
	return err
}

// Extend extends the lock TTL.
func (l *Lock) Extend(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.held {
		return ErrLockNotHeld
	}

	return l.locker.Extend(ctx, l.key, l.ttl)
}

// IsHeld returns true if the lock is held.
func (l *Lock) IsHeld() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.held
}

// WithLock executes a function while holding a lock.
// The lock is automatically released when the function returns.
func WithLock(ctx context.Context, locker Locker, key string, ttl time.Duration, fn func(ctx context.Context) error) error {
	lock := NewLock(locker, key, ttl)

	ok, err := lock.Acquire(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockNotAcquired
	}
	defer lock.Release(ctx)

	return fn(ctx)
}

// WithLockWait executes a function while holding a lock, waiting to acquire.
func WithLockWait(ctx context.Context, locker Locker, key string, ttl time.Duration, retryInterval time.Duration, fn func(ctx context.Context) error) error {
	lock := NewLock(locker, key, ttl)

	if err := lock.AcquireWait(ctx, retryInterval); err != nil {
		return err
	}
	defer lock.Release(ctx)

	return fn(ctx)
}
