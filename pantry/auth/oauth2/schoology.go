// auth/oauth2/schoology.go
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
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// SchoologyUserRole represents the type of user in Schoology.
type SchoologyUserRole string

const (
	SchoologyUserRoleStudent SchoologyUserRole = "student"
	SchoologyUserRoleTeacher SchoologyUserRole = "teacher"
	SchoologyUserRoleAdmin   SchoologyUserRole = "admin"
	SchoologyUserRoleParent  SchoologyUserRole = "parent"
)

// SchoologyConfig holds configuration for Schoology OAuth2 authentication.
type SchoologyConfig struct {
	// ClientID is the Schoology OAuth2 consumer key.
	ClientID string

	// ClientSecret is the Schoology OAuth2 consumer secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Schoology.
	// Example: "https://school.app/auth/schoology/callback"
	RedirectURL string

	// Domain is the Schoology domain for your district/school.
	// Example: "myschool.schoology.com" or "app.schoology.com"
	// If empty, defaults to "app.schoology.com"
	Domain string

	// Scopes are the OAuth2 scopes to request.
	// Schoology uses a simple scope model, typically empty or custom.
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

// Schoology creates a new OAuth2 provider configured for Schoology authentication.
//
// Schoology (owned by PowerSchool) is a popular Learning Management System (LMS)
// used in K-12 and higher education. It provides OAuth2 authentication and access
// to courses, assignments, grades, and user data.
//
// Setup in Schoology:
//  1. Log in to Schoology as a System Administrator
//  2. Go to Tools → API → API Credentials (or System Settings → Integration → API)
//  3. Click "Add Credentials" or "Register an App"
//  4. Set your Redirect URI
//  5. Copy your Consumer Key (Client ID) and Consumer Secret
//
// Usage in BuildHandler:
//
//	schoologyAuth, err := oauth2.Schoology(oauth2.SchoologyConfig{
//	    ClientID:     appCfg.SchoologyClientID,
//	    ClientSecret: appCfg.SchoologyClientSecret,
//	    RedirectURL:  "https://school.app/auth/schoology/callback",
//	    Domain:       "myschool.schoology.com", // Optional, defaults to app.schoology.com
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/schoology/login", schoologyAuth.LoginHandler())
//	r.Get("/auth/schoology/callback", schoologyAuth.CallbackHandler())
//	r.Get("/auth/schoology/logout", schoologyAuth.LogoutHandler())
//
// The User.Extra map contains Schoology-specific fields:
//   - "schoology_uid": Schoology user ID
//   - "role": student, teacher, admin, or parent
//   - "school_id": Primary school ID
//   - "school_nid": School node ID
//   - "building_id": Building ID
//   - "name_title": User's title (Mr., Mrs., etc.)
//   - "name_first_preferred": Preferred first name
//   - "username": Schoology username
//   - "primary_email": User's primary email
//   - "grad_year": Graduation year (students)
func Schoology(cfg SchoologyConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/schoology: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/schoology: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/schoology: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/schoology: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/schoology: StateStore is required")
	}

	// Default domain
	domain := cfg.Domain
	if domain == "" {
		domain = "app.schoology.com"
	}
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	// Schoology OAuth2 endpoint
	baseURL := fmt.Sprintf("https://%s", domain)
	endpoint := oauth2.Endpoint{
		AuthURL:  baseURL + "/oauth/authorize",
		TokenURL: baseURL + "/oauth/access_token",
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint:     endpoint,
	}

	// Create a fetcher that captures the domain
	fetchUserInfo := createSchoologyUserInfoFetcher(baseURL)

	providerCfg := &Config{
		ProviderName:  "schoology",
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

// schoologyUserInfo represents the response from Schoology's users/me endpoint.
type schoologyUserInfo struct {
	UID                string `json:"uid"`
	ID                 int64  `json:"id"`
	SchoolID           int64  `json:"school_id"`
	SchoolNID          string `json:"school_nid"`
	BuildingID         int64  `json:"building_id"`
	NameTitle          string `json:"name_title"`
	NameTitleShow      int    `json:"name_title_show"`
	NameFirst          string `json:"name_first"`
	NameFirstPreferred string `json:"name_first_preferred"`
	NameMiddle         string `json:"name_middle"`
	NameMiddleShow     int    `json:"name_middle_show"`
	NameLast           string `json:"name_last"`
	NameDisplay        string `json:"name_display"`
	Username           string `json:"username"`
	PrimaryEmail       string `json:"primary_email"`
	Picture            string `json:"picture_url"`
	Gender             string `json:"gender"`
	GradYear           string `json:"grad_year"`
	Birthday           string `json:"birthday"`
	Role               string `json:"role_id"`
	RoleTitle          string `json:"role_title"`
	Email              string `json:"email"`
	Position           string `json:"position"`
	Timezone           string `json:"tz_name"`
	Locale             string `json:"language"`
}

// createSchoologyUserInfoFetcher creates a UserInfoFetcher for Schoology.
func createSchoologyUserInfoFetcher(baseURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from Schoology's API
		resp, err := client.Get(baseURL + "/v1/users/me")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info schoologyUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		// Build display name
		name := info.NameDisplay
		if name == "" {
			firstName := info.NameFirstPreferred
			if firstName == "" {
				firstName = info.NameFirst
			}
			name = strings.TrimSpace(firstName + " " + info.NameLast)
		}

		// Determine email
		email := info.PrimaryEmail
		if email == "" {
			email = info.Email
		}

		// Normalize role
		role := normalizeSchoologyRole(info.Role, info.RoleTitle)

		// Use UID as ID, or string ID if UID is empty
		userID := info.UID
		if userID == "" && info.ID > 0 {
			userID = strconv.FormatInt(info.ID, 10)
		}

		return &User{
			ID:            userID,
			Email:         email,
			EmailVerified: email != "", // Schoology doesn't provide verification status
			Name:          name,
			Picture:       info.Picture,
			Raw: map[string]any{
				"uid":                  info.UID,
				"id":                   info.ID,
				"school_id":            info.SchoolID,
				"school_nid":           info.SchoolNID,
				"building_id":          info.BuildingID,
				"name_title":           info.NameTitle,
				"name_first":           info.NameFirst,
				"name_first_preferred": info.NameFirstPreferred,
				"name_middle":          info.NameMiddle,
				"name_last":            info.NameLast,
				"name_display":         info.NameDisplay,
				"username":             info.Username,
				"primary_email":        info.PrimaryEmail,
				"picture_url":          info.Picture,
				"gender":               info.Gender,
				"grad_year":            info.GradYear,
				"birthday":             info.Birthday,
				"role_id":              info.Role,
				"role_title":           info.RoleTitle,
				"position":             info.Position,
				"tz_name":              info.Timezone,
				"language":             info.Locale,
			},
			Extra: map[string]string{
				"schoology_uid":        userID,
				"role":                 role,
				"school_id":            strconv.FormatInt(info.SchoolID, 10),
				"school_nid":           info.SchoolNID,
				"building_id":          strconv.FormatInt(info.BuildingID, 10),
				"name_title":           info.NameTitle,
				"name_first_preferred": info.NameFirstPreferred,
				"username":             info.Username,
				"primary_email":        email,
				"grad_year":            info.GradYear,
				"first_name":           info.NameFirst,
				"last_name":            info.NameLast,
				"timezone":             info.Timezone,
				"locale":               info.Locale,
			},
		}, nil
	}
}

// normalizeSchoologyRole converts Schoology role IDs/titles to standard form.
func normalizeSchoologyRole(roleID, roleTitle string) string {
	// Schoology uses numeric role IDs:
	// 1 = Student, 2 = Parent, 3 = Teacher, 4 = Member, 5 = Admin
	switch roleID {
	case "1":
		return string(SchoologyUserRoleStudent)
	case "2":
		return string(SchoologyUserRoleParent)
	case "3":
		return string(SchoologyUserRoleTeacher)
	case "5":
		return string(SchoologyUserRoleAdmin)
	}

	// Fall back to role title
	titleLower := strings.ToLower(roleTitle)
	switch {
	case strings.Contains(titleLower, "student"):
		return string(SchoologyUserRoleStudent)
	case strings.Contains(titleLower, "teacher"), strings.Contains(titleLower, "instructor"):
		return string(SchoologyUserRoleTeacher)
	case strings.Contains(titleLower, "admin"):
		return string(SchoologyUserRoleAdmin)
	case strings.Contains(titleLower, "parent"), strings.Contains(titleLower, "guardian"):
		return string(SchoologyUserRoleParent)
	default:
		return roleTitle
	}
}

// IsSchoologyStudent returns true if the user is a student.
func IsSchoologyStudent(user *User) bool {
	return user.Extra["role"] == string(SchoologyUserRoleStudent)
}

// IsSchoologyTeacher returns true if the user is a teacher.
func IsSchoologyTeacher(user *User) bool {
	return user.Extra["role"] == string(SchoologyUserRoleTeacher)
}

// IsSchoologyAdmin returns true if the user is an administrator.
func IsSchoologyAdmin(user *User) bool {
	return user.Extra["role"] == string(SchoologyUserRoleAdmin)
}

// IsSchoologyParent returns true if the user is a parent/guardian.
func IsSchoologyParent(user *User) bool {
	return user.Extra["role"] == string(SchoologyUserRoleParent)
}

// GetSchoologyUserRole returns the Schoology user role.
func GetSchoologyUserRole(user *User) SchoologyUserRole {
	return SchoologyUserRole(user.Extra["role"])
}

// SchoologyAPIClient provides methods to call Schoology API endpoints.
// Use this after authentication to fetch courses, assignments, grades, etc.
type SchoologyAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewSchoologyAPIClient creates a client for calling Schoology API.
// Requires a valid OAuth2 access token and optionally a custom domain.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	schoologyClient := oauth2.NewSchoologyAPIClient(user.AccessToken, "myschool.schoology.com")
//	courses, err := schoologyClient.GetUserCourses(r.Context())
func NewSchoologyAPIClient(accessToken, domain string) *SchoologyAPIClient {
	if domain == "" {
		domain = "app.schoology.com"
	}
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	return &SchoologyAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: fmt.Sprintf("https://%s/v1", domain),
	}
}

