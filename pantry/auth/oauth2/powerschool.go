// auth/oauth2/powerschool.go
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
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// PowerSchoolUserType represents the type of user in PowerSchool.
type PowerSchoolUserType string

const (
	PowerSchoolUserTypeStudent PowerSchoolUserType = "student"
	PowerSchoolUserTypeTeacher PowerSchoolUserType = "teacher"
	PowerSchoolUserTypeAdmin   PowerSchoolUserType = "admin"
	PowerSchoolUserTypeParent  PowerSchoolUserType = "parent"
)

// PowerSchoolConfig holds configuration for PowerSchool OAuth2 authentication.
type PowerSchoolConfig struct {
	// ClientID is the PowerSchool OAuth2 client ID (Plugin Client ID).
	ClientID string

	// ClientSecret is the PowerSchool OAuth2 client secret (Plugin Client Secret).
	ClientSecret string

	// RedirectURL is the callback URL registered with PowerSchool.
	// Example: "https://school.app/auth/powerschool/callback"
	RedirectURL string

	// ServerURL is the PowerSchool server URL for your district.
	// Example: "https://district.powerschool.com"
	// This is required as each district has their own PowerSchool instance.
	ServerURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid profile email
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

// PowerSchool creates a new OAuth2 provider configured for PowerSchool authentication.
//
// PowerSchool is one of the most widely used Student Information Systems (SIS) in K-12
// education. It provides OAuth2/OIDC authentication and access to student, teacher,
// and parent data.
//
// Important: Each school district has their own PowerSchool server URL, so you must
// configure the ServerURL for your specific district.
//
// Usage in BuildHandler:
//
//	powerSchoolAuth, err := oauth2.PowerSchool(oauth2.PowerSchoolConfig{
//	    ClientID:     appCfg.PowerSchoolClientID,
//	    ClientSecret: appCfg.PowerSchoolClientSecret,
//	    RedirectURL:  "https://school.app/auth/powerschool/callback",
//	    ServerURL:    appCfg.PowerSchoolServerURL, // e.g., "https://district.powerschool.com"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/powerschool/login", powerSchoolAuth.LoginHandler())
//	r.Get("/auth/powerschool/callback", powerSchoolAuth.CallbackHandler())
//	r.Get("/auth/powerschool/logout", powerSchoolAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(powerSchoolAuth.RequireAuth("/auth/powerschool/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains PowerSchool-specific fields:
//   - "user_type": student, teacher, admin, or parent
//   - "dcid": PowerSchool internal database ID
//   - "student_dcid": Student DCID (for parents, links to their student)
//   - "school_id": Current school ID
//   - "district_id": District ID
//   - "grade_level": Student's grade level (students only)
//   - "student_number": Student number/ID (students only)
func PowerSchool(cfg PowerSchoolConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/powerschool: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/powerschool: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/powerschool: RedirectURL is required")
	}
	if cfg.ServerURL == "" {
		return nil, errors.New("oauth2/powerschool: ServerURL is required (e.g., https://district.powerschool.com)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/powerschool: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/powerschool: StateStore is required")
	}

	// Build PowerSchool OAuth2 endpoint for this district
	endpoint := oauth2.Endpoint{
		AuthURL:  cfg.ServerURL + "/oauth/access_token",
		TokenURL: cfg.ServerURL + "/oauth/access_token",
	}

	// Default scopes for PowerSchool
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

	// Create a fetcher that captures the server URL
	fetchUserInfo := createPowerSchoolUserInfoFetcher(cfg.ServerURL)

	providerCfg := &Config{
		ProviderName:  "powerschool",
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

// powerSchoolUserInfo represents user information from PowerSchool's OpenID Connect.
type powerSchoolUserInfo struct {
	Sub           string `json:"sub"`            // Subject identifier
	Name          string `json:"name"`           // Full name
	GivenName     string `json:"given_name"`     // First name
	FamilyName    string `json:"family_name"`    // Last name
	Email         string `json:"email"`          // Email address
	EmailVerified bool   `json:"email_verified"` // Email verification status
	// PowerSchool-specific claims
	UserType      string `json:"usertype"`       // Type of user (student, teacher, admin, guardian)
	DCID          int64  `json:"dcid"`           // PowerSchool internal ID
	StudentDCID   int64  `json:"student_dcid"`   // Student DCID (for parents)
	SchoolID      int64  `json:"school_id"`      // School ID
	DistrictID    int64  `json:"district_id"`    // District ID
	GradeLevel    string `json:"grade_level"`    // Grade level (students)
	StudentNumber string `json:"student_number"` // Student number
}

// createPowerSchoolUserInfoFetcher creates a UserInfoFetcher bound to a specific server URL.
func createPowerSchoolUserInfoFetcher(serverURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from PowerSchool's OpenID Connect userinfo endpoint
		resp, err := client.Get(serverURL + "/oauth/userinfo")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info powerSchoolUserInfo
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

		// Normalize user type
		userType := normalizePowerSchoolUserType(info.UserType)

		// Use Sub as ID, or DCID if Sub is not available
		userID := info.Sub
		if userID == "" && info.DCID > 0 {
			userID = strconv.FormatInt(info.DCID, 10)
		}

		return &User{
			ID:            userID,
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
			Name:          name,
			Picture:       "", // PowerSchool doesn't typically provide profile pictures
			Raw: map[string]any{
				"sub":            info.Sub,
				"name":           info.Name,
				"given_name":     info.GivenName,
				"family_name":    info.FamilyName,
				"email":          info.Email,
				"email_verified": info.EmailVerified,
				"usertype":       info.UserType,
				"dcid":           info.DCID,
				"student_dcid":   info.StudentDCID,
				"school_id":      info.SchoolID,
				"district_id":    info.DistrictID,
				"grade_level":    info.GradeLevel,
				"student_number": info.StudentNumber,
			},
			Extra: map[string]string{
				"user_type":      userType,
				"dcid":           strconv.FormatInt(info.DCID, 10),
				"student_dcid":   strconv.FormatInt(info.StudentDCID, 10),
				"school_id":      strconv.FormatInt(info.SchoolID, 10),
				"district_id":    strconv.FormatInt(info.DistrictID, 10),
				"grade_level":    info.GradeLevel,
				"student_number": info.StudentNumber,
				"first_name":     info.GivenName,
				"last_name":      info.FamilyName,
			},
		}, nil
	}
}

// normalizePowerSchoolUserType converts PowerSchool user types to standard form.
func normalizePowerSchoolUserType(userType string) string {
	switch userType {
	case "Student", "student", "S":
		return string(PowerSchoolUserTypeStudent)
	case "Teacher", "teacher", "Staff", "staff", "T":
		return string(PowerSchoolUserTypeTeacher)
	case "Admin", "admin", "Administrator", "administrator", "A":
		return string(PowerSchoolUserTypeAdmin)
	case "Guardian", "guardian", "Parent", "parent", "G", "P":
		return string(PowerSchoolUserTypeParent)
	default:
		return userType
	}
}

// IsPowerSchoolStudent returns true if the user is a student.
func IsPowerSchoolStudent(user *User) bool {
	return user.Extra["user_type"] == string(PowerSchoolUserTypeStudent)
}

// IsPowerSchoolTeacher returns true if the user is a teacher/staff member.
func IsPowerSchoolTeacher(user *User) bool {
	return user.Extra["user_type"] == string(PowerSchoolUserTypeTeacher)
}

// IsPowerSchoolAdmin returns true if the user is an administrator.
func IsPowerSchoolAdmin(user *User) bool {
	return user.Extra["user_type"] == string(PowerSchoolUserTypeAdmin)
}

// IsPowerSchoolParent returns true if the user is a parent/guardian.
func IsPowerSchoolParent(user *User) bool {
	return user.Extra["user_type"] == string(PowerSchoolUserTypeParent)
}

// GetPowerSchoolUserType returns the PowerSchool user type from the User.
func GetPowerSchoolUserType(user *User) PowerSchoolUserType {
	return PowerSchoolUserType(user.Extra["user_type"])
}

// PowerSchoolAPIClient provides methods to call PowerSchool API endpoints.
// Use this after authentication to fetch additional data like enrollments, schedules, etc.
type PowerSchoolAPIClient struct {
	client    *http.Client
	serverURL string
}

// NewPowerSchoolAPIClient creates a client for calling PowerSchool API.
// Requires a valid OAuth2 access token and the PowerSchool server URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	psClient := oauth2.NewPowerSchoolAPIClient(user.AccessToken, "https://district.powerschool.com")
//	student, err := psClient.GetStudent(r.Context(), studentDCID)
func NewPowerSchoolAPIClient(accessToken, serverURL string) *PowerSchoolAPIClient {
	return &PowerSchoolAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		serverURL: serverURL,
	}
}

