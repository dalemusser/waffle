// auth/oauth2/canvas.go
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

// CanvasEnrollmentType represents the type of enrollment in Canvas.
type CanvasEnrollmentType string

const (
	CanvasEnrollmentStudent    CanvasEnrollmentType = "student"
	CanvasEnrollmentTeacher    CanvasEnrollmentType = "teacher"
	CanvasEnrollmentTA         CanvasEnrollmentType = "ta"
	CanvasEnrollmentObserver   CanvasEnrollmentType = "observer"
	CanvasEnrollmentDesigner   CanvasEnrollmentType = "designer"
)

// CanvasConfig holds configuration for Canvas LMS OAuth2 authentication.
type CanvasConfig struct {
	// ClientID is the Canvas OAuth2 client ID (Developer Key ID).
	ClientID string

	// ClientSecret is the Canvas OAuth2 client secret (Developer Key Secret).
	ClientSecret string

	// RedirectURL is the callback URL registered with Canvas.
	// Example: "https://school.app/auth/canvas/callback"
	RedirectURL string

	// CanvasURL is the Canvas instance URL for your institution.
	// Example: "https://myschool.instructure.com" or "https://canvas.myschool.edu"
	// This is required as each institution has their own Canvas instance.
	CanvasURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: url:GET|/api/v1/users/:user_id/profile
	// See: https://canvas.instructure.com/doc/api/file.oauth.html#oauth2-flow
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

// Canvas creates a new OAuth2 provider configured for Canvas LMS authentication.
//
// Canvas by Instructure is one of the most popular Learning Management Systems (LMS)
// used in K-12 and higher education. It provides OAuth2 authentication and access
// to courses, assignments, grades, and user data.
//
// Important: Each institution has their own Canvas instance, so you must configure
// the CanvasURL for your specific school/district.
//
// To create a Developer Key:
//  1. Log in to Canvas as an admin
//  2. Go to Admin → Developer Keys
//  3. Click "+ Developer Key" → "+ API Key"
//  4. Set the redirect URI and configure scopes
//  5. Save and note your Client ID and Secret
//
// Usage in BuildHandler:
//
//	canvasAuth, err := oauth2.Canvas(oauth2.CanvasConfig{
//	    ClientID:     appCfg.CanvasClientID,
//	    ClientSecret: appCfg.CanvasClientSecret,
//	    RedirectURL:  "https://school.app/auth/canvas/callback",
//	    CanvasURL:    appCfg.CanvasURL, // e.g., "https://myschool.instructure.com"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/canvas/login", canvasAuth.LoginHandler())
//	r.Get("/auth/canvas/callback", canvasAuth.CallbackHandler())
//	r.Get("/auth/canvas/logout", canvasAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(canvasAuth.RequireAuth("/auth/canvas/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains Canvas-specific fields:
//   - "canvas_user_id": Canvas user ID
//   - "login_id": User's login ID (username)
//   - "sis_user_id": SIS user ID (if available)
//   - "integration_id": Integration ID (if available)
//   - "primary_email": User's primary email
//   - "locale": User's preferred locale
//   - "time_zone": User's time zone
func Canvas(cfg CanvasConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/canvas: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/canvas: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/canvas: RedirectURL is required")
	}
	if cfg.CanvasURL == "" {
		return nil, errors.New("oauth2/canvas: CanvasURL is required (e.g., https://myschool.instructure.com)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/canvas: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/canvas: StateStore is required")
	}

	// Build Canvas OAuth2 endpoint for this institution
	endpoint := oauth2.Endpoint{
		AuthURL:  cfg.CanvasURL + "/login/oauth2/auth",
		TokenURL: cfg.CanvasURL + "/login/oauth2/token",
	}

	// Default scopes for Canvas
	// Canvas uses URL-based scopes
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"url:GET|/api/v1/users/:user_id/profile",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	// Create a fetcher that captures the Canvas URL
	fetchUserInfo := createCanvasUserInfoFetcher(cfg.CanvasURL)

	providerCfg := &Config{
		ProviderName:  "canvas",
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

// canvasUserProfile represents the response from Canvas's profile endpoint.
type canvasUserProfile struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	ShortName     string `json:"short_name"`
	SortableName  string `json:"sortable_name"`
	Title         string `json:"title"`
	Bio           string `json:"bio"`
	PrimaryEmail  string `json:"primary_email"`
	LoginID       string `json:"login_id"`
	SISUserID     string `json:"sis_user_id"`
	LTIUSERID     string `json:"lti_user_id"`
	IntegrationID string `json:"integration_id"`
	AvatarURL     string `json:"avatar_url"`
	Locale        string `json:"locale"`
	TimeZone      string `json:"time_zone"`
}

// createCanvasUserInfoFetcher creates a UserInfoFetcher bound to a specific Canvas URL.
func createCanvasUserInfoFetcher(canvasURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user profile from Canvas API
		resp, err := client.Get(canvasURL + "/api/v1/users/self/profile")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user profile: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var profile canvasUserProfile
		if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
			return nil, fmt.Errorf("failed to decode user profile: %w", err)
		}

		return &User{
			ID:            strconv.FormatInt(profile.ID, 10),
			Email:         profile.PrimaryEmail,
			EmailVerified: profile.PrimaryEmail != "",
			Name:          profile.Name,
			Picture:       profile.AvatarURL,
			Raw: map[string]any{
				"id":             profile.ID,
				"name":           profile.Name,
				"short_name":     profile.ShortName,
				"sortable_name":  profile.SortableName,
				"title":          profile.Title,
				"bio":            profile.Bio,
				"primary_email":  profile.PrimaryEmail,
				"login_id":       profile.LoginID,
				"sis_user_id":    profile.SISUserID,
				"lti_user_id":    profile.LTIUSERID,
				"integration_id": profile.IntegrationID,
				"avatar_url":     profile.AvatarURL,
				"locale":         profile.Locale,
				"time_zone":      profile.TimeZone,
			},
			Extra: map[string]string{
				"canvas_user_id": strconv.FormatInt(profile.ID, 10),
				"login_id":       profile.LoginID,
				"sis_user_id":    profile.SISUserID,
				"integration_id": profile.IntegrationID,
				"primary_email":  profile.PrimaryEmail,
				"locale":         profile.Locale,
				"time_zone":      profile.TimeZone,
				"short_name":     profile.ShortName,
			},
		}, nil
	}
}