// SchoolorgyCourse represents a course in Schoology.
type SchoologyCourse struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	CourseCode      string `json:"course_code"`
	SectionID       int64  `json:"section_id"`
	SectionTitle    string `json:"section_title"`
	SectionCode     string `json:"section_code"`
	SectionSchoolID int64  `json:"section_school_id"`
	GradingPeriods  []int64 `json:"grading_periods"`
	Description     string `json:"description"`
	Subject         string `json:"subject_area"`
	GradeLevel      string `json:"grade_level_range"`
}

// schoologyCoursesResponse represents the API response for courses.
type schoologyCoursesResponse struct {
	Course []SchoologyCourse `json:"course"`
}

// sectionResponse wraps the section array from Schoology API.
type schoologySectionsResponse struct {
	Section []SchoologyCourse `json:"section"`
}

// GetUserCourses fetches the courses for the current user.
func (c *SchoologyAPIClient) GetUserCourses(ctx context.Context) ([]SchoologyCourse, error) {
	resp, err := c.client.Get(c.baseURL + "/users/me/sections")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result schoologySectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return result.Section, nil
}

// SchoologyAssignment represents an assignment in Schoology.
type SchoologyAssignment struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	DueDate        string  `json:"due"`
	GradingScale   int64   `json:"grading_scale"`
	MaxPoints      float64 `json:"max_points"`
	Factor         float64 `json:"factor"`
	Type           string  `json:"type"`
	GradingPeriod  int64   `json:"grading_period"`
	GradingGroup   int64   `json:"grading_group"`
	Published      int     `json:"published"`
	AllowDropbox   int     `json:"allow_dropbox"`
	ShowComments   int     `json:"show_comments"`
}

