// auth/oauth2/moodle.go
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
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// MoodleRoleArchetype represents a role archetype in Moodle.
// Role archetypes are the standard role types that custom roles are based on.
type MoodleRoleArchetype string

const (
	// MoodleRoleArchetypeManager represents site-wide management capabilities.
	MoodleRoleArchetypeManager MoodleRoleArchetype = "manager"

	// MoodleRoleArchetypeCoursecreator can create courses and teach in them.
	MoodleRoleArchetypeCoursecreator MoodleRoleArchetype = "coursecreator"

	// MoodleRoleArchetypeEditingteacher can edit course content and grade students.
	MoodleRoleArchetypeEditingteacher MoodleRoleArchetype = "editingteacher"

	// MoodleRoleArchetypeTeacher can grade students but not edit course content.
	MoodleRoleArchetypeTeacher MoodleRoleArchetype = "teacher"

	// MoodleRoleArchetypeStudent is enrolled to participate in a course.
	MoodleRoleArchetypeStudent MoodleRoleArchetype = "student"

	// MoodleRoleArchetypeGuest can view courses but not participate.
	MoodleRoleArchetypeGuest MoodleRoleArchetype = "guest"

	// MoodleRoleArchetypeUser is an authenticated user with no specific role.
	MoodleRoleArchetypeUser MoodleRoleArchetype = "user"

	// MoodleRoleArchetypeFrontpage is for the site front page.
	MoodleRoleArchetypeFrontpage MoodleRoleArchetype = "frontpage"

	// MoodleRoleArchetypeUnknown represents an unknown role archetype.
	MoodleRoleArchetypeUnknown MoodleRoleArchetype = "unknown"
)

