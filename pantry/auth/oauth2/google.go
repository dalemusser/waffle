// auth/oauth2/google.go
package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleConfig holds configuration for Google OAuth2 authentication.
type GoogleConfig struct {
	// ClientID is the Google OAuth2 client ID.
	ClientID string

	// ClientSecret is the Google OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Google.
	// Example: "https://myapp.com/auth/google/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, email, profile
	Scopes []string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid.
	// Default: 24 hours.
	SessionDuration int

	// CookieName is the name of the session cookie.
	// Default: "waffle_session".
	CookieName string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	// Default: true.
	CookieSecure bool

	// OnSuccess is called after successful authentication.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// Google creates a new OAuth2 provider configured for Google authentication.
//
// Usage in BuildHandler:
//
//	googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
//	    ClientID:     appCfg.GoogleClientID,
//	    ClientSecret: appCfg.GoogleClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/google/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/google/login", googleAuth.LoginHandler())
//	r.Get("/auth/google/callback", googleAuth.CallbackHandler())
//	r.Get("/auth/google/logout", googleAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(googleAuth.RequireAuth("/auth/google/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
func Google(cfg GoogleConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/google: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/google: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/google: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/google: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/google: StateStore is required")
	}

	// Default scopes for Google
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	providerCfg := &Config{
		ProviderName:  "google",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchGoogleUserInfo,
		SessionStore:  cfg.SessionStore,
		StateStore:    cfg.StateStore,
		CookieName:    cfg.CookieName,
		CookieSecure:  cfg.CookieSecure,
		OnSuccess:     cfg.OnSuccess,
		OnError:       cfg.OnError,
		Logger:        logger,
	}

	if cfg.SessionDuration > 0 {
		providerCfg.SessionDuration = time.Duration(cfg.SessionDuration) * time.Second
	}

	return NewProvider(providerCfg)
}

// googleUserInfo represents the response from Google's userinfo endpoint.
type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// fetchGoogleUserInfo retrieves user information from Google's userinfo endpoint.
func fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &User{
		ID:            info.ID,
		Email:         info.Email,
		EmailVerified: info.EmailVerified,
		Name:          info.Name,
		Picture:       info.Picture,
		Raw: map[string]any{
			"id":             info.ID,
			"email":          info.Email,
			"verified_email": info.EmailVerified,
			"name":           info.Name,
			"given_name":     info.GivenName,
			"family_name":    info.FamilyName,
			"picture":        info.Picture,
			"locale":         info.Locale,
		},
	}, nil
}
