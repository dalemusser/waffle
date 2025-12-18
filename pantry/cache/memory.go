// cache/memory.go
package cache

import (
	"context"
	"sync"
	"time"
)

// Memory implements an in-memory cache with TTL support.
type Memory struct {
	mu      sync.RWMutex
	items   map[string]*item
	closed  bool
	stopCh  chan struct{}
	cleanCh chan struct{}
}

type item struct {
	value     []byte
	expiresAt time.Time
	noExpiry  bool
}

// MemoryConfig configures the in-memory cache.
type MemoryConfig struct {
	// CleanupInterval is how often to remove expired items.
	// Default: 1 minute. Set to 0 to disable background cleanup.
	CleanupInterval time.Duration

	// InitialCapacity is the initial map capacity.
	// Default: 100.
	InitialCapacity int
}

// DefaultMemoryConfig returns sensible defaults.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		CleanupInterval: time.Minute,
		InitialCapacity: 100,
	}
}

// NewMemory creates a new in-memory cache.
func NewMemory() *Memory {
	return NewMemoryWithConfig(DefaultMemoryConfig())
}

// NewMemoryWithConfig creates an in-memory cache with custom configuration.
func NewMemoryWithConfig(cfg MemoryConfig) *Memory {
	if cfg.InitialCapacity <= 0 {
		cfg.InitialCapacity = 100
	}

	m := &Memory{
		items:   make(map[string]*item, cfg.InitialCapacity),
		stopCh:  make(chan struct{}),
		cleanCh: make(chan struct{}),
	}

	// Start cleanup goroutine if interval is set
	if cfg.CleanupInterval > 0 {
		go m.cleanup(cfg.CleanupInterval)
	}

	return m
}

// Get retrieves a value by key.
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrClosed
	}

	it, exists := m.items[key]
	if !exists {
		return nil, ErrNotFound
	}

	// Check expiration
	if !it.noExpiry && time.Now().After(it.expiresAt) {
		return nil, ErrNotFound
	}

	// Return a copy to prevent mutation
	result := make([]byte, len(it.value))
	copy(result, it.value)
	return result, nil
}

// Set stores a value with the given TTL.
func (m *Memory) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	// Copy value to prevent external mutation
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	it := &item{
		value: valueCopy,
	}

	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	} else {
		it.noExpiry = true
	}

	m.items[key] = it
	return nil
}

// Delete removes a key from the cache.
func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	delete(m.items, key)
	return nil
}

// Exists checks if a key exists and is not expired.
func (m *Memory) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, ErrClosed
	}

	it, exists := m.items[key]
	if !exists {
		return false, nil
	}

	if !it.noExpiry && time.Now().After(it.expiresAt) {
		return false, nil
	}

	return true, nil
}

// Clear removes all entries from the cache.
func (m *Memory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	m.items = make(map[string]*item, 100)
	return nil
}

// Close stops the cleanup goroutine and releases resources.
func (m *Memory) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.stopCh)
	<-m.cleanCh // Wait for cleanup goroutine to stop
	return nil
}

// Size returns the number of items in the cache (including expired).
func (m *Memory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// GetMulti retrieves multiple values at once.
func (m *Memory) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrClosed
	}

	now := time.Now()
	result := make(map[string][]byte, len(keys))

	for _, key := range keys {
		it, exists := m.items[key]
		if !exists {
			continue
		}
		if !it.noExpiry && now.After(it.expiresAt) {
			continue
		}
		// Copy value
		val := make([]byte, len(it.value))
		copy(val, it.value)
		result[key] = val
	}

	return result, nil
}

// SetMulti stores multiple values at once.
func (m *Memory) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	var expiresAt time.Time
	noExpiry := ttl <= 0
	if !noExpiry {
		expiresAt = time.Now().Add(ttl)
	}

	for key, value := range items {
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)
		m.items[key] = &item{
			value:     valueCopy,
			expiresAt: expiresAt,
			noExpiry:  noExpiry,
		}
	}

	return nil
}

// DeleteMulti removes multiple keys at once.
func (m *Memory) DeleteMulti(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	for _, key := range keys {
		delete(m.items, key)
	}

	return nil
}

// cleanup periodically removes expired items.
func (m *Memory) cleanup(interval time.Duration) {
	defer close(m.cleanCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.removeExpired()
		}
	}
}

// removeExpired removes all expired items.
func (m *Memory) removeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, it := range m.items {
		if !it.noExpiry && now.After(it.expiresAt) {
			delete(m.items, key)
		}
	}
}