// MoodleConfig holds configuration for Moodle OAuth2 authentication.
type MoodleConfig struct {
	// ClientID is the Moodle OAuth2 client ID.
	ClientID string

	// ClientSecret is the Moodle OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Moodle.
	// Example: "https://myapp.com/auth/moodle/callback"
	RedirectURL string

	// SiteURL is your Moodle site's base URL.
	// Example: "https://moodle.myschool.edu" or "https://learn.company.com"
	// Do NOT include trailing slashes.
	SiteURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: basic user info
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

// Moodle creates a new OAuth2 provider configured for Moodle LMS authentication.
//
// Moodle is the world's most widely used open-source Learning Management System,
// deployed by schools, universities, and organizations worldwide. It supports
// OAuth 2 authentication through its built-in OAuth 2 services feature.
//
// Setup in Moodle (requires admin access):
//  1. Go to Site administration → Server → OAuth 2 services
//  2. Click "Create new custom service" or configure an existing OAuth 2 issuer
//  3. For custom service:
//     - Name: Your application name
//     - Client ID: Will be generated
//     - Client secret: Will be generated
//  4. Configure the service endpoints:
//     - Authorization endpoint: {siteurl}/admin/oauth2callback.php
//     - Token endpoint: {siteurl}/oauth/token.php
//  5. Set the Redirect URI to your callback URL
//  6. Enable the service
//
// Alternatively, Moodle 3.9+ supports OAuth 2 client registration:
//  1. Go to Site administration → Server → OAuth 2 services
//  2. Click "OAuth 2 identity issuers"
//  3. Create a new issuer or use existing (Google, Microsoft, etc.)
//
// Usage in BuildHandler:
//
//	moodleAuth, err := oauth2.Moodle(oauth2.MoodleConfig{
//	    ClientID:     appCfg.MoodleClientID,
//	    ClientSecret: appCfg.MoodleClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/moodle/callback",
//	    SiteURL:      appCfg.MoodleSiteURL, // e.g., "https://moodle.myschool.edu"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/moodle/login", moodleAuth.LoginHandler())
//	r.Get("/auth/moodle/callback", moodleAuth.CallbackHandler())
//	r.Get("/auth/moodle/logout", moodleAuth.LogoutHandler())
//
// The User.Extra map contains Moodle-specific fields:
//   - "moodle_id": Moodle user ID
//   - "username": Moodle username
//   - "first_name": User's first name
//   - "last_name": User's last name
//   - "full_name": User's full name
//   - "lang": User's preferred language
//   - "timezone": User's timezone
//   - "city": User's city
//   - "country": User's country code
//   - "institution": User's institution
//   - "department": User's department
//   - "suspended": Whether user account is suspended ("true"/"false")
//   - "confirmed": Whether user account is confirmed ("true"/"false")
func Moodle(cfg MoodleConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/moodle: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/moodle: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/moodle: RedirectURL is required")
	}
	if cfg.SiteURL == "" {
		return nil, errors.New("oauth2/moodle: SiteURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/moodle: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/moodle: StateStore is required")
	}

	// Clean up site URL - remove trailing slashes
	siteURL := strings.TrimSuffix(cfg.SiteURL, "/")

	// Build OAuth2 endpoint URLs
	// Moodle uses standard OAuth2 endpoints
	endpoint := oauth2.Endpoint{
		AuthURL:  siteURL + "/admin/oauth2callback.php",
		TokenURL: siteURL + "/admin/tool/oauth2/token.php",
	}

	// Default scopes - Moodle uses custom scope names
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

	// Create a fetcher that uses the Moodle Web Services API
	fetchUserInfo := createMoodleUserInfoFetcher(siteURL)

	providerCfg := &Config{
		ProviderName:  "moodle",
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

// moodleUserInfo represents the response from Moodle user info endpoint.
type moodleUserInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"firstname"`
	LastName    string `json:"lastname"`
	FullName    string `json:"fullname"`
	Email       string `json:"email"`
	Lang        string `json:"lang"`
	Timezone    string `json:"timezone"`
	City        string `json:"city"`
	Country     string `json:"country"`
	ProfileURL  string `json:"profileimageurl"`
	Institution string `json:"institution"`
	Department  string `json:"department"`
	Suspended   bool   `json:"suspended"`
	Confirmed   bool   `json:"confirmed"`
}

// createMoodleUserInfoFetcher creates a UserInfoFetcher for Moodle.
func createMoodleUserInfoFetcher(siteURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info using Moodle Web Services API
		// The core_webservice_get_site_info function returns current user info
		resp, err := client.Get(siteURL + "/webservice/rest/server.php?wsfunction=core_webservice_get_site_info&moodlewsrestformat=json")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Moodle returns site info with user details
		var siteInfo struct {
			UserID       int64  `json:"userid"`
			Username     string `json:"username"`
			FirstName    string `json:"firstname"`
			LastName     string `json:"lastname"`
			FullName     string `json:"fullname"`
			Lang         string `json:"lang"`
			UserPictureURL string `json:"userpictureurl"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&siteInfo); err != nil {
			return nil, fmt.Errorf("failed to decode site info: %w", err)
		}

		// Fetch additional user details
		userResp, err := client.Get(fmt.Sprintf("%s/webservice/rest/server.php?wsfunction=core_user_get_users_by_field&field=id&values[0]=%d&moodlewsrestformat=json", siteURL, siteInfo.UserID))
		if err != nil {
			// If we can't get additional info, use what we have from site info
			return &User{
				ID:            fmt.Sprintf("%d", siteInfo.UserID),
				Email:         "", // Not available from site info
				EmailVerified: false,
				Name:          siteInfo.FullName,
				Picture:       siteInfo.UserPictureURL,
				Raw: map[string]any{
					"id":       siteInfo.UserID,
					"username": siteInfo.Username,
					"fullname": siteInfo.FullName,
				},
				Extra: map[string]string{
					"moodle_id":  fmt.Sprintf("%d", siteInfo.UserID),
					"username":   siteInfo.Username,
					"first_name": siteInfo.FirstName,
					"last_name":  siteInfo.LastName,
					"full_name":  siteInfo.FullName,
					"lang":       siteInfo.Lang,
				},
			}, nil
		}
		defer userResp.Body.Close()

		if userResp.StatusCode == http.StatusOK {
			var users []moodleUserInfo
			if err := json.NewDecoder(userResp.Body).Decode(&users); err == nil && len(users) > 0 {
				info := users[0]

				// Build extra fields
				extra := map[string]string{
					"moodle_id":   fmt.Sprintf("%d", info.ID),
					"username":    info.Username,
					"first_name":  info.FirstName,
					"last_name":   info.LastName,
					"full_name":   info.FullName,
					"lang":        info.Lang,
					"timezone":    info.Timezone,
					"city":        info.City,
					"country":     info.Country,
					"institution": info.Institution,
					"department":  info.Department,
					"suspended":   fmt.Sprintf("%t", info.Suspended),
					"confirmed":   fmt.Sprintf("%t", info.Confirmed),
				}

				return &User{
					ID:            fmt.Sprintf("%d", info.ID),
					Email:         info.Email,
					EmailVerified: info.Confirmed && info.Email != "",
					Name:          info.FullName,
					Picture:       info.ProfileURL,
					Raw: map[string]any{
						"id":          info.ID,
						"username":    info.Username,
						"firstname":   info.FirstName,
						"lastname":    info.LastName,
						"fullname":    info.FullName,
						"email":       info.Email,
						"lang":        info.Lang,
						"timezone":    info.Timezone,
						"city":        info.City,
						"country":     info.Country,
						"institution": info.Institution,
						"department":  info.Department,
						"suspended":   info.Suspended,
						"confirmed":   info.Confirmed,
					},
					Extra: extra,
				}, nil
			}
		}

		// Fall back to site info if user lookup fails
		return &User{
			ID:            fmt.Sprintf("%d", siteInfo.UserID),
			Email:         "",
			EmailVerified: false,
			Name:          siteInfo.FullName,
			Picture:       siteInfo.UserPictureURL,
			Raw: map[string]any{
				"id":       siteInfo.UserID,
				"username": siteInfo.Username,
				"fullname": siteInfo.FullName,
			},
			Extra: map[string]string{
				"moodle_id":  fmt.Sprintf("%d", siteInfo.UserID),
				"username":   siteInfo.Username,
				"first_name": siteInfo.FirstName,
				"last_name":  siteInfo.LastName,
				"full_name":  siteInfo.FullName,
				"lang":       siteInfo.Lang,
			},
		}, nil
	}
}

