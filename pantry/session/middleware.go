// session/middleware.go
package session

import (
	"context"
	"net/http"
)

// contextKey is the type for context keys.
type contextKey string

const sessionContextKey contextKey = "session"

// Middleware returns HTTP middleware that loads sessions automatically.
// The session is available via FromContext(r.Context()).
func Middleware(m *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := m.Get(r)
			if err != nil {
				// Create new session on error
				session, _ = m.New()
			}

			// Add session to context
			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			r = r.WithContext(ctx)

			// Create response wrapper to save session
			sw := &sessionWriter{
				ResponseWriter: w,
				request:        r,
				session:        session,
				manager:        m,
				written:        false,
			}

			next.ServeHTTP(sw, r)

			// Save session if modified and not already written
			if !sw.written && session.Modified() {
				m.Save(w, r, session)
			}
		})
	}
}

// FromContext retrieves the session from the request context.
// Returns nil if no session is in context (middleware not used).
func FromContext(ctx context.Context) *Session {
	session, _ := ctx.Value(sessionContextKey).(*Session)
	return session
}

// MustFromContext retrieves the session from context, panicking if not found.
func MustFromContext(ctx context.Context) *Session {
	session := FromContext(ctx)
	if session == nil {
		panic("session: no session in context (middleware not used?)")
	}
	return session
}

// sessionWriter wraps ResponseWriter to save session before writing.
type sessionWriter struct {
	http.ResponseWriter
	request *http.Request
	session *Session
	manager *Manager
	written bool
}

func (sw *sessionWriter) WriteHeader(code int) {
	if !sw.written && sw.session.Modified() {
		sw.manager.Save(sw.ResponseWriter, sw.request, sw.session)
		sw.written = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *sessionWriter) Write(b []byte) (int, error) {
	if !sw.written && sw.session.Modified() {
		sw.manager.Save(sw.ResponseWriter, sw.request, sw.session)
		sw.written = true
	}
	return sw.ResponseWriter.Write(b)
}

// RequireSession middleware returns 401 if no valid session exists.
func RequireSession(m *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := FromContext(r.Context())
			if session == nil || session.IsNew() {
				http.Error(w, "session required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireKey middleware returns 401 if the session doesn't have a specific key.
func RequireKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := FromContext(r.Context())
			if session == nil {
				http.Error(w, "session required", http.StatusUnauthorized)
				return
			}

			if _, ok := session.Get(key); !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Flash stores a message that will be deleted after being read.
// Useful for one-time messages like "Login successful".
func Flash(session *Session, key string, value any) {
	session.Set("_flash_"+key, value)
}

// GetFlash retrieves and removes a flash message.
func GetFlash(session *Session, key string) (any, bool) {
	flashKey := "_flash_" + key
	value, ok := session.Get(flashKey)
	if ok {
		session.Delete(flashKey)
	}
	return value, ok
}

// GetFlashString retrieves and removes a flash message as a string.
func GetFlashString(session *Session, key string) string {
	value, ok := GetFlash(session, key)
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}