// CanvasAPIClient provides methods to call Canvas API endpoints.
// Use this after authentication to fetch courses, assignments, grades, etc.
type CanvasAPIClient struct {
	client    *http.Client
	canvasURL string
}

// NewCanvasAPIClient creates a client for calling Canvas API.
// Requires a valid OAuth2 access token and the Canvas instance URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	canvasClient := oauth2.NewCanvasAPIClient(user.AccessToken, "https://myschool.instructure.com")
//	courses, err := canvasClient.GetCourses(r.Context())
func NewCanvasAPIClient(accessToken, canvasURL string) *CanvasAPIClient {
	return &CanvasAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		canvasURL: canvasURL,
	}
}

// CanvasCourse represents a course in Canvas.
type CanvasCourse struct {
	ID                   int64   `json:"id"`
	Name                 string  `json:"name"`
	CourseCode           string  `json:"course_code"`
	WorkflowState        string  `json:"workflow_state"` // available, completed, deleted
	AccountID            int64   `json:"account_id"`
	EnrollmentTermID     int64   `json:"enrollment_term_id"`
	StartAt              string  `json:"start_at"`
	EndAt                string  `json:"end_at"`
	SISCourseID          string  `json:"sis_course_id"`
	IntegrationID        string  `json:"integration_id"`
	TimeZone             string  `json:"time_zone"`
	DefaultView          string  `json:"default_view"`
	PublicSyllabus       bool    `json:"public_syllabus"`
	StorageQuotaMB       int64   `json:"storage_quota_mb"`
	TotalStudents        int     `json:"total_students"`
	Enrollments          []CanvasEnrollment `json:"enrollments,omitempty"` // Only if include[]=enrollments
}