// MoodleAPIClient provides methods to call Moodle Web Services API endpoints.
// Use this after authentication for additional Moodle API operations.
type MoodleAPIClient struct {
	client  *http.Client
	siteURL string
	token   string
}

// NewMoodleAPIClient creates a client for calling Moodle Web Services API.
// Requires a valid OAuth2 access token and the Moodle site URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	moodleClient := oauth2.NewMoodleAPIClient(user.AccessToken, "https://moodle.myschool.edu")
//	courses, err := moodleClient.GetUserCourses(r.Context(), userID)
func NewMoodleAPIClient(accessToken, siteURL string) *MoodleAPIClient {
	// Clean up site URL
	siteURL = strings.TrimSuffix(siteURL, "/")

	return &MoodleAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		siteURL: siteURL,
		token:   accessToken,
	}
}

// callWebService makes a call to Moodle's Web Services API.
func (c *MoodleAPIClient) callWebService(ctx context.Context, function string, params url.Values) (*http.Response, error) {
	baseURL := c.siteURL + "/webservice/rest/server.php"
	params.Set("wsfunction", function)
	params.Set("moodlewsrestformat", "json")

	fullURL := baseURL + "?" + params.Encode()
	return c.client.Get(fullURL)
}

// MoodleCourse represents a course in Moodle.
type MoodleCourse struct {
	ID             int64  `json:"id"`
	ShortName      string `json:"shortname"`
	FullName       string `json:"fullname"`
	DisplayName    string `json:"displayname"`
	Summary        string `json:"summary"`
	CategoryID     int64  `json:"categoryid"`
	CategoryName   string `json:"categoryname,omitempty"`
	StartDate      int64  `json:"startdate"`
	EndDate        int64  `json:"enddate"`
	Visible        bool   `json:"visible"`
	Format         string `json:"format"`
	EnrolledUserCount int `json:"enrolledusercount,omitempty"`
	Progress       *float64 `json:"progress,omitempty"`
	Completed      *bool   `json:"completed,omitempty"`
}

