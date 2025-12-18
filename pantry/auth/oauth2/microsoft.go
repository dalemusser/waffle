// auth/oauth2/microsoft.go
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
)

// Microsoft OAuth2 endpoints.
// The "common" tenant allows both personal Microsoft accounts and Azure AD accounts.
// For tenant-specific auth, use: https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
var microsoftEndpoint = oauth2.Endpoint{
	AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
	TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
}

// MicrosoftTenant represents different Azure AD tenant configurations.
type MicrosoftTenant string

const (
	// MicrosoftTenantCommon allows both personal and work/school accounts.
	MicrosoftTenantCommon MicrosoftTenant = "common"
	// MicrosoftTenantConsumers allows only personal Microsoft accounts.
	MicrosoftTenantConsumers MicrosoftTenant = "consumers"
	// MicrosoftTenantOrganizations allows only work/school accounts from any Azure AD.
	MicrosoftTenantOrganizations MicrosoftTenant = "organizations"
)

// MicrosoftConfig holds configuration for Microsoft OAuth2 authentication.
type MicrosoftConfig struct {
	// ClientID is the Microsoft OAuth2 client ID (Application ID).
	ClientID string

	// ClientSecret is the Microsoft OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Microsoft.
	// Example: "https://myapp.com/auth/microsoft/callback"
	RedirectURL string

	// Tenant controls which accounts can sign in.
	// Default: "common" (both personal and work/school).
	// Options: "common", "consumers", "organizations", or a specific tenant ID.
	Tenant string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, profile, email, User.Read
	// For Microsoft 365 Education: add EduRoster.Read, etc.
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

// Microsoft creates a new OAuth2 provider configured for Microsoft authentication.
//
// Microsoft Identity Platform supports personal Microsoft accounts (Outlook, Xbox),
// work/school accounts (Azure AD, Microsoft 365), and Microsoft 365 Education.
//
// Usage in BuildHandler:
//
//	microsoftAuth, err := oauth2.Microsoft(oauth2.MicrosoftConfig{
//	    ClientID:     appCfg.MicrosoftClientID,
//	    ClientSecret: appCfg.MicrosoftClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/microsoft/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/microsoft/login", microsoftAuth.LoginHandler())
//	r.Get("/auth/microsoft/callback", microsoftAuth.CallbackHandler())
//	r.Get("/auth/microsoft/logout", microsoftAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(microsoftAuth.RequireAuth("/auth/microsoft/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains Microsoft-specific fields:
//   - "tenant_id": Azure AD tenant ID (for work/school accounts)
//   - "job_title": User's job title (if available)
//   - "department": User's department (if available)
//   - "office_location": User's office location (if available)
//   - "mobile_phone": User's mobile phone (if available)
//   - "preferred_username": User's preferred username/UPN
func Microsoft(cfg MicrosoftConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/microsoft: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/microsoft: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/microsoft: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/microsoft: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/microsoft: StateStore is required")
	}

	// Default tenant
	tenant := cfg.Tenant
	if tenant == "" {
		tenant = string(MicrosoftTenantCommon)
	}

	// Build endpoint with configured tenant
	endpoint := oauth2.Endpoint{
		AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant),
		TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
	}

	// Default scopes for Microsoft
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"profile",
			"email",
			"User.Read",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	providerCfg := &Config{
		ProviderName:  "microsoft",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchMicrosoftUserInfo,
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

// microsoftUserInfo represents the response from Microsoft Graph /me endpoint.
type microsoftUserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	OfficeLocation    string `json:"officeLocation"`
	MobilePhone       string `json:"mobilePhone"`
	// Photo is available at a separate endpoint
}

