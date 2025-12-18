// session/session.go
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

// Session represents a user session with key-value data.
type Session struct {
	mu        sync.RWMutex
	id        string
	data      map[string]any
	isNew     bool
	modified  bool
	expiresAt time.Time
}

// ID returns the session ID.
func (s *Session) ID() string {
	return s.id
}

// IsNew returns true if the session was just created.
func (s *Session) IsNew() bool {
	return s.isNew
}

// Get retrieves a value from the session.
func (s *Session) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString retrieves a string value.
func (s *Session) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, _ := val.(string)
	return str
}

// GetInt retrieves an int value.
func (s *Session) GetInt(key string) int {
	val, ok := s.Get(key)
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

// GetBool retrieves a bool value.
func (s *Session) GetBool(key string) bool {
	val, ok := s.Get(key)
	if !ok {
		return false
	}
	b, _ := val.(bool)
	return b
}

// GetTime retrieves a time value.
func (s *Session) GetTime(key string) time.Time {
	val, ok := s.Get(key)
	if !ok {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		return v
	case string:
		t, _ := time.Parse(time.RFC3339, v)
		return t
	default:
		return time.Time{}
	}
}

// Set stores a value in the session.
func (s *Session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	s.modified = true
}

// Delete removes a value from the session.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	s.modified = true
}

// Clear removes all values from the session.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]any)
	s.modified = true
}

// Keys returns all keys in the session.
func (s *Session) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a copy of all session data.
func (s *Session) Values() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[string]any, len(s.data))
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}

// Modified returns true if the session data has been changed.
func (s *Session) Modified() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modified
}

// ExpiresAt returns when the session expires.
func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

// Store defines the interface for session storage backends.
type Store interface {
	// Load retrieves session data by ID.
	// Returns ErrNotFound if the session doesn't exist.
	Load(ctx context.Context, id string) (*SessionData, error)

	// Save stores session data.
	Save(ctx context.Context, data *SessionData) error

	// Delete removes a session by ID.
	Delete(ctx context.Context, id string) error

	// Close releases any resources.
	Close() error
}

// SessionData is the serializable session data stored in backends.
type SessionData struct {
	ID        string         `json:"id"`
	Data      map[string]any `json:"data"`
	ExpiresAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s *SessionData) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *SessionData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

// Common errors
var (
	ErrNotFound       = errors.New("session: not found")
	ErrExpired        = errors.New("session: expired")
	ErrInvalidSession = errors.New("session: invalid session")
)

// generateID creates a cryptographically secure session ID.
func generateID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Manager handles session creation, retrieval, and persistence.
type Manager struct {
	store      Store
	cookieName string
	config     Config
}

// Config configures the session manager.
type Config struct {
	// CookieName is the name of the session cookie.
	// Default: "session_id".
	CookieName string

	// MaxAge is the session lifetime.
	// Default: 24 hours.
	MaxAge time.Duration

	// Path is the cookie path.
	// Default: "/".
	Path string

	// Domain is the cookie domain.
	// Default: "" (current domain).
	Domain string

	// Secure sets the Secure flag on the cookie.
	// Default: true.
	Secure bool

	// HttpOnly sets the HttpOnly flag on the cookie.
	// Default: true.
	HttpOnly bool

	// SameSite sets the SameSite attribute.
	// Default: http.SameSiteLaxMode.
	SameSite http.SameSite

	// IDGenerator generates session IDs.
	// Default: cryptographically secure random ID.
	IDGenerator func() (string, error)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		CookieName:  "session_id",
		MaxAge:      24 * time.Hour,
		Path:        "/",
		Secure:      true,
		HttpOnly:    true,
		SameSite:    http.SameSiteLaxMode,
		IDGenerator: generateID,
	}
}

// NewManager creates a session manager with the given store and config.
func NewManager(store Store, cfg Config) *Manager {
	if cfg.CookieName == "" {
		cfg.CookieName = "session_id"
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 24 * time.Hour
	}
	if cfg.Path == "" {
		cfg.Path = "/"
	}
	if cfg.SameSite == 0 {
		cfg.SameSite = http.SameSiteLaxMode
	}
	if cfg.IDGenerator == nil {
		cfg.IDGenerator = generateID
	}

	return &Manager{
		store:      store,
		cookieName: cfg.CookieName,
		config:     cfg,
	}
}

// Get retrieves the session from the request, creating a new one if needed.
func (m *Manager) Get(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(m.cookieName)
	if err == nil && cookie.Value != "" {
		// Try to load existing session
		data, err := m.store.Load(r.Context(), cookie.Value)
		if err == nil {
			// Check expiration
			if time.Now().After(data.ExpiresAt) {
				// Session expired, delete it
				m.store.Delete(r.Context(), cookie.Value)
			} else {
				return &Session{
					id:        data.ID,
					data:      data.Data,
					isNew:     false,
					expiresAt: data.ExpiresAt,
				}, nil
			}
		}
	}

	// Create new session
	return m.New()
}

// New creates a new session.
func (m *Manager) New() (*Session, error) {
	id, err := m.config.IDGenerator()
	if err != nil {
		return nil, err
	}

	return &Session{
		id:        id,
		data:      make(map[string]any),
		isNew:     true,
		modified:  true,
		expiresAt: time.Now().Add(m.config.MaxAge),
	}, nil
}

// Save persists the session and sets the cookie.
func (m *Manager) Save(w http.ResponseWriter, r *http.Request, session *Session) error {
	session.mu.RLock()
	data := &SessionData{
		ID:        session.id,
		Data:      session.data,
		ExpiresAt: session.expiresAt,
		UpdatedAt: time.Now(),
	}
	if session.isNew {
		data.CreatedAt = time.Now()
	}
	session.mu.RUnlock()

	if err := m.store.Save(r.Context(), data); err != nil {
		return err
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    session.id,
		Path:     m.config.Path,
		Domain:   m.config.Domain,
		MaxAge:   int(m.config.MaxAge.Seconds()),
		Secure:   m.config.Secure,
		HttpOnly: m.config.HttpOnly,
		SameSite: m.config.SameSite,
	})

	return nil
}

// Destroy deletes the session and clears the cookie.
func (m *Manager) Destroy(w http.ResponseWriter, r *http.Request, session *Session) error {
	if err := m.store.Delete(r.Context(), session.id); err != nil {
		return err
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     m.config.Path,
		Domain:   m.config.Domain,
		MaxAge:   -1,
		Secure:   m.config.Secure,
		HttpOnly: m.config.HttpOnly,
		SameSite: m.config.SameSite,
	})

	return nil
}

// Regenerate creates a new session ID while preserving data.
// Use this after authentication to prevent session fixation attacks.
func (m *Manager) Regenerate(w http.ResponseWriter, r *http.Request, session *Session) error {
	oldID := session.id

	// Generate new ID
	newID, err := m.config.IDGenerator()
	if err != nil {
		return err
	}

	// Update session
	session.mu.Lock()
	session.id = newID
	session.modified = true
	session.mu.Unlock()

	// Delete old session from store
	m.store.Delete(r.Context(), oldID)

	// Save with new ID
	return m.Save(w, r, session)
}

// Refresh extends the session expiration.
func (m *Manager) Refresh(w http.ResponseWriter, r *http.Request, session *Session) error {
	session.mu.Lock()
	session.expiresAt = time.Now().Add(m.config.MaxAge)
	session.modified = true
	session.mu.Unlock()

	return m.Save(w, r, session)
}

// Store returns the underlying session store.
func (m *Manager) Store() Store {
	return m.store
}

// Close closes the session manager and underlying store.
func (m *Manager) Close() error {
	return m.store.Close()
}