// MoodleEnrollment represents a user's enrollment in a course.
type MoodleEnrollment struct {
	CourseID    int64             `json:"id"`
	ShortName   string            `json:"shortname"`
	FullName    string            `json:"fullname"`
	EnrolledUserCount int         `json:"enrolledusercount"`
	Roles       []MoodleRole      `json:"roles,omitempty"`
}

// MoodleRole represents a role in Moodle.
type MoodleRole struct {
	RoleID    int64  `json:"roleid"`
	Name      string `json:"name"`
	ShortName string `json:"shortname"`
	SortOrder int    `json:"sortorder"`
}

// MoodleUser represents a user in Moodle.
type MoodleUser struct {
	ID            int64        `json:"id"`
	Username      string       `json:"username"`
	FirstName     string       `json:"firstname"`
	LastName      string       `json:"lastname"`
	FullName      string       `json:"fullname"`
	Email         string       `json:"email"`
	ProfileURL    string       `json:"profileimageurl"`
	Department    string       `json:"department"`
	Institution   string       `json:"institution"`
	Roles         []MoodleRole `json:"roles,omitempty"`
	EnrolledCourses []MoodleCourse `json:"enrolledcourses,omitempty"`
}

// MoodleGrade represents a grade item in Moodle.
type MoodleGrade struct {
	CourseID     int64   `json:"courseid"`
	ItemID       int64   `json:"id"`
	ItemName     string  `json:"itemname"`
	ItemType     string  `json:"itemtype"`
	ItemModule   string  `json:"itemmodule"`
	CategoryID   int64   `json:"categoryid"`
	GradeRaw     float64 `json:"graderaw"`
	GradeMin     float64 `json:"grademin"`
	GradeMax     float64 `json:"grademax"`
	GradeFormatted string `json:"gradeformatted"`
	Percentage   float64 `json:"percentageformatted,omitempty"`
	Feedback     string  `json:"feedback"`
	Locked       bool    `json:"locked"`
	Hidden       bool    `json:"hidden"`
}

// MoodleAssignment represents an assignment in Moodle.
type MoodleAssignment struct {
	ID                int64  `json:"id"`
	CourseID          int64  `json:"course"`
	Name              string `json:"name"`
	Intro             string `json:"intro"`
	DueDate           int64  `json:"duedate"`
	AllowSubmissionsFrom int64 `json:"allowsubmissionsfromdate"`
	GradeMax          int    `json:"grade"`
	TimeModified      int64  `json:"timemodified"`
	CutoffDate        int64  `json:"cutoffdate"`
	GradingDueDate    int64  `json:"gradingduedate"`
	SubmissionDrafts  int    `json:"submissiondrafts"`
	TeamSubmission    int    `json:"teamsubmission"`
}