// CanvasEnrollment represents an enrollment in a Canvas course.
type CanvasEnrollment struct {
	ID                    int64  `json:"id"`
	CourseID              int64  `json:"course_id"`
	CourseSectionID       int64  `json:"course_section_id"`
	UserID                int64  `json:"user_id"`
	Type                  string `json:"type"` // StudentEnrollment, TeacherEnrollment, TaEnrollment, etc.
	Role                  string `json:"role"`
	RoleID                int64  `json:"role_id"`
	EnrollmentState       string `json:"enrollment_state"` // active, invited, completed
	LimitPrivilegesToCourseSection bool `json:"limit_privileges_to_course_section"`
	SISImportID           int64  `json:"sis_import_id"`
	SISSectionID          string `json:"sis_section_id"`
	SISUserID             string `json:"sis_user_id"`
}

// GetCourses fetches the courses for the authenticated user.
// By default returns active courses. Use state parameter to filter.
func (c *CanvasAPIClient) GetCourses(ctx context.Context) ([]CanvasCourse, error) {
	return c.GetCoursesWithOptions(ctx, "active", true)
}

// GetCoursesWithOptions fetches courses with specific state and enrollment details.
// state: "active", "available", "completed", "all"
// includeEnrollments: include enrollment type in response
func (c *CanvasAPIClient) GetCoursesWithOptions(ctx context.Context, state string, includeEnrollments bool) ([]CanvasCourse, error) {
	url := c.canvasURL + "/api/v1/courses?per_page=100"
	if state != "" {
		url += "&state[]=" + state
	}
	if includeEnrollments {
		url += "&include[]=enrollments"
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []CanvasCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return courses, nil
}

// GetCourse fetches a specific course by ID.
func (c *CanvasAPIClient) GetCourse(ctx context.Context, courseID int64) (*CanvasCourse, error) {
	url := fmt.Sprintf("%s/api/v1/courses/%d?include[]=enrollments", c.canvasURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch course: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var course CanvasCourse
	if err := json.NewDecoder(resp.Body).Decode(&course); err != nil {
		return nil, fmt.Errorf("failed to decode course: %w", err)
	}

	return &course, nil
}

// CanvasUser represents a user in Canvas.
type CanvasUser struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	ShortName     string `json:"short_name"`
	SortableName  string `json:"sortable_name"`
	LoginID       string `json:"login_id"`
	SISUserID     string `json:"sis_user_id"`
	IntegrationID string `json:"integration_id"`
	Email         string `json:"email"`
	AvatarURL     string `json:"avatar_url"`
}

// GetCourseStudents fetches the students enrolled in a course.
// Requires appropriate permissions (teacher/TA/admin).
func (c *CanvasAPIClient) GetCourseStudents(ctx context.Context, courseID int64) ([]CanvasUser, error) {
	url := fmt.Sprintf("%s/api/v1/courses/%d/users?enrollment_type[]=student&per_page=100", c.canvasURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var users []CanvasUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return users, nil
}

// GetCourseTeachers fetches the teachers enrolled in a course.
func (c *CanvasAPIClient) GetCourseTeachers(ctx context.Context, courseID int64) ([]CanvasUser, error) {
	url := fmt.Sprintf("%s/api/v1/courses/%d/users?enrollment_type[]=teacher&per_page=100", c.canvasURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teachers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var users []CanvasUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode teachers: %w", err)
	}

	return users, nil
}

// CanvasAssignment represents an assignment in Canvas.
type CanvasAssignment struct {
	ID                  int64    `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	DueAt               string   `json:"due_at"`
	UnlockAt            string   `json:"unlock_at"`
	LockAt              string   `json:"lock_at"`
	PointsPossible      float64  `json:"points_possible"`
	CourseID            int64    `json:"course_id"`
	SubmissionTypes     []string `json:"submission_types"`
	GradingType         string   `json:"grading_type"`
	AssignmentGroupID   int64    `json:"assignment_group_id"`
	Published           bool     `json:"published"`
	OnlyVisibleToOverrides bool  `json:"only_visible_to_overrides"`
	HTMLURL             string   `json:"html_url"`
}

// GetCourseAssignments fetches assignments for a course.
func (c *CanvasAPIClient) GetCourseAssignments(ctx context.Context, courseID int64) ([]CanvasAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/courses/%d/assignments?per_page=100", c.canvasURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var assignments []CanvasAssignment
	if err := json.NewDecoder(resp.Body).Decode(&assignments); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return assignments, nil
}

// CanvasSubmission represents a submission for an assignment.
type CanvasSubmission struct {
	ID                int64   `json:"id"`
	AssignmentID      int64   `json:"assignment_id"`
	UserID            int64   `json:"user_id"`
	SubmittedAt       string  `json:"submitted_at"`
	Score             float64 `json:"score"`
	Grade             string  `json:"grade"`
	GradedAt          string  `json:"graded_at"`
	WorkflowState     string  `json:"workflow_state"` // submitted, graded, pending_review
	Late              bool    `json:"late"`
	Missing           bool    `json:"missing"`
	Attempt           int     `json:"attempt"`
	SubmissionType    string  `json:"submission_type"`
	PreviewURL        string  `json:"preview_url"`
	URL               string  `json:"url"`
	Body              string  `json:"body"`
}

// GetMySubmissions fetches the authenticated user's submissions for a course.
func (c *CanvasAPIClient) GetMySubmissions(ctx context.Context, courseID int64) ([]CanvasSubmission, error) {
	url := fmt.Sprintf("%s/api/v1/courses/%d/students/submissions?student_ids[]=self&per_page=100", c.canvasURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var submissions []CanvasSubmission
	if err := json.NewDecoder(resp.Body).Decode(&submissions); err != nil {
		return nil, fmt.Errorf("failed to decode submissions: %w", err)
	}

	return submissions, nil
}

// GetEnrollmentType returns the enrollment type for a user in a course.
// Returns empty string if not found in the course.
func GetCanvasEnrollmentType(course *CanvasCourse) CanvasEnrollmentType {
	if len(course.Enrollments) == 0 {
		return ""
	}
	// Return the first enrollment type (user may have multiple)
	enrollment := course.Enrollments[0]
	switch enrollment.Type {
	case "StudentEnrollment":
		return CanvasEnrollmentStudent
	case "TeacherEnrollment":
		return CanvasEnrollmentTeacher
	case "TaEnrollment":
		return CanvasEnrollmentTA
	case "ObserverEnrollment":
		return CanvasEnrollmentObserver
	case "DesignerEnrollment":
		return CanvasEnrollmentDesigner
	default:
		return CanvasEnrollmentType(enrollment.Type)
	}
}

// IsCanvasTeacher returns true if the user is a teacher in the course.
func IsCanvasTeacher(course *CanvasCourse) bool {
	for _, e := range course.Enrollments {
		if e.Type == "TeacherEnrollment" {
			return true
		}
	}
	return false
}

// IsCanvasStudent returns true if the user is a student in the course.
func IsCanvasStudent(course *CanvasCourse) bool {
	for _, e := range course.Enrollments {
		if e.Type == "StudentEnrollment" {
			return true
		}
	}
	return false
}

// IsCanvasTA returns true if the user is a TA in the course.
func IsCanvasTA(course *CanvasCourse) bool {
	for _, e := range course.Enrollments {
		if e.Type == "TaEnrollment" {
			return true
		}
	}
	return false
}

// HasCanvasTeacherRole returns true if the user has any teaching role (teacher or TA).
func HasCanvasTeacherRole(course *CanvasCourse) bool {
	return IsCanvasTeacher(course) || IsCanvasTA(course)
}
