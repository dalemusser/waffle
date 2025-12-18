// session/memory.go
package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore implements in-memory session storage.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
	stopCh   chan struct{}
	cleanCh  chan struct{}
}

// MemoryStoreConfig configures the memory store.
type MemoryStoreConfig struct {
	// CleanupInterval is how often to remove expired sessions.
	// Default: 10 minutes.
	CleanupInterval time.Duration
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithConfig(MemoryStoreConfig{
		CleanupInterval: 10 * time.Minute,
	})
}

// NewMemoryStoreWithConfig creates a memory store with custom configuration.
func NewMemoryStoreWithConfig(cfg MemoryStoreConfig) *MemoryStore {
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 10 * time.Minute
	}

	s := &MemoryStore{
		sessions: make(map[string]*SessionData),
		stopCh:   make(chan struct{}),
		cleanCh:  make(chan struct{}),
	}

	go s.cleanup(cfg.CleanupInterval)

	return s
}

// Load retrieves session data by ID.
func (s *MemoryStore) Load(ctx context.Context, id string) (*SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.sessions[id]
	if !exists {
		return nil, ErrNotFound
	}

	if time.Now().After(data.ExpiresAt) {
		return nil, ErrExpired
	}

	// Return a copy to prevent mutation
	return copySessionData(data), nil
}

// Save stores session data.
func (s *MemoryStore) Save(ctx context.Context, data *SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy
	s.sessions[data.ID] = copySessionData(data)
	return nil
}

// Delete removes a session by ID.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// Close stops the cleanup goroutine.
func (s *MemoryStore) Close() error {
	close(s.stopCh)
	<-s.cleanCh
	return nil
}

// Size returns the number of sessions.
func (s *MemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// cleanup periodically removes expired sessions.
func (s *MemoryStore) cleanup(interval time.Duration) {
	defer close(s.cleanCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.removeExpired()
		}
	}
}

// removeExpired removes all expired sessions.
func (s *MemoryStore) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, data := range s.sessions {
		if now.After(data.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

// copySessionData creates a deep copy of session data.
func copySessionData(data *SessionData) *SessionData {
	dataCopy := make(map[string]any, len(data.Data))
	for k, v := range data.Data {
		dataCopy[k] = v
	}

	return &SessionData{
		ID:        data.ID,
		Data:      dataCopy,
		ExpiresAt: data.ExpiresAt,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}
}