// MoodleSubmission represents a submission for an assignment.
type MoodleSubmission struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"userid"`
	Status       string `json:"status"` // new, draft, submitted, reopened
	TimeCreated  int64  `json:"timecreated"`
	TimeModified int64  `json:"timemodified"`
	AttemptNumber int   `json:"attemptnumber"`
	GroupID      int64  `json:"groupid"`
}

// MoodleCategory represents a course category in Moodle.
type MoodleCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      int64  `json:"parent"`
	CourseCount int    `json:"coursecount"`
	Depth       int    `json:"depth"`
	Path        string `json:"path"`
}

// GetUserCourses fetches the courses a user is enrolled in.
func (c *MoodleAPIClient) GetUserCourses(ctx context.Context, userID int64) ([]MoodleCourse, error) {
	params := url.Values{}
	params.Set("userid", fmt.Sprintf("%d", userID))

	resp, err := c.callWebService(ctx, "core_enrol_get_users_courses", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []MoodleCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return courses, nil
}

// GetCourse fetches details for a specific course.
func (c *MoodleAPIClient) GetCourse(ctx context.Context, courseID int64) (*MoodleCourse, error) {
	params := url.Values{}
	params.Set("options[ids][0]", fmt.Sprintf("%d", courseID))

	resp, err := c.callWebService(ctx, "core_course_get_courses", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch course: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []MoodleCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode course: %w", err)
	}

	if len(courses) == 0 {
		return nil, fmt.Errorf("course not found: %d", courseID)
	}

	return &courses[0], nil
}

// GetCourseEnrolledUsers fetches users enrolled in a course.
func (c *MoodleAPIClient) GetCourseEnrolledUsers(ctx context.Context, courseID int64) ([]MoodleUser, error) {
	params := url.Values{}
	params.Set("courseid", fmt.Sprintf("%d", courseID))

	resp, err := c.callWebService(ctx, "core_enrol_get_enrolled_users", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrolled users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var users []MoodleUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode enrolled users: %w", err)
	}

	return users, nil
}

// GetUserGrades fetches grades for a user in a course.
func (c *MoodleAPIClient) GetUserGrades(ctx context.Context, courseID, userID int64) ([]MoodleGrade, error) {
	params := url.Values{}
	params.Set("courseid", fmt.Sprintf("%d", courseID))
	params.Set("userid", fmt.Sprintf("%d", userID))

	resp, err := c.callWebService(ctx, "gradereport_user_get_grade_items", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		UserGrades []struct {
			GradeItems []MoodleGrade `json:"gradeitems"`
		} `json:"usergrades"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	if len(response.UserGrades) == 0 {
		return []MoodleGrade{}, nil
	}

	return response.UserGrades[0].GradeItems, nil
}

// GetCourseAssignments fetches assignments for a course.
func (c *MoodleAPIClient) GetCourseAssignments(ctx context.Context, courseID int64) ([]MoodleAssignment, error) {
	params := url.Values{}
	params.Set("courseids[0]", fmt.Sprintf("%d", courseID))

	resp, err := c.callWebService(ctx, "mod_assign_get_assignments", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Courses []struct {
			Assignments []MoodleAssignment `json:"assignments"`
		} `json:"courses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	if len(response.Courses) == 0 {
		return []MoodleAssignment{}, nil
	}

	return response.Courses[0].Assignments, nil
}

// GetAssignmentSubmissions fetches submissions for an assignment.
func (c *MoodleAPIClient) GetAssignmentSubmissions(ctx context.Context, assignmentID int64) ([]MoodleSubmission, error) {
	params := url.Values{}
	params.Set("assignmentids[0]", fmt.Sprintf("%d", assignmentID))

	resp, err := c.callWebService(ctx, "mod_assign_get_submissions", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Assignments []struct {
			Submissions []MoodleSubmission `json:"submissions"`
		} `json:"assignments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode submissions: %w", err)
	}

	if len(response.Assignments) == 0 {
		return []MoodleSubmission{}, nil
	}

	return response.Assignments[0].Submissions, nil
}

// GetCategories fetches course categories.
func (c *MoodleAPIClient) GetCategories(ctx context.Context) ([]MoodleCategory, error) {
	params := url.Values{}

	resp, err := c.callWebService(ctx, "core_course_get_categories", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var categories []MoodleCategory
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %w", err)
	}

	return categories, nil
}

// GetUserRolesInCourse fetches a user's roles in a specific course.
func (c *MoodleAPIClient) GetUserRolesInCourse(ctx context.Context, courseID, userID int64) ([]MoodleRole, error) {
	// Get enrolled users and find the specific user
	users, err := c.GetCourseEnrolledUsers(ctx, courseID)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == userID {
			return user.Roles, nil
		}
	}

	return []MoodleRole{}, nil
}