// PowerSchoolStudent represents a student record from PowerSchool.
type PowerSchoolStudent struct {
	DCID          int64  `json:"dcid"`
	StudentNumber string `json:"student_number"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	MiddleName    string `json:"middle_name"`
	GradeLevel    string `json:"grade_level"`
	SchoolID      int64  `json:"school_id"`
	EntryDate     string `json:"entry_date"`
	ExitDate      string `json:"exit_date"`
	Email         string `json:"email"`
}

// GetStudent fetches a student record by DCID.
// Requires appropriate API access permissions.
func (c *PowerSchoolAPIClient) GetStudent(ctx context.Context, dcid int64) (*PowerSchoolStudent, error) {
	url := fmt.Sprintf("%s/ws/v1/student/%d", c.serverURL, dcid)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Student PowerSchoolStudent `json:"student"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode student: %w", err)
	}

	return &result.Student, nil
}

// PowerSchoolSection represents a class section from PowerSchool.
type PowerSchoolSection struct {
	DCID         int64  `json:"dcid"`
	SectionNumber string `json:"section_number"`
	CourseNumber string `json:"course_number"`
	CourseName   string `json:"course_name"`
	TeacherDCID  int64  `json:"teacher_dcid"`
	TeacherName  string `json:"teacher_name"`
	RoomNumber   string `json:"room"`
	Period       string `json:"expression"` // Period/expression
	TermID       int64  `json:"term_id"`
}

