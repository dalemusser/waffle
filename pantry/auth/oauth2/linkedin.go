// auth/oauth2/linkedin.go
package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

// LinkedInConfig holds configuration for LinkedIn OAuth2 authentication.
type LinkedInConfig struct {
	// ClientID is the LinkedIn OAuth2 client ID.
	ClientID string

	// ClientSecret is the LinkedIn OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with LinkedIn.
	// Example: "https://myapp.com/auth/linkedin/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, profile, email
	// For additional access, add: w_member_social (posting)
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

// LinkedIn creates a new OAuth2 provider configured for LinkedIn authentication.
//
// LinkedIn is the world's largest professional network, used for business networking,
// job searching, and professional identity. It provides OAuth2/OIDC authentication
// with access to professional profile data.
//
// Setup in LinkedIn Developer Portal:
//  1. Go to https://www.linkedin.com/developers/apps
//  2. Click "Create app"
//  3. Fill in app details and verify your company page
//  4. Go to "Auth" tab
//  5. Add your redirect URL under "Authorized redirect URLs"
//  6. Copy your Client ID and Client Secret
//  7. Request access to products: "Sign In with LinkedIn using OpenID Connect"
//
// Usage in BuildHandler:
//
//	linkedinAuth, err := oauth2.LinkedIn(oauth2.LinkedInConfig{
//	    ClientID:     appCfg.LinkedInClientID,
//	    ClientSecret: appCfg.LinkedInClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/linkedin/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/linkedin/login", linkedinAuth.LoginHandler())
//	r.Get("/auth/linkedin/callback", linkedinAuth.CallbackHandler())
//	r.Get("/auth/linkedin/logout", linkedinAuth.LogoutHandler())
//
// The User.Extra map contains LinkedIn-specific fields:
//   - "linkedin_id": LinkedIn member ID (sub claim)
//   - "given_name": User's first name
//   - "family_name": User's last name
//   - "locale": User's locale (e.g., "en_US")
func LinkedIn(cfg LinkedInConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/linkedin: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/linkedin: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/linkedin: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/linkedin: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/linkedin: StateStore is required")
	}

	// Default scopes for LinkedIn OpenID Connect
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"profile",
			"email",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     linkedin.Endpoint,
	}

	providerCfg := &Config{
		ProviderName:  "linkedin",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: linkedinFetchUserInfo,
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

// linkedinUserInfo represents the response from LinkedIn's userinfo endpoint.
type linkedinUserInfo struct {
	Sub           string `json:"sub"`            // LinkedIn member ID
	Name          string `json:"name"`           // Full name
	GivenName     string `json:"given_name"`     // First name
	FamilyName    string `json:"family_name"`    // Last name
	Picture       string `json:"picture"`        // Profile picture URL
	Email         string `json:"email"`          // Email address
	EmailVerified bool   `json:"email_verified"` // Email verification status
	Locale        string `json:"locale"`         // User's locale
}

// linkedinFetchUserInfo fetches user information from LinkedIn's userinfo endpoint.
func linkedinFetchUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Use LinkedIn's OpenID Connect userinfo endpoint
	resp, err := client.Get("https://api.linkedin.com/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info linkedinUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Build display name
	name := info.Name
	if name == "" {
		name = strings.TrimSpace(info.GivenName + " " + info.FamilyName)
	}

	return &User{
		ID:            info.Sub,
		Email:         info.Email,
		EmailVerified: info.EmailVerified,
		Name:          name,
		Picture:       info.Picture,
		Raw: map[string]any{
			"sub":            info.Sub,
			"name":           info.Name,
			"given_name":     info.GivenName,
			"family_name":    info.FamilyName,
			"picture":        info.Picture,
			"email":          info.Email,
			"email_verified": info.EmailVerified,
			"locale":         info.Locale,
		},
		Extra: map[string]string{
			"linkedin_id": info.Sub,
			"given_name":  info.GivenName,
			"family_name": info.FamilyName,
			"locale":      info.Locale,
		},
	}, nil
}

// LinkedInAPIClient provides methods to call LinkedIn API endpoints.
// Use this after authentication for additional LinkedIn API operations.
type LinkedInAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewLinkedInAPIClient creates a client for calling LinkedIn API.
// Requires a valid OAuth2 access token.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	linkedinClient := oauth2.NewLinkedInAPIClient(user.AccessToken)
//	profile, err := linkedinClient.GetProfile(r.Context())
func NewLinkedInAPIClient(accessToken string) *LinkedInAPIClient {
	return &LinkedInAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: "https://api.linkedin.com/v2",
	}
}