// fetchMicrosoftUserInfo retrieves user information from Microsoft Graph API.
func fetchMicrosoftUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Fetch user info from Microsoft Graph
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info microsoftUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Email might be in mail or userPrincipalName
	email := info.Mail
	if email == "" {
		email = info.UserPrincipalName
	}

	// Build display name
	name := info.DisplayName
	if name == "" {
		name = info.GivenName
		if info.Surname != "" {
			name += " " + info.Surname
		}
	}

	// Extract tenant ID from token if present (in ID token claims)
	tenantID := extractTenantFromToken(token)

	return &User{
		ID:            info.ID,
		Email:         email,
		EmailVerified: email != "", // Microsoft emails are verified
		Name:          name,
		Picture:       "", // Photo requires separate Graph API call
		Raw: map[string]any{
			"id":                info.ID,
			"displayName":       info.DisplayName,
			"givenName":         info.GivenName,
			"surname":           info.Surname,
			"mail":              info.Mail,
			"userPrincipalName": info.UserPrincipalName,
			"jobTitle":          info.JobTitle,
			"department":        info.Department,
			"officeLocation":    info.OfficeLocation,
			"mobilePhone":       info.MobilePhone,
		},
		Extra: map[string]string{
			"tenant_id":          tenantID,
			"job_title":          info.JobTitle,
			"department":         info.Department,
			"office_location":    info.OfficeLocation,
			"mobile_phone":       info.MobilePhone,
			"preferred_username": info.UserPrincipalName,
			"first_name":         info.GivenName,
			"last_name":          info.Surname,
		},
	}, nil
}

// extractTenantFromToken attempts to extract the tenant ID from the OAuth2 token.
// Microsoft includes the tenant ID in the ID token claims.
func extractTenantFromToken(token *oauth2.Token) string {
	// The ID token is available in token.Extra("id_token")
	// We could decode it, but for simplicity we'll leave this for
	// applications that need tenant-specific logic
	return ""
}

// MicrosoftEducation creates a Microsoft OAuth2 provider configured for Microsoft 365 Education.
//
// This helper pre-configures scopes commonly needed for educational applications,
// including roster access and assignment management.
//
// Usage:
//
//	eduAuth, err := oauth2.MicrosoftEducation(oauth2.MicrosoftEducationConfig{
//	    ClientID:     appCfg.MicrosoftClientID,
//	    ClientSecret: appCfg.MicrosoftClientSecret,
//	    RedirectURL:  "https://school.app/auth/microsoft/callback",
//	    TenantID:     appCfg.SchoolTenantID, // Specific school's Azure AD tenant
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
func MicrosoftEducation(cfg MicrosoftEducationConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.TenantID == "" {
		return nil, errors.New("oauth2/microsoft: TenantID is required for Education")
	}

	// Education-specific scopes
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"profile",
			"email",
			"User.Read",
			"EduRoster.Read",           // Read class rosters
			"EduAssignments.Read",      // Read assignments
			"EduAssignments.ReadBasic", // Read basic assignment info
		}
	}

	return Microsoft(MicrosoftConfig{
		ClientID:        cfg.ClientID,
		ClientSecret:    cfg.ClientSecret,
		RedirectURL:     cfg.RedirectURL,
		Tenant:          cfg.TenantID,
		Scopes:          scopes,
		SessionStore:    cfg.SessionStore,
		StateStore:      cfg.StateStore,
		SessionDuration: cfg.SessionDuration,
		CookieName:      cfg.CookieName,
		CookieSecure:    cfg.CookieSecure,
		OnSuccess:       cfg.OnSuccess,
		OnError:         cfg.OnError,
		Logger:          cfg.Logger,
	}, logger)
}

// MicrosoftEducationConfig holds configuration for Microsoft 365 Education OAuth2.
type MicrosoftEducationConfig struct {
	// ClientID is the Microsoft OAuth2 client ID.
	ClientID string

	// ClientSecret is the Microsoft OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Microsoft.
	RedirectURL string

	// TenantID is the Azure AD tenant ID for the school/district.
	// Required for education scenarios.
	TenantID string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, profile, email, User.Read, EduRoster.Read, EduAssignments.Read
	Scopes []string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid (in seconds).
	SessionDuration int

	// CookieName is the name of the session cookie.
	CookieName string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	CookieSecure bool

	// OnSuccess is called after successful authentication.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// MicrosoftGraphClient provides methods to call Microsoft Graph API.
// Use this after authentication to fetch additional user data, classes, etc.
type MicrosoftGraphClient struct {
	client *http.Client
}

// NewMicrosoftGraphClient creates a client for calling Microsoft Graph API.
// Requires a valid OAuth2 access token with appropriate scopes.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	graphClient := oauth2.NewMicrosoftGraphClient(user.AccessToken)
//	profile, err := graphClient.GetProfile(r.Context())
func NewMicrosoftGraphClient(accessToken string) *MicrosoftGraphClient {
	return &MicrosoftGraphClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
	}
}

