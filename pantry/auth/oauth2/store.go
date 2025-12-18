// auth/oauth2/store.go
package oauth2

import (
	"context"
	"sync"
	"time"
)

// MemorySessionStore is an in-memory implementation of SessionStore.
// Suitable for development and single-instance deployments.
// For production with multiple instances, use Redis or another distributed store.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// Save stores a session in memory.
func (s *MemorySessionStore) Save(_ context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID. Returns nil if not found or expired.
func (s *MemorySessionStore) Get(_ context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	if session.IsExpired() {
		// Don't delete here (read lock), let cleanup handle it
		return nil, nil
	}
	return session, nil
}

// Delete removes a session by ID.
func (s *MemorySessionStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

// Cleanup removes all expired sessions. Call this periodically.
func (s *MemorySessionStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for id, session := range s.sessions {
		if session.IsExpired() {
			delete(s.sessions, id)
			count++
		}
	}
	return count
}

// StartCleanupTask starts a background goroutine that periodically cleans up
// expired sessions. Returns a cancel function to stop the cleanup task.
func (s *MemorySessionStore) StartCleanupTask(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

// MemoryStateStore is an in-memory implementation of StateStore.
// Suitable for development and single-instance deployments.
// For production with multiple instances, use Redis or another distributed store.
type MemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]time.Time
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		states: make(map[string]time.Time),
	}
}

// Save stores a state value with an expiration time.
func (s *MemoryStateStore) Save(_ context.Context, state string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state] = expiresAt
	return nil
}

// Validate checks if a state exists and removes it. Returns true if valid.
func (s *MemoryStateStore) Validate(_ context.Context, state string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.states[state]
	if !ok {
		return false, nil
	}
	// Always delete the state (one-time use)
	delete(s.states, state)
	// Check if expired
	if time.Now().After(expiresAt) {
		return false, nil
	}
	return true, nil
}

// Cleanup removes all expired states. Call this periodically.
func (s *MemoryStateStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for state, expiresAt := range s.states {
		if now.After(expiresAt) {
			delete(s.states, state)
			count++
		}
	}
	return count
}

// StartCleanupTask starts a background goroutine that periodically cleans up
// expired states. Returns a cancel function to stop the cleanup task.
func (s *MemoryStateStore) StartCleanupTask(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}