// powerSchoolSectionsResponse represents the API response for sections.
type powerSchoolSectionsResponse struct {
	Sections struct {
		Section []PowerSchoolSection `json:"section"`
	} `json:"sections"`
}

// GetStudentSections fetches the class sections for a student.
// Requires appropriate API access permissions.
func (c *PowerSchoolAPIClient) GetStudentSections(ctx context.Context, studentDCID int64) ([]PowerSchoolSection, error) {
	url := fmt.Sprintf("%s/ws/v1/student/%d/sections", c.serverURL, studentDCID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result powerSchoolSectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return result.Sections.Section, nil
}

// PowerSchoolSchool represents a school from PowerSchool.
type PowerSchoolSchool struct {
	DCID         int64  `json:"dcid"`
	SchoolNumber int64  `json:"school_number"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Zip          string `json:"zip"`
	Phone        string `json:"phone"`
	LowGrade     string `json:"low_grade"`
	HighGrade    string `json:"high_grade"`
}

// GetSchool fetches a school record by ID.
func (c *PowerSchoolAPIClient) GetSchool(ctx context.Context, schoolID int64) (*PowerSchoolSchool, error) {
	url := fmt.Sprintf("%s/ws/v1/school/%d", c.serverURL, schoolID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch school: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		School PowerSchoolSchool `json:"school"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode school: %w", err)
	}

	return &result.School, nil
}
