// auth/oauth2/okta.go
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
)

// OktaConfig holds configuration for Okta OAuth2 authentication.
type OktaConfig struct {
	// ClientID is the Okta OAuth2 client ID.
	ClientID string

	// ClientSecret is the Okta OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Okta.
	// Example: "https://myapp.com/auth/okta/callback"
	RedirectURL string

	// Domain is your Okta domain.
	// Example: "dev-123456.okta.com" or "mycompany.okta.com"
	// Do NOT include https:// prefix.
	Domain string

	// AuthorizationServerID is the Okta authorization server ID.
	// Use "default" for the default authorization server.
	// For custom authorization servers, use the server ID.
	// Leave empty to use the org authorization server (no /oauth2 path).
	AuthorizationServerID string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, profile, email
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

// Okta creates a new OAuth2 provider configured for Okta authentication.
//
// Okta is a leading enterprise identity provider used for workforce identity,
// customer identity (CIAM), and B2B scenarios. It supports OIDC/OAuth2.
//
// Setup in Okta Admin Console:
//  1. Go to Applications → Create App Integration
//  2. Select "OIDC - OpenID Connect" and "Web Application"
//  3. Configure grant types (Authorization Code is recommended)
//  4. Set Sign-in redirect URIs to your callback URL
//  5. Note your Client ID and Client Secret
//
// Usage in BuildHandler:
//
//	oktaAuth, err := oauth2.Okta(oauth2.OktaConfig{
//	    ClientID:              appCfg.OktaClientID,
//	    ClientSecret:          appCfg.OktaClientSecret,
//	    RedirectURL:           "https://myapp.com/auth/okta/callback",
//	    Domain:                "dev-123456.okta.com",
//	    AuthorizationServerID: "default", // or custom server ID
//	    SessionStore:          mySessionStore,
//	    StateStore:            myStateStore,
//	}, logger)
//
//	r.Get("/auth/okta/login", oktaAuth.LoginHandler())
//	r.Get("/auth/okta/callback", oktaAuth.CallbackHandler())
//	r.Get("/auth/okta/logout", oktaAuth.LogoutHandler())
//
// The User.Extra map contains Okta-specific fields:
//   - "okta_user_id": Okta user ID (sub claim)
//   - "preferred_username": User's preferred username
//   - "locale": User's locale
//   - "zoneinfo": User's time zone
//   - "groups": Comma-separated list of groups (if groups claim is enabled)
//   - "first_name": User's first name
//   - "last_name": User's last name
func Okta(cfg OktaConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/okta: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/okta: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/okta: RedirectURL is required")
	}
	if cfg.Domain == "" {
		return nil, errors.New("oauth2/okta: Domain is required (e.g., dev-123456.okta.com)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/okta: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/okta: StateStore is required")
	}

	// Clean up domain - remove protocol if present
	domain := cfg.Domain
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	// Build the base URL for the authorization server
	var baseURL string
	if cfg.AuthorizationServerID == "" {
		// Org authorization server (no /oauth2 path)
		baseURL = fmt.Sprintf("https://%s", domain)
	} else {
		// Custom or default authorization server
		baseURL = fmt.Sprintf("https://%s/oauth2/%s", domain, cfg.AuthorizationServerID)
	}

	endpoint := oauth2.Endpoint{
		AuthURL:  baseURL + "/v1/authorize",
		TokenURL: baseURL + "/v1/token",
	}

	// Default scopes for Okta
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
		Endpoint:     endpoint,
	}

	// Create a fetcher that uses the userinfo endpoint
	fetchUserInfo := createOktaUserInfoFetcher(baseURL)

	providerCfg := &Config{
		ProviderName:  "okta",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchUserInfo,
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

// oktaUserInfo represents the response from Okta's userinfo endpoint.
type oktaUserInfo struct {
	Sub               string   `json:"sub"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	MiddleName        string   `json:"middle_name"`
	Nickname          string   `json:"nickname"`
	PreferredUsername string   `json:"preferred_username"`
	Profile           string   `json:"profile"`
	Picture           string   `json:"picture"`
	Website           string   `json:"website"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Gender            string   `json:"gender"`
	Birthdate         string   `json:"birthdate"`
	Zoneinfo          string   `json:"zoneinfo"`
	Locale            string   `json:"locale"`
	PhoneNumber       string   `json:"phone_number"`
	Address           any      `json:"address"`
	UpdatedAt         int64    `json:"updated_at"`
	Groups            []string `json:"groups"` // If groups scope/claim is configured
}

// createOktaUserInfoFetcher creates a UserInfoFetcher for Okta.
func createOktaUserInfoFetcher(baseURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from Okta's userinfo endpoint
		resp, err := client.Get(baseURL + "/v1/userinfo")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info oktaUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		// Build display name
		name := info.Name
		if name == "" {
			name = info.GivenName
			if info.FamilyName != "" {
				name += " " + info.FamilyName
			}
		}

		// Build groups string
		groupsStr := ""
		if len(info.Groups) > 0 {
			groupsStr = strings.Join(info.Groups, ",")
		}

		return &User{
			ID:            info.Sub,
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
			Name:          name,
			Picture:       info.Picture,
			Raw: map[string]any{
				"sub":                info.Sub,
				"name":               info.Name,
				"given_name":         info.GivenName,
				"family_name":        info.FamilyName,
				"middle_name":        info.MiddleName,
				"nickname":           info.Nickname,
				"preferred_username": info.PreferredUsername,
				"profile":            info.Profile,
				"picture":            info.Picture,
				"website":            info.Website,
				"email":              info.Email,
				"email_verified":     info.EmailVerified,
				"gender":             info.Gender,
				"birthdate":          info.Birthdate,
				"zoneinfo":           info.Zoneinfo,
				"locale":             info.Locale,
				"phone_number":       info.PhoneNumber,
				"address":            info.Address,
				"updated_at":         info.UpdatedAt,
				"groups":             info.Groups,
			},
			Extra: map[string]string{
				"okta_user_id":       info.Sub,
				"preferred_username": info.PreferredUsername,
				"locale":             info.Locale,
				"zoneinfo":           info.Zoneinfo,
				"groups":             groupsStr,
				"first_name":         info.GivenName,
				"last_name":          info.FamilyName,
				"phone_number":       info.PhoneNumber,
				"nickname":           info.Nickname,
			},
		}, nil
	}
}

