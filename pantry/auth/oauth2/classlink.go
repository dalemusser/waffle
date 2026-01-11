// auth/oauth2/classlink.go
package oauth2

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// ClassLink OAuth2 endpoints.
var classLinkEndpoint = oauth2.Endpoint{
	AuthURL:  "https://launchpad.classlink.com/oauth2/v2/auth",
	TokenURL: "https://launchpad.classlink.com/oauth2/v2/token",
}

// ClassLinkRole represents the role of a user in ClassLink.
type ClassLinkRole string

const (
	ClassLinkRoleStudent      ClassLinkRole = "student"
	ClassLinkRoleTeacher      ClassLinkRole = "teacher"
	ClassLinkRoleAdmin        ClassLinkRole = "administrator"
	ClassLinkRoleParent       ClassLinkRole = "parent"
	ClassLinkRoleAide         ClassLinkRole = "aide"
	ClassLinkRoleOther        ClassLinkRole = "other"
)

// ClassLinkConfig holds configuration for ClassLink OAuth2 authentication.
type ClassLinkConfig struct {
	// ClientID is the ClassLink OAuth2 client ID.
	ClientID string

	// ClientSecret is the ClassLink OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with ClassLink.
	// Example: "https://school.app/auth/classlink/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: profile
	// Other options: oneroster (for rostering data), full_profile
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

// ClassLink creates a new OAuth2 provider configured for ClassLink authentication.
//
// ClassLink is a popular SSO provider for K-12 schools. It provides user information
// including role (student, teacher, administrator), tenant (district) info, and SIS IDs.
//
// Usage in BuildHandler:
//
//	classLinkAuth, err := oauth2.ClassLink(oauth2.ClassLinkConfig{
//	    ClientID:     appCfg.ClassLinkClientID,
//	    ClientSecret: appCfg.ClassLinkClientSecret,
//	    RedirectURL:  "https://school.app/auth/classlink/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/classlink/login", classLinkAuth.LoginHandler())
//	r.Get("/auth/classlink/callback", classLinkAuth.CallbackHandler())
//	r.Get("/auth/classlink/logout", classLinkAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(classLinkAuth.RequireAuth("/auth/classlink/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains ClassLink-specific fields:
//   - "role": student, teacher, administrator, parent, aide, or other
//   - "tenant_id": ClassLink tenant (district) ID
//   - "sourced_id": OneRoster SourcedId (SIS ID)
//   - "login_id": User's login identifier
//   - "building_id": School/building ID (if available)
func ClassLink(cfg ClassLinkConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/classlink: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/classlink: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/classlink: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/classlink: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/classlink: StateStore is required")
	}

	// Default scopes for ClassLink
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"profile",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     classLinkEndpoint,
	}

	providerCfg := &Config{
		ProviderName:  "classlink",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchClassLinkUserInfo,
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

// classLinkUserInfo represents the response from ClassLink's user info endpoint.
type classLinkUserInfo struct {
	UserId      string `json:"UserId"`
	LoginId     string `json:"LoginId"`
	TenantId    string `json:"TenantId"`
	Email       string `json:"Email"`
	FirstName   string `json:"FirstName"`
	LastName    string `json:"LastName"`
	DisplayName string `json:"DisplayName"`
	Role        string `json:"Role"`
	SourcedId   string `json:"SourcedId"`
	BuildingId  string `json:"BuildingId"`
	// Additional fields that may be present
	Tenant      string `json:"Tenant"`
	ImageUrl    string `json:"ImageUrl"`
	PhoneNumber string `json:"PhoneNumber"`
}

// fetchClassLinkUserInfo retrieves user information from ClassLink's API.
func fetchClassLinkUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Fetch user info from ClassLink's info endpoint
	resp, err := client.Get("https://nodeapi.classlink.com/v2/my/info")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info classLinkUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Build display name if not provided
	name := info.DisplayName
	if name == "" {
		name = info.FirstName
		if info.LastName != "" {
			name += " " + info.LastName
		}
	}

	// Normalize role to lowercase
	role := normalizeClassLinkRole(info.Role)

	return &User{
		ID:            info.UserId,
		Email:         info.Email,
		EmailVerified: info.Email != "", // ClassLink emails are verified
		Name:          name,
		Picture:       info.ImageUrl,
		Raw: map[string]any{
			"UserId":      info.UserId,
			"LoginId":     info.LoginId,
			"TenantId":    info.TenantId,
			"Email":       info.Email,
			"FirstName":   info.FirstName,
			"LastName":    info.LastName,
			"DisplayName": info.DisplayName,
			"Role":        info.Role,
			"SourcedId":   info.SourcedId,
			"BuildingId":  info.BuildingId,
			"Tenant":      info.Tenant,
			"ImageUrl":    info.ImageUrl,
		},
		Extra: map[string]string{
			"role":        role,
			"tenant_id":   info.TenantId,
			"sourced_id":  info.SourcedId,
			"login_id":    info.LoginId,
			"building_id": info.BuildingId,
			"first_name":  info.FirstName,
			"last_name":   info.LastName,
		},
	}, nil
}

// normalizeClassLinkRole converts ClassLink role to lowercase standard form.
func normalizeClassLinkRole(role string) string {
	switch role {
	case "Student", "student":
		return string(ClassLinkRoleStudent)
	case "Teacher", "teacher":
		return string(ClassLinkRoleTeacher)
	case "Administrator", "administrator", "Admin", "admin":
		return string(ClassLinkRoleAdmin)
	case "Parent", "parent", "Guardian", "guardian":
		return string(ClassLinkRoleParent)
	case "Aide", "aide":
		return string(ClassLinkRoleAide)
	default:
		return string(ClassLinkRoleOther)
	}
}

// IsClassLinkStudent returns true if the user is a student.
func IsClassLinkStudent(user *User) bool {
	return user.Extra["role"] == string(ClassLinkRoleStudent)
}

// IsClassLinkTeacher returns true if the user is a teacher.
func IsClassLinkTeacher(user *User) bool {
	return user.Extra["role"] == string(ClassLinkRoleTeacher)
}

// IsClassLinkAdmin returns true if the user is an administrator.
func IsClassLinkAdmin(user *User) bool {
	return user.Extra["role"] == string(ClassLinkRoleAdmin)
}

// GetClassLinkRole returns the ClassLink role from the User.
func GetClassLinkRole(user *User) ClassLinkRole {
	return ClassLinkRole(user.Extra["role"])
}