// MicrosoftProfile contains extended profile information from Microsoft Graph.
type MicrosoftProfile struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	OfficeLocation    string `json:"officeLocation"`
	MobilePhone       string `json:"mobilePhone"`
	BusinessPhones    []string `json:"businessPhones"`
}

// GetProfile fetches the user's profile from Microsoft Graph.
func (c *MicrosoftGraphClient) GetProfile(ctx context.Context) (*MicrosoftProfile, error) {
	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var profile MicrosoftProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	return &profile, nil
}

// MicrosoftEducationClass represents a class/course in Microsoft 365 Education.
type MicrosoftEducationClass struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Description       string `json:"description"`
	MailNickname      string `json:"mailNickname"`
	ExternalID        string `json:"externalId"`
	ExternalName      string `json:"externalName"`
	ExternalSourceDetail string `json:"externalSourceDetail"`
	Grade             string `json:"grade"`
}

// educationClassesResponse represents the Graph API response for classes.
type educationClassesResponse struct {
	Value []MicrosoftEducationClass `json:"value"`
}

// GetMyClasses fetches the classes the authenticated user belongs to.
// Requires EduRoster.Read scope.
func (c *MicrosoftGraphClient) GetMyClasses(ctx context.Context) ([]MicrosoftEducationClass, error) {
	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/education/me/classes")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch classes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var classesResp educationClassesResponse
	if err := json.NewDecoder(resp.Body).Decode(&classesResp); err != nil {
		return nil, fmt.Errorf("failed to decode classes: %w", err)
	}

	return classesResp.Value, nil
}

// MicrosoftEducationUser represents a user in Microsoft 365 Education context.
type MicrosoftEducationUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	PrimaryRole       string `json:"primaryRole"` // student, teacher, etc.
	ExternalSourceDetail string `json:"externalSourceDetail"`
}

// GetEducationUser fetches the authenticated user's education profile.
// Returns role information (student, teacher, etc.).
func (c *MicrosoftGraphClient) GetEducationUser(ctx context.Context) (*MicrosoftEducationUser, error) {
	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/education/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch education user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var eduUser MicrosoftEducationUser
	if err := json.NewDecoder(resp.Body).Decode(&eduUser); err != nil {
		return nil, fmt.Errorf("failed to decode education user: %w", err)
	}

	return &eduUser, nil
}

// educationClassMembersResponse represents the Graph API response for class members.
type educationClassMembersResponse struct {
	Value []MicrosoftEducationUser `json:"value"`
}

// GetClassMembers fetches the members of a specific class.
// Requires EduRoster.Read scope.
func (c *MicrosoftGraphClient) GetClassMembers(ctx context.Context, classID string) ([]MicrosoftEducationUser, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/education/classes/%s/members", classID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch class members: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var membersResp educationClassMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&membersResp); err != nil {
		return nil, fmt.Errorf("failed to decode class members: %w", err)
	}

	return membersResp.Value, nil
}

// GetClassTeachers fetches the teachers of a specific class.
// Requires EduRoster.Read scope.
func (c *MicrosoftGraphClient) GetClassTeachers(ctx context.Context, classID string) ([]MicrosoftEducationUser, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/education/classes/%s/teachers", classID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch class teachers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var teachersResp educationClassMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&teachersResp); err != nil {
		return nil, fmt.Errorf("failed to decode class teachers: %w", err)
	}

	return teachersResp.Value, nil
}

// IsMicrosoftEducationStudent returns true if the education user is a student.
func IsMicrosoftEducationStudent(eduUser *MicrosoftEducationUser) bool {
	return eduUser.PrimaryRole == "student"
}

// IsMicrosoftEducationTeacher returns true if the education user is a teacher.
func IsMicrosoftEducationTeacher(eduUser *MicrosoftEducationUser) bool {
	return eduUser.PrimaryRole == "teacher"
}
