// auth/oauth2/oauth2.go
package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// User represents the authenticated user information retrieved from an OAuth2 provider.
type User struct {
	ID            string            `json:"id"`             // Provider-specific user ID
	Email         string            `json:"email"`          // User's email address
	EmailVerified bool              `json:"email_verified"` // Whether email is verified by provider
	Name          string            `json:"name"`           // Display name
	Picture       string            `json:"picture"`        // Profile picture URL
	Provider      string            `json:"provider"`       // OAuth provider name (e.g., "google", "github")
	Raw           map[string]any    `json:"raw"`            // Raw claims from the provider
	AccessToken   string            `json:"-"`              // OAuth access token (not serialized)
	RefreshToken  string            `json:"-"`              // OAuth refresh token (not serialized)
	TokenExpiry   time.Time         `json:"token_expiry"`   // When the access token expires
	Extra         map[string]string `json:"extra"`          // App-specific metadata
}

// Session represents a user session stored by the SessionStore.
type Session struct {
	ID        string    `json:"id"`         // Unique session ID
	User      User      `json:"user"`       // Authenticated user info
	CreatedAt time.Time `json:"created_at"` // When the session was created
	ExpiresAt time.Time `json:"expires_at"` // When the session expires
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// SessionStore defines the interface for persisting OAuth2 sessions.
// Implementations can use Redis, MongoDB, in-memory storage, etc.
type SessionStore interface {
	// Save stores a session. The session ID should be used as the key.
	Save(ctx context.Context, session *Session) error

	// Get retrieves a session by ID. Returns nil if not found.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session by ID.
	Delete(ctx context.Context, sessionID string) error
}

// StateStore defines the interface for storing OAuth2 state parameters.
// State is used to prevent CSRF attacks during the OAuth flow.
type StateStore interface {
	// Save stores a state value with an expiration time.
	Save(ctx context.Context, state string, expiresAt time.Time) error

	// Validate checks if a state exists and removes it. Returns true if valid.
	Validate(ctx context.Context, state string) (bool, error)
}

// UserInfoFetcher is a function that retrieves user information from an OAuth2 provider.
// Each provider (Google, GitHub, etc.) has its own endpoint and response format.
type UserInfoFetcher func(ctx context.Context, token *oauth2.Token) (*User, error)

// Config holds the configuration for an OAuth2 provider.
type Config struct {
	// ProviderName identifies this provider (e.g., "google", "github").
	ProviderName string

	// OAuth2Config is the standard oauth2 configuration.
	OAuth2Config *oauth2.Config

	// FetchUserInfo retrieves user information after successful authentication.
	FetchUserInfo UserInfoFetcher

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid.
	// Default: 24 hours.
	SessionDuration time.Duration

	// StateDuration controls how long OAuth2 state remains valid.
	// Default: 10 minutes.
	StateDuration time.Duration

	// CookieName is the name of the session cookie.
	// Default: "waffle_session".
	CookieName string

	// CookiePath sets the path for the session cookie.
	// Default: "/".
	CookiePath string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	// Default: true.
	CookieSecure bool

	// CookieSameSite sets the SameSite attribute for cookies.
	// Default: http.SameSiteLaxMode.
	CookieSameSite http.SameSite

	// OnSuccess is called after successful authentication.
	// Use this to redirect users or perform post-auth logic.
	// If nil, redirects to "/" by default.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	// If nil, returns a generic error response.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// Provider handles OAuth2 authentication for a specific provider.
type Provider struct {
	config *Config
	logger *zap.Logger
}

// NewProvider creates a new OAuth2 provider with the given configuration.
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg.OAuth2Config == nil {
		return nil, errors.New("oauth2: OAuth2Config is required")
	}
	if cfg.FetchUserInfo == nil {
		return nil, errors.New("oauth2: FetchUserInfo is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2: StateStore is required")
	}
	if cfg.ProviderName == "" {
		cfg.ProviderName = "oauth2"
	}

	// Apply defaults
	if cfg.SessionDuration == 0 {
		cfg.SessionDuration = 24 * time.Hour
	}
	if cfg.StateDuration == 0 {
		cfg.StateDuration = 10 * time.Minute
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "waffle_session"
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/"
	}
	if cfg.CookieSameSite == 0 {
		cfg.CookieSameSite = http.SameSiteLaxMode
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Provider{
		config: cfg,
		logger: logger,
	}, nil
}

// LoginHandler returns an HTTP handler that initiates the OAuth2 flow.
// It generates a state parameter, stores it, and redirects to the provider.
func (p *Provider) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := generateState()
		if err != nil {
			p.logger.Error("failed to generate OAuth2 state",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to generate state: %w", err))
			return
		}

		expiresAt := time.Now().Add(p.config.StateDuration)
		if err := p.config.StateStore.Save(r.Context(), state, expiresAt); err != nil {
			p.logger.Error("failed to save OAuth2 state",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to save state: %w", err))
			return
		}

		url := p.config.OAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

		p.logger.Debug("initiating OAuth2 flow",
			zap.String("provider", p.config.ProviderName),
			zap.String("redirect_url", url),
		)

		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// CallbackHandler returns an HTTP handler that processes the OAuth2 callback.
// It validates the state, exchanges the code for tokens, fetches user info,
// creates a session, and sets the session cookie.
func (p *Provider) CallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check for errors from the provider
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			p.logger.Warn("OAuth2 provider returned error",
				zap.String("provider", p.config.ProviderName),
				zap.String("error", errParam),
				zap.String("description", errDesc),
			)
			p.handleError(w, r, fmt.Errorf("provider error: %s - %s", errParam, errDesc))
			return
		}

		// Validate state parameter
		state := r.URL.Query().Get("state")
		if state == "" {
			p.logger.Warn("missing OAuth2 state parameter",
				zap.String("provider", p.config.ProviderName),
			)
			p.handleError(w, r, errors.New("missing state parameter"))
			return
		}

		valid, err := p.config.StateStore.Validate(ctx, state)
		if err != nil {
			p.logger.Error("failed to validate OAuth2 state",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to validate state: %w", err))
			return
		}
		if !valid {
			p.logger.Warn("invalid OAuth2 state parameter",
				zap.String("provider", p.config.ProviderName),
			)
			p.handleError(w, r, errors.New("invalid or expired state"))
			return
		}

		// Exchange code for token
		code := r.URL.Query().Get("code")
		if code == "" {
			p.logger.Warn("missing OAuth2 code parameter",
				zap.String("provider", p.config.ProviderName),
			)
			p.handleError(w, r, errors.New("missing code parameter"))
			return
		}

		token, err := p.config.OAuth2Config.Exchange(ctx, code)
		if err != nil {
			p.logger.Error("failed to exchange OAuth2 code",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to exchange code: %w", err))
			return
		}

		// Fetch user info
		user, err := p.config.FetchUserInfo(ctx, token)
		if err != nil {
			p.logger.Error("failed to fetch user info",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to fetch user info: %w", err))
			return
		}

		user.Provider = p.config.ProviderName
		user.AccessToken = token.AccessToken
		user.RefreshToken = token.RefreshToken
		user.TokenExpiry = token.Expiry

		// Create session
		sessionID, err := generateSessionID()
		if err != nil {
			p.logger.Error("failed to generate session ID",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to generate session: %w", err))
			return
		}

		session := &Session{
			ID:        sessionID,
			User:      *user,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(p.config.SessionDuration),
		}

		if err := p.config.SessionStore.Save(ctx, session); err != nil {
			p.logger.Error("failed to save session",
				zap.String("provider", p.config.ProviderName),
				zap.Error(err),
			)
			p.handleError(w, r, fmt.Errorf("failed to save session: %w", err))
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     p.config.CookieName,
			Value:    sessionID,
			Path:     p.config.CookiePath,
			Secure:   p.config.CookieSecure,
			HttpOnly: true,
			SameSite: p.config.CookieSameSite,
			MaxAge:   int(p.config.SessionDuration.Seconds()),
		})

		p.logger.Info("OAuth2 authentication successful",
			zap.String("provider", p.config.ProviderName),
			zap.String("user_id", user.ID),
			zap.String("email", user.Email),
		)

		// Call success handler
		if p.config.OnSuccess != nil {
			p.config.OnSuccess(w, r, user)
		} else {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}
}

// LogoutHandler returns an HTTP handler that ends the user session.
func (p *Provider) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(p.config.CookieName)
		if err == nil && cookie.Value != "" {
			// Delete the session from the store
			if err := p.config.SessionStore.Delete(r.Context(), cookie.Value); err != nil {
				p.logger.Warn("failed to delete session",
					zap.String("provider", p.config.ProviderName),
					zap.Error(err),
				)
			}
		}

		// Clear the cookie
		http.SetCookie(w, &http.Cookie{
			Name:     p.config.CookieName,
			Value:    "",
			Path:     p.config.CookiePath,
			Secure:   p.config.CookieSecure,
			HttpOnly: true,
			SameSite: p.config.CookieSameSite,
			MaxAge:   -1,
		})

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

// RequireAuth returns middleware that requires a valid session.
// If the user is not authenticated, they are redirected to the login URL.
func (p *Provider) RequireAuth(loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := p.GetSession(r)
			if err != nil || session == nil || session.IsExpired() {
				http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
				return
			}

			// Add user to context
			ctx := ContextWithUser(r.Context(), &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthJSON returns middleware that requires a valid session.
// If the user is not authenticated, returns a 401 JSON response.
func (p *Provider) RequireAuthJSON() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := p.GetSession(r)
			if err != nil || session == nil || session.IsExpired() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}

			// Add user to context
			ctx := ContextWithUser(r.Context(), &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession retrieves the session from the request cookie.
func (p *Provider) GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil {
		return nil, err
	}
	if cookie.Value == "" {
		return nil, errors.New("empty session cookie")
	}
	return p.config.SessionStore.Get(r.Context(), cookie.Value)
}

// handleError calls the configured error handler or returns a generic error.
func (p *Provider) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if p.config.OnError != nil {
		p.config.OnError(w, r, err)
		return
	}
	http.Error(w, "authentication failed", http.StatusUnauthorized)
}

// generateState creates a cryptographically secure random state string.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateSessionID creates a cryptographically secure random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Context key for storing user in request context.
type contextKey string

const userContextKey contextKey = "oauth2_user"

// ContextWithUser returns a new context with the user attached.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from the context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}