// schoologyAssignmentsResponse represents the API response for assignments.
type schoologyAssignmentsResponse struct {
	Assignment []SchoologyAssignment `json:"assignment"`
}

// GetCourseAssignments fetches assignments for a course/section.
func (c *SchoologyAPIClient) GetCourseAssignments(ctx context.Context, sectionID int64) ([]SchoologyAssignment, error) {
	url := fmt.Sprintf("%s/sections/%d/assignments", c.baseURL, sectionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result schoologyAssignmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return result.Assignment, nil
}

// SchoologyGrade represents a grade in Schoology.
type SchoologyGrade struct {
	AssignmentID int64   `json:"assignment_id"`
	EnrollmentID int64   `json:"enrollment_id"`
	Grade        float64 `json:"grade"`
	Exception    int     `json:"exception"` // 0=none, 1=excused, 2=incomplete
	MaxPoints    float64 `json:"max_points"`
	Comment      string  `json:"comment"`
	Timestamp    int64   `json:"timestamp"`
}

// schoologyGradesResponse represents the API response for grades.
type schoologyGradesResponse struct {
	Grades struct {
		Grade []SchoologyGrade `json:"grade"`
	} `json:"grades"`
}

// GetCourseGrades fetches grades for a course/section.
func (c *SchoologyAPIClient) GetCourseGrades(ctx context.Context, sectionID int64) ([]SchoologyGrade, error) {
	url := fmt.Sprintf("%s/sections/%d/grades", c.baseURL, sectionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result schoologyGradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return result.Grades.Grade, nil
}

// SchoologyEnrollment represents a user's enrollment in a course.
type SchoologyEnrollment struct {
	ID          int64  `json:"id"`
	UID         string `json:"uid"`
	SectionID   int64  `json:"section_id"`
	Status      int    `json:"status"` // 1=active, 2=expired, 3=invited, etc.
	Admin       int    `json:"admin"`  // 1 if course admin
	Name        string `json:"name_display"`
	Picture     string `json:"picture_url"`
}

// schoologyEnrollmentsResponse represents the API response for enrollments.
type schoologyEnrollmentsResponse struct {
	Enrollment []SchoologyEnrollment `json:"enrollment"`
}

// GetCourseEnrollments fetches enrollments (roster) for a course/section.
func (c *SchoologyAPIClient) GetCourseEnrollments(ctx context.Context, sectionID int64) ([]SchoologyEnrollment, error) {
	url := fmt.Sprintf("%s/sections/%d/enrollments", c.baseURL, sectionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrollments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result schoologyEnrollmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode enrollments: %w", err)
	}

	return result.Enrollment, nil
}

// SchoologySchool represents a school in Schoology.
type SchoologySchool struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	Website     string `json:"website"`
	Phone       string `json:"phone"`
	Fax         string `json:"fax"`
	Picture     string `json:"picture_url"`
}

// GetSchool fetches a school by ID.
func (c *SchoologyAPIClient) GetSchool(ctx context.Context, schoolID int64) (*SchoologySchool, error) {
	url := fmt.Sprintf("%s/schools/%d", c.baseURL, schoolID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch school: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var school SchoologySchool
	if err := json.NewDecoder(resp.Body).Decode(&school); err != nil {
		return nil, fmt.Errorf("failed to decode school: %w", err)
	}

	return &school, nil
}

// SchoologyGradingPeriod represents a grading period.
type SchoologyGradingPeriod struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"start"` // YYYY-MM-DD
	EndDate   string `json:"end"`   // YYYY-MM-DD
	Active    int    `json:"active"`
}

// schoologyGradingPeriodsResponse represents the API response for grading periods.
type schoologyGradingPeriodsResponse struct {
	GradingPeriod []SchoologyGradingPeriod `json:"grading_period"`
}

// GetSchoolGradingPeriods fetches grading periods for a school.
func (c *SchoologyAPIClient) GetSchoolGradingPeriods(ctx context.Context, schoolID int64) ([]SchoologyGradingPeriod, error) {
	url := fmt.Sprintf("%s/schools/%d/grading_periods", c.baseURL, schoolID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grading periods: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result schoologyGradingPeriodsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode grading periods: %w", err)
	}

	return result.GradingPeriod, nil
}

// GetSchoologySchoolID returns the user's school ID as int64.
func GetSchoologySchoolID(user *User) int64 {
	id, _ := strconv.ParseInt(user.Extra["school_id"], 10, 64)
	return id
}
