// auth/oauth2/github.go
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
	"golang.org/x/oauth2/github"
)

// GitHubConfig holds configuration for GitHub OAuth2 authentication.
type GitHubConfig struct {
	// ClientID is the GitHub OAuth2 client ID.
	ClientID string

	// ClientSecret is the GitHub OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with GitHub.
	// Example: "https://myapp.com/auth/github/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: user:email, read:user
	Scopes []string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid (in seconds).
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

// GitHub creates a new OAuth2 provider configured for GitHub authentication.
//
// Usage in BuildHandler:
//
//	githubAuth, err := oauth2.GitHub(oauth2.GitHubConfig{
//	    ClientID:     appCfg.GitHubClientID,
//	    ClientSecret: appCfg.GitHubClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/github/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/github/login", githubAuth.LoginHandler())
//	r.Get("/auth/github/callback", githubAuth.CallbackHandler())
//	r.Get("/auth/github/logout", githubAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(githubAuth.RequireAuth("/auth/github/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
func GitHub(cfg GitHubConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/github: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/github: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/github: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/github: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/github: StateStore is required")
	}

	// Default scopes for GitHub
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"user:email",
			"read:user",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     github.Endpoint,
	}

	providerCfg := &Config{
		ProviderName:  "github",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchGitHubUserInfo,
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

// githubUserInfo represents the response from GitHub's user endpoint.
type githubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Bio       string `json:"bio"`
	Company   string `json:"company"`
	Location  string `json:"location"`
}

// githubEmail represents an email from GitHub's emails endpoint.
type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// fetchGitHubUserInfo retrieves user information from GitHub's API.
func fetchGitHubUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Fetch basic user info
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info githubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is not public, fetch from emails endpoint
	email := info.Email
	emailVerified := false
	if email == "" {
		email, emailVerified, _ = fetchGitHubPrimaryEmail(ctx, client)
	} else {
		// Public email is assumed verified
		emailVerified = true
	}

	return &User{
		ID:            fmt.Sprintf("%d", info.ID),
		Email:         email,
		EmailVerified: emailVerified,
		Name:          info.Name,
		Picture:       info.AvatarURL,
		Raw: map[string]any{
			"id":         info.ID,
			"login":      info.Login,
			"name":       info.Name,
			"email":      email,
			"avatar_url": info.AvatarURL,
			"html_url":   info.HTMLURL,
			"bio":        info.Bio,
			"company":    info.Company,
			"location":   info.Location,
		},
		Extra: map[string]string{
			"login":    info.Login,
			"html_url": info.HTMLURL,
		},
	}, nil
}

// fetchGitHubPrimaryEmail retrieves the primary verified email from GitHub.
func fetchGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, bool, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false, err
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true, nil
		}
	}

	// Fall back to any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, true, nil
		}
	}

	// Fall back to any email
	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}

	return "", false, nil
}
