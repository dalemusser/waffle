// auth/oauth2/clever.go
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

// Clever OAuth2 endpoints.
var cleverEndpoint = oauth2.Endpoint{
	AuthURL:  "https://clever.com/oauth/authorize",
	TokenURL: "https://clever.com/oauth/tokens",
}

// CleverUserType represents the type of user in Clever.
type CleverUserType string

const (
	CleverUserTypeStudent       CleverUserType = "student"
	CleverUserTypeTeacher       CleverUserType = "teacher"
	CleverUserTypeDistrictAdmin CleverUserType = "district_admin"
	CleverUserTypeSchoolAdmin   CleverUserType = "school_admin"
	CleverUserTypeContact       CleverUserType = "contact"
)

// CleverConfig holds configuration for Clever OAuth2 authentication.
type CleverConfig struct {
	// ClientID is the Clever OAuth2 client ID.
	ClientID string

	// ClientSecret is the Clever OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Clever.
	// Example: "https://school.app/auth/clever/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: read:user_id read:sis
	// Other options: read:students, read:teachers, read:sections, etc.
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

// Clever creates a new OAuth2 provider configured for Clever authentication.
//
// Clever is a popular SSO provider for K-12 schools. It provides user information
// including role (student, teacher, admin), SIS IDs, and district information.
//
// Usage in BuildHandler:
//
//	cleverAuth, err := oauth2.Clever(oauth2.CleverConfig{
//	    ClientID:     appCfg.CleverClientID,
//	    ClientSecret: appCfg.CleverClientSecret,
//	    RedirectURL:  "https://school.app/auth/clever/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/clever/login", cleverAuth.LoginHandler())
//	r.Get("/auth/clever/callback", cleverAuth.CallbackHandler())
//	r.Get("/auth/clever/logout", cleverAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(cleverAuth.RequireAuth("/auth/clever/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains Clever-specific fields:
//   - "user_type": student, teacher, district_admin, school_admin, or contact
//   - "district_id": Clever district ID
//   - "sis_id": Student Information System ID (if available)
//   - "school_ids": Comma-separated list of school IDs (if available)
func Clever(cfg CleverConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/clever: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/clever: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/clever: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/clever: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/clever: StateStore is required")
	}

	// Default scopes for Clever
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"read:user_id",
			"read:sis",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     cleverEndpoint,
	}

	providerCfg := &Config{
		ProviderName:  "clever",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchCleverUserInfo,
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

// cleverMeResponse represents the response from Clever's /me endpoint.
type cleverMeResponse struct {
	Type string          `json:"type"` // "student", "teacher", "district_admin", etc.
	Data json.RawMessage `json:"data"` // User data varies by type
}

// cleverUserData represents common user data from Clever.
type cleverUserData struct {
	ID       string `json:"id"`
	District string `json:"district"`
	Email    string `json:"email"`
	Name     struct {
		First  string `json:"first"`
		Middle string `json:"middle"`
		Last   string `json:"last"`
	} `json:"name"`
	// Student-specific
	SISId   string   `json:"sis_id"`
	Schools []string `json:"schools"`
	// Teacher-specific
	SchoolID string `json:"school"`
}

// fetchCleverUserInfo retrieves user information from Clever's API.
func fetchCleverUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Fetch user info from /me endpoint
	resp, err := client.Get("https://api.clever.com/v3.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var meResp cleverMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&meResp); err != nil {
		return nil, fmt.Errorf("failed to decode /me response: %w", err)
	}

	var userData cleverUserData
	if err := json.Unmarshal(meResp.Data, &userData); err != nil {
		return nil, fmt.Errorf("failed to decode user data: %w", err)
	}

	// Build display name
	name := userData.Name.First
	if userData.Name.Last != "" {
		name += " " + userData.Name.Last
	}

	// Build school IDs string
	schoolIDs := ""
	if len(userData.Schools) > 0 {
		for i, s := range userData.Schools {
			if i > 0 {
				schoolIDs += ","
			}
			schoolIDs += s
		}
	} else if userData.SchoolID != "" {
		schoolIDs = userData.SchoolID
	}

	// Parse raw data for the Raw field
	var rawData map[string]any
	json.Unmarshal(meResp.Data, &rawData)
	rawData["type"] = meResp.Type

	return &User{
		ID:            userData.ID,
		Email:         userData.Email,
		EmailVerified: userData.Email != "", // Clever emails are verified
		Name:          name,
		Picture:       "", // Clever doesn't provide profile pictures
		Raw:           rawData,
		Extra: map[string]string{
			"user_type":   meResp.Type,
			"district_id": userData.District,
			"sis_id":      userData.SISId,
			"school_ids":  schoolIDs,
			"first_name":  userData.Name.First,
			"last_name":   userData.Name.Last,
		},
	}, nil
}

// IsStudent returns true if the user is a student.
func IsCleverStudent(user *User) bool {
	return user.Extra["user_type"] == string(CleverUserTypeStudent)
}

// IsTeacher returns true if the user is a teacher.
func IsCleverTeacher(user *User) bool {
	return user.Extra["user_type"] == string(CleverUserTypeTeacher)
}

// IsCleverAdmin returns true if the user is a district or school admin.
func IsCleverAdmin(user *User) bool {
	userType := user.Extra["user_type"]
	return userType == string(CleverUserTypeDistrictAdmin) || userType == string(CleverUserTypeSchoolAdmin)
}

// CleverUserType returns the Clever user type from the User.
func GetCleverUserType(user *User) CleverUserType {
	return CleverUserType(user.Extra["user_type"])
}