// GetMoodleUserID returns the Moodle user ID from the user.
func GetMoodleUserID(user *User) string {
	return user.Extra["moodle_id"]
}

// GetMoodleUsername returns the Moodle username.
func GetMoodleUsername(user *User) string {
	return user.Extra["username"]
}

// GetMoodleInstitution returns the user's institution.
func GetMoodleInstitution(user *User) string {
	return user.Extra["institution"]
}

// GetMoodleDepartment returns the user's department.
func GetMoodleDepartment(user *User) string {
	return user.Extra["department"]
}

// IsMoodleSuspended checks if the user account is suspended.
func IsMoodleSuspended(user *User) bool {
	return user.Extra["suspended"] == "true"
}

// IsMoodleConfirmed checks if the user account is confirmed.
func IsMoodleConfirmed(user *User) bool {
	return user.Extra["confirmed"] == "true"
}

// IsMoodleEditingTeacher checks if a user has the editing teacher role in a course.
func IsMoodleEditingTeacher(roles []MoodleRole) bool {
	for _, role := range roles {
		if role.ShortName == "editingteacher" {
			return true
		}
	}
	return false
}

// IsMoodleTeacher checks if a user has the non-editing teacher role in a course.
func IsMoodleTeacher(roles []MoodleRole) bool {
	for _, role := range roles {
		if role.ShortName == "teacher" {
			return true
		}
	}
	return false
}

// IsMoodleStudent checks if a user has the student role in a course.
func IsMoodleStudent(roles []MoodleRole) bool {
	for _, role := range roles {
		if role.ShortName == "student" {
			return true
		}
	}
	return false
}

// IsMoodleManager checks if a user has the manager role.
func IsMoodleManager(roles []MoodleRole) bool {
	for _, role := range roles {
		if role.ShortName == "manager" {
			return true
		}
	}
	return false
}

// IsMoodleCourseCreator checks if a user has the course creator role.
func IsMoodleCourseCreator(roles []MoodleRole) bool {
	for _, role := range roles {
		if role.ShortName == "coursecreator" {
			return true
		}
	}
	return false
}

// HasMoodleTeachingRole checks if a user has any teaching role (editingteacher or teacher).
func HasMoodleTeachingRole(roles []MoodleRole) bool {
	return IsMoodleEditingTeacher(roles) || IsMoodleTeacher(roles)
}

// GetMoodleRoleArchetype returns the role archetype for a given role.
func GetMoodleRoleArchetype(role MoodleRole) MoodleRoleArchetype {
	switch role.ShortName {
	case "manager":
		return MoodleRoleArchetypeManager
	case "coursecreator":
		return MoodleRoleArchetypeCoursecreator
	case "editingteacher":
		return MoodleRoleArchetypeEditingteacher
	case "teacher":
		return MoodleRoleArchetypeTeacher
	case "student":
		return MoodleRoleArchetypeStudent
	case "guest":
		return MoodleRoleArchetypeGuest
	case "user":
		return MoodleRoleArchetypeUser
	case "frontpage":
		return MoodleRoleArchetypeFrontpage
	default:
		return MoodleRoleArchetypeUnknown
	}
}

// GetHighestMoodleRole returns the highest-privilege role from a list of roles.
// Order: manager > coursecreator > editingteacher > teacher > student > guest > user.
func GetHighestMoodleRole(roles []MoodleRole) *MoodleRole {
	if len(roles) == 0 {
		return nil
	}

	roleOrder := map[string]int{
		"manager":        1,
		"coursecreator":  2,
		"editingteacher": 3,
		"teacher":        4,
		"student":        5,
		"guest":          6,
		"user":           7,
		"frontpage":      8,
	}

	var highest *MoodleRole
	highestOrder := 999

	for i := range roles {
		order, ok := roleOrder[roles[i].ShortName]
		if ok && order < highestOrder {
			highestOrder = order
			highest = &roles[i]
		}
	}

	if highest == nil && len(roles) > 0 {
		return &roles[0]
	}

	return highest
}