// LinkedInProfile represents a user's LinkedIn profile.
type LinkedInProfile struct {
	ID             string `json:"id"`
	LocalizedFirstName  string `json:"localizedFirstName"`
	LocalizedLastName   string `json:"localizedLastName"`
	LocalizedHeadline   string `json:"localizedHeadline"`
	VanityName     string `json:"vanityName"` // Custom URL name
}

// GetProfile fetches the user's basic LinkedIn profile.
// Note: This uses the legacy v2 API which may have limited data.
// Most profile data is now accessed via the userinfo endpoint (OIDC).
func (c *LinkedInAPIClient) GetProfile(ctx context.Context) (*LinkedInProfile, error) {
	resp, err := c.client.Get(c.baseURL + "/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var profile LinkedInProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	return &profile, nil
}

// LinkedInEmailAddress represents an email address from LinkedIn.
type LinkedInEmailAddress struct {
	Email string `json:"email"`
}

// linkedinEmailResponse represents the response from the email endpoint.
type linkedinEmailResponse struct {
	Elements []struct {
		Handle      string `json:"handle"`
		HandleTilde struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"handle~"`
	} `json:"elements"`
}

// GetPrimaryEmail fetches the user's primary email address.
// Requires the "email" scope.
// Note: This is typically not needed as email is included in OIDC userinfo.
func (c *LinkedInAPIClient) GetPrimaryEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/emailAddress?q=members&projection=(elements*(handle~))", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var emailResp linkedinEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		return "", fmt.Errorf("failed to decode email response: %w", err)
	}

	if len(emailResp.Elements) > 0 {
		return emailResp.Elements[0].HandleTilde.EmailAddress, nil
	}

	return "", errors.New("no email address found")
}

// LinkedInShare represents a share/post on LinkedIn.
type LinkedInShare struct {
	Author         string                 `json:"author"`
	LifecycleState string                 `json:"lifecycleState"`
	SpecificContent LinkedInShareContent  `json:"specificContent"`
	Visibility     LinkedInShareVisibility `json:"visibility"`
}

// LinkedInShareContent represents the content of a LinkedIn share.
type LinkedInShareContent struct {
	ShareCommentary LinkedInShareCommentary `json:"com.linkedin.ugc.ShareContent"`
}

// LinkedInShareCommentary represents the text content of a share.
type LinkedInShareCommentary struct {
	ShareCommentary struct {
		Text string `json:"text"`
	} `json:"shareCommentary"`
	ShareMediaCategory string `json:"shareMediaCategory"`
}

// LinkedInShareVisibility represents who can see a share.
type LinkedInShareVisibility struct {
	MemberNetworkVisibility string `json:"com.linkedin.ugc.MemberNetworkVisibility"`
}

// CreateTextShare creates a text-only share on LinkedIn.
// Requires the "w_member_social" scope.
//
// Note: LinkedIn's posting API is complex and has rate limits.
// This is a simplified implementation for basic text posts.
func (c *LinkedInAPIClient) CreateTextShare(ctx context.Context, personURN, text string) error {
	share := map[string]any{
		"author":         personURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": map[string]any{
				"shareCommentary": map[string]any{
					"text": text,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]any{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	body, err := json.Marshal(share)
	if err != nil {
		return fmt.Errorf("failed to marshal share: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linkedin.com/v2/ugcPosts", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetLinkedInMemberURN returns the member URN for API calls.
// The URN format is "urn:li:person:{id}".
func GetLinkedInMemberURN(user *User) string {
	return "urn:li:person:" + user.ID
}

// GetLinkedInProfileURL returns the user's LinkedIn profile URL.
// Note: This constructs a URL based on the member ID, which may redirect.
func GetLinkedInProfileURL(user *User) string {
	return "https://www.linkedin.com/in/" + user.ID
}