// OktaAPIClient provides methods to call Okta API endpoints.
// Use this after authentication to fetch additional user data, groups, etc.
type OktaAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewOktaAPIClient creates a client for calling Okta API.
// Requires a valid OAuth2 access token and the Okta domain.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	oktaClient := oauth2.NewOktaAPIClient(user.AccessToken, "dev-123456.okta.com")
//	groups, err := oktaClient.GetUserGroups(r.Context())
func NewOktaAPIClient(accessToken, domain string) *OktaAPIClient {
	// Clean up domain
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	return &OktaAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: fmt.Sprintf("https://%s", domain),
	}
}

// OktaGroup represents a group in Okta.
type OktaGroup struct {
	ID          string         `json:"id"`
	Created     string         `json:"created"`
	LastUpdated string         `json:"lastUpdated"`
	Type        string         `json:"type"` // OKTA_GROUP, APP_GROUP, BUILT_IN
	Profile     OktaGroupProfile `json:"profile"`
}

// OktaGroupProfile contains the group's profile information.
type OktaGroupProfile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetUserGroups fetches the groups for the current user.
// Requires the groups scope and appropriate claims configuration in Okta.
func (c *OktaAPIClient) GetUserGroups(ctx context.Context, userID string) ([]OktaGroup, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/groups", c.baseURL, userID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var groups []OktaGroup
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	return groups, nil
}

// OktaUser represents a user in Okta.
type OktaUser struct {
	ID          string         `json:"id"`
	Status      string         `json:"status"` // ACTIVE, STAGED, PROVISIONED, etc.
	Created     string         `json:"created"`
	Activated   string         `json:"activated"`
	LastLogin   string         `json:"lastLogin"`
	LastUpdated string         `json:"lastUpdated"`
	Profile     OktaUserProfile `json:"profile"`
}

// OktaUserProfile contains the user's profile information.
type OktaUserProfile struct {
	Login       string `json:"login"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	NickName    string `json:"nickName"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	SecondEmail string `json:"secondEmail"`
	MobilePhone string `json:"mobilePhone"`
	PrimaryPhone string `json:"primaryPhone"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Manager     string `json:"manager"`
	ManagerId   string `json:"managerId"`
	Organization string `json:"organization"`
	Division    string `json:"division"`
	EmployeeNumber string `json:"employeeNumber"`
	CostCenter  string `json:"costCenter"`
}

// GetCurrentUser fetches the current user's full profile from Okta.
func (c *OktaAPIClient) GetCurrentUser(ctx context.Context) (*OktaUser, error) {
	resp, err := c.client.Get(c.baseURL + "/api/v1/users/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user OktaUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

// GetUser fetches a user by ID from Okta.
func (c *OktaAPIClient) GetUser(ctx context.Context, userID string) (*OktaUser, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s", c.baseURL, userID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user OktaUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

// GetOktaGroups returns the user's Okta groups from the User.Extra map.
// Groups are returned as a slice of strings.
func GetOktaGroups(user *User) []string {
	groupsStr := user.Extra["groups"]
	if groupsStr == "" {
		return nil
	}
	return strings.Split(groupsStr, ",")
}

// IsInOktaGroup checks if the user is a member of a specific Okta group.
func IsInOktaGroup(user *User, groupName string) bool {
	groups := GetOktaGroups(user)
	for _, g := range groups {
		if g == groupName {
			return true
		}
	}
	return false
}

// HasAnyOktaGroup checks if the user is a member of any of the specified groups.
func HasAnyOktaGroup(user *User, groupNames ...string) bool {
	groups := GetOktaGroups(user)
	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[g] = true
	}
	for _, name := range groupNames {
		if groupSet[name] {
			return true
		}
	}
	return false
}

// HasAllOktaGroups checks if the user is a member of all specified groups.
func HasAllOktaGroups(user *User, groupNames ...string) bool {
	groups := GetOktaGroups(user)
	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[g] = true
	}
	for _, name := range groupNames {
		if !groupSet[name] {
			return false
		}
	}
	return true
}
