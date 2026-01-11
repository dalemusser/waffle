// auth/oauth2/blackboard.go
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
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// BlackboardUserRole represents a user's role in Blackboard.
type BlackboardUserRole string

const (
	// BlackboardUserRoleStudent represents a student in Blackboard.
	BlackboardUserRoleStudent BlackboardUserRole = "Student"

	// BlackboardUserRoleInstructor represents an instructor/teacher in Blackboard.
	BlackboardUserRoleInstructor BlackboardUserRole = "Instructor"

	// BlackboardUserRoleTeachingAssistant represents a teaching assistant in Blackboard.
	BlackboardUserRoleTeachingAssistant BlackboardUserRole = "TeachingAssistant"

	// BlackboardUserRoleCourseBuilder represents a course builder in Blackboard.
	BlackboardUserRoleCourseBuilder BlackboardUserRole = "CourseBuilder"

	// BlackboardUserRoleGrader represents a grader in Blackboard.
	BlackboardUserRoleGrader BlackboardUserRole = "Grader"

	// BlackboardUserRoleGuest represents a guest user in Blackboard.
	BlackboardUserRoleGuest BlackboardUserRole = "Guest"

	// BlackboardUserRoleSystemAdmin represents a system administrator in Blackboard.
	BlackboardUserRoleSystemAdmin BlackboardUserRole = "SystemAdmin"

	// BlackboardUserRoleSystemSupport represents a system support user in Blackboard.
	BlackboardUserRoleSystemSupport BlackboardUserRole = "SystemSupport"

	// BlackboardUserRoleAccountAdmin represents an account administrator in Blackboard.
	BlackboardUserRoleAccountAdmin BlackboardUserRole = "AccountAdmin"
)

// BlackboardConfig holds configuration for Blackboard OAuth2 authentication.
type BlackboardConfig struct {
	// ClientID is the Blackboard OAuth2 application key.
	ClientID string

	// ClientSecret is the Blackboard OAuth2 application secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Blackboard.
	// Example: "https://myapp.com/auth/blackboard/callback"
	RedirectURL string

	// Domain is your Blackboard Learn instance domain.
	// Example: "myschool.blackboard.com" or "learn.myschool.edu"
	// Do NOT include https:// prefix.
	Domain string

	// Scopes are the OAuth2 scopes to request.
	// Default: read (basic user info)
	// Available scopes depend on your Blackboard REST API registration.
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

// Blackboard creates a new OAuth2 provider configured for Blackboard Learn authentication.
//
// Blackboard Learn is one of the most widely used learning management systems (LMS) in
// higher education and K-12. It provides course management, content delivery, assessments,
// and grade tracking.
//
// Setup in Blackboard Developer Portal:
//  1. Go to https://developer.blackboard.com/
//  2. Create a developer account if you don't have one
//  3. Register a new application
//  4. Select the APIs you need access to
//  5. Set your redirect URI
//  6. Note your Application Key (ClientID) and Secret
//  7. Work with your institution's Blackboard admin to enable your app
//
// Important: Blackboard uses institution-specific domains. Each school has its own
// Blackboard instance with its own domain (e.g., learn.myschool.edu).
//
// Usage in BuildHandler:
//
//	blackboardAuth, err := oauth2.Blackboard(oauth2.BlackboardConfig{
//	    ClientID:     appCfg.BlackboardClientID,
//	    ClientSecret: appCfg.BlackboardClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/blackboard/callback",
//	    Domain:       appCfg.BlackboardDomain, // e.g., "learn.myschool.edu"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/blackboard/login", blackboardAuth.LoginHandler())
//	r.Get("/auth/blackboard/callback", blackboardAuth.CallbackHandler())
//	r.Get("/auth/blackboard/logout", blackboardAuth.LogoutHandler())
//
// The User.Extra map contains Blackboard-specific fields:
//   - "blackboard_user_id": Blackboard user ID
//   - "blackboard_uuid": Blackboard user UUID
//   - "user_name": Blackboard username
//   - "given_name": User's first name
//   - "family_name": User's last name
//   - "institution_roles": Comma-separated list of institution roles
//   - "system_roles": Comma-separated list of system roles
//   - "student_id": Student ID if applicable
//   - "job_title": User's job title
//   - "department": User's department
func Blackboard(cfg BlackboardConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/blackboard: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/blackboard: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/blackboard: RedirectURL is required")
	}
	if cfg.Domain == "" {
		return nil, errors.New("oauth2/blackboard: Domain is required (e.g., learn.myschool.edu)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/blackboard: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/blackboard: StateStore is required")
	}

	// Clean up domain - remove protocol if present
	domain := cfg.Domain
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	baseURL := fmt.Sprintf("https://%s", domain)

	endpoint := oauth2.Endpoint{
		AuthURL:  baseURL + "/learn/api/public/v1/oauth2/authorizationcode",
		TokenURL: baseURL + "/learn/api/public/v1/oauth2/token",
	}

	// Default scopes for Blackboard
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"read",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	// Create a fetcher that uses the Blackboard REST API
	fetchUserInfo := createBlackboardUserInfoFetcher(baseURL)

	providerCfg := &Config{
		ProviderName:  "blackboard",
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

// blackboardUserInfo represents the response from Blackboard's users/me endpoint.
type blackboardUserInfo struct {
	ID                string                      `json:"id"`       // Primary key
	UUID              string                      `json:"uuid"`     // UUID
	ExternalID        string                      `json:"externalId"`
	DataSourceID      string                      `json:"dataSourceId"`
	UserName          string                      `json:"userName"`
	StudentID         string                      `json:"studentId"`
	EducationLevel    string                      `json:"educationLevel"`
	Gender            string                      `json:"gender"`
	Birthdate         string                      `json:"birthDate"`
	Created           string                      `json:"created"`
	Modified          string                      `json:"modified"`
	LastLogin         string                      `json:"lastLogin"`
	InstitutionRoleIDs []string                   `json:"institutionRoleIds"`
	SystemRoleIDs     []string                    `json:"systemRoleIds"`
	Availability      blackboardAvailability      `json:"availability"`
	Name              blackboardName              `json:"name"`
	Job               blackboardJob               `json:"job"`
	Contact           blackboardContact           `json:"contact"`
	Address           blackboardAddress           `json:"address"`
	Locale            blackboardLocale            `json:"locale"`
	Avatar            blackboardAvatar            `json:"avatar"`
}

type blackboardAvailability struct {
	Available string `json:"available"` // Yes, No, Disabled
}

type blackboardName struct {
	Given       string `json:"given"`
	Family      string `json:"family"`
	Middle      string `json:"middle"`
	Other       string `json:"other"`
	Suffix      string `json:"suffix"`
	Title       string `json:"title"`
	PreferredDisplayName string `json:"preferredDisplayName"`
}

type blackboardJob struct {
	Title      string `json:"title"`
	Department string `json:"department"`
	Company    string `json:"company"`
}

type blackboardContact struct {
	HomePhone    string `json:"homePhone"`
	MobilePhone  string `json:"mobilePhone"`
	BusinessPhone string `json:"businessPhone"`
	BusinessFax  string `json:"businessFax"`
	Email        string `json:"email"`
	WebPage      string `json:"webPage"`
}

type blackboardAddress struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zipCode"`
	Country    string `json:"country"`
}

type blackboardLocale struct {
	ID           string `json:"id"`
	Calendar     string `json:"calendar"`
	FirstDayOfWeek string `json:"firstDayOfWeek"`
}

type blackboardAvatar struct {
	ViewURL   string `json:"viewUrl"`
	Source    string `json:"source"` // Default, Uploaded
}

// createBlackboardUserInfoFetcher creates a UserInfoFetcher for Blackboard.
func createBlackboardUserInfoFetcher(baseURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from Blackboard's users/me endpoint
		resp, err := client.Get(baseURL + "/learn/api/public/v1/users/me")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info blackboardUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		// Build display name
		name := info.Name.PreferredDisplayName
		if name == "" {
			name = strings.TrimSpace(info.Name.Given + " " + info.Name.Family)
		}
		if name == "" {
			name = info.UserName
		}

		// Build institution roles string
		institutionRolesStr := strings.Join(info.InstitutionRoleIDs, ",")
		systemRolesStr := strings.Join(info.SystemRoleIDs, ",")

		// Get avatar URL
		avatarURL := ""
		if info.Avatar.ViewURL != "" {
			avatarURL = baseURL + info.Avatar.ViewURL
		}

		return &User{
			ID:            info.ID,
			Email:         info.Contact.Email,
			EmailVerified: info.Contact.Email != "", // Blackboard doesn't expose verification status
			Name:          name,
			Picture:       avatarURL,
			Raw: map[string]any{
				"id":                  info.ID,
				"uuid":                info.UUID,
				"externalId":          info.ExternalID,
				"dataSourceId":        info.DataSourceID,
				"userName":            info.UserName,
				"studentId":           info.StudentID,
				"educationLevel":      info.EducationLevel,
				"gender":              info.Gender,
				"birthDate":           info.Birthdate,
				"created":             info.Created,
				"modified":            info.Modified,
				"lastLogin":           info.LastLogin,
				"institutionRoleIds":  info.InstitutionRoleIDs,
				"systemRoleIds":       info.SystemRoleIDs,
				"availability":        info.Availability.Available,
				"name":                info.Name,
				"job":                 info.Job,
				"contact":             info.Contact,
				"address":             info.Address,
				"locale":              info.Locale,
				"avatar":              info.Avatar,
			},
			Extra: map[string]string{
				"blackboard_user_id":  info.ID,
				"blackboard_uuid":     info.UUID,
				"user_name":           info.UserName,
				"given_name":          info.Name.Given,
				"family_name":         info.Name.Family,
				"institution_roles":   institutionRolesStr,
				"system_roles":        systemRolesStr,
				"student_id":          info.StudentID,
				"job_title":           info.Job.Title,
				"department":          info.Job.Department,
				"education_level":     info.EducationLevel,
				"external_id":         info.ExternalID,
			},
		}, nil
	}
}

// BlackboardAPIClient provides methods to call Blackboard Learn REST API endpoints.
// Use this after authentication for additional Blackboard API operations.
type BlackboardAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewBlackboardAPIClient creates a client for calling Blackboard Learn REST API.
// Requires a valid OAuth2 access token and the Blackboard domain.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	bbClient := oauth2.NewBlackboardAPIClient(user.AccessToken, "learn.myschool.edu")
//	courses, err := bbClient.GetMyCourses(r.Context())
func NewBlackboardAPIClient(accessToken, domain string) *BlackboardAPIClient {
	// Clean up domain
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	return &BlackboardAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: fmt.Sprintf("https://%s", domain),
	}
}

// BlackboardCourse represents a course in Blackboard.
type BlackboardCourse struct {
	ID                string `json:"id"`
	UUID              string `json:"uuid"`
	ExternalID        string `json:"externalId"`
	DataSourceID      string `json:"dataSourceId"`
	CourseID          string `json:"courseId"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Created           string `json:"created"`
	Modified          string `json:"modified"`
	Organization      bool   `json:"organization"` // true if this is an organization, not a course
	UltraStatus       string `json:"ultraStatus"`  // Undecided, Classic, Ultra, UltraPreview
	AllowGuests       bool   `json:"allowGuests"`
	ClosedComplete    bool   `json:"closedComplete"`
	TermID            string `json:"termId"`
	Availability      struct {
		Available string `json:"available"`
		Duration  struct {
			Type      string `json:"type"` // Continuous, DateRange, FixedNumDays, Term
			Start     string `json:"start"`
			End       string `json:"end"`
			DaysOfUse int    `json:"daysOfUse"`
		} `json:"duration"`
	} `json:"availability"`
	Enrollment struct {
		Type       string `json:"type"` // InstructorLed, SelfEnrollment, EmailEnrollment
		Start      string `json:"start"`
		End        string `json:"end"`
		AccessCode string `json:"accessCode"`
	} `json:"enrollment"`
	Locale struct {
		ID          string `json:"id"`
		ForceLocale bool   `json:"force"`
	} `json:"locale"`
}

// BlackboardCourseMembership represents a user's membership in a course.
type BlackboardCourseMembership struct {
	UserID           string `json:"userId"`
	CourseID         string `json:"courseId"`
	DataSourceID     string `json:"dataSourceId"`
	Created          string `json:"created"`
	Modified         string `json:"modified"`
	CourseRoleID     string `json:"courseRoleId"` // Student, Instructor, TeachingAssistant, CourseBuilder, Grader, Guest
	Availability     struct {
		Available string `json:"available"`
	} `json:"availability"`
	LastAccessed     string `json:"lastAccessed"`
}

// BlackboardCoursesResponse represents the paginated response for courses.
type BlackboardCoursesResponse struct {
	Results []BlackboardCourse `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// BlackboardMembershipsResponse represents the paginated response for memberships.
type BlackboardMembershipsResponse struct {
	Results []BlackboardCourseMembership `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// GetMyCourses fetches all courses the current user is enrolled in.
func (c *BlackboardAPIClient) GetMyCourses(ctx context.Context) ([]BlackboardCourseMembership, error) {
	resp, err := c.client.Get(c.baseURL + "/learn/api/public/v1/users/me/courses")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response BlackboardMembershipsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return response.Results, nil
}

// GetCourse fetches a specific course by ID.
func (c *BlackboardAPIClient) GetCourse(ctx context.Context, courseID string) (*BlackboardCourse, error) {
	url := fmt.Sprintf("%s/learn/api/public/v3/courses/%s", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch course: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var course BlackboardCourse
	if err := json.NewDecoder(resp.Body).Decode(&course); err != nil {
		return nil, fmt.Errorf("failed to decode course: %w", err)
	}

	return &course, nil
}

// BlackboardAssignment represents an assignment in Blackboard.
type BlackboardAssignment struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Instructions    string `json:"instructions"`
	Position        int    `json:"position"`
	GradebookColumnID string `json:"gradeColumnId"`
	DueDate         string `json:"due"`
	Score           struct {
		Possible float64 `json:"possible"`
	} `json:"score"`
	Availability struct {
		Available          string `json:"available"`
		AllowAttemptGrades bool   `json:"allowAttemptGrades"`
		AdaptiveRelease    struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"adaptiveRelease"`
	} `json:"availability"`
}

// BlackboardAssignmentsResponse represents the paginated response for assignments.
type BlackboardAssignmentsResponse struct {
	Results []BlackboardAssignment `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// GetCourseAssignments fetches assignments for a specific course.
func (c *BlackboardAPIClient) GetCourseAssignments(ctx context.Context, courseID string) ([]BlackboardAssignment, error) {
	url := fmt.Sprintf("%s/learn/api/public/v1/courses/%s/contents", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response BlackboardAssignmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode assignments: %w", err)
	}

	return response.Results, nil
}

// BlackboardGrade represents a grade entry in Blackboard.
type BlackboardGrade struct {
	UserID     string  `json:"userId"`
	ColumnID   string  `json:"columnId"`
	Status     string  `json:"status"` // Graded, NeedsGrading, InProgress, Completed, etc.
	Score      float64 `json:"score"`
	Text       string  `json:"text"`
	Notes      string  `json:"notes"`
	Feedback   string  `json:"feedback"`
	Exempt     bool    `json:"exempt"`
	Corrupt    bool    `json:"corrupt"`
	GradeNotation struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	} `json:"gradeNotation"`
	Created    string `json:"created"`
	Modified   string `json:"modified"`
}

// BlackboardGradesResponse represents the paginated response for grades.
type BlackboardGradesResponse struct {
	Results []BlackboardGrade `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// GetMyGrades fetches the current user's grades for a specific course.
func (c *BlackboardAPIClient) GetMyGrades(ctx context.Context, courseID string) ([]BlackboardGrade, error) {
	url := fmt.Sprintf("%s/learn/api/public/v2/courses/%s/gradebook/users/me", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response BlackboardGradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return response.Results, nil
}

// BlackboardGradebookColumn represents a column in the gradebook.
type BlackboardGradebookColumn struct {
	ID             string  `json:"id"`
	ExternalID     string  `json:"externalId"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	ExternalGrade  bool    `json:"externalGrade"`
	Created        string  `json:"created"`
	ContentID      string  `json:"contentId"`
	Score          struct {
		Possible   float64 `json:"possible"`
		DecimalPlaces int  `json:"decimalPlaces"`
	} `json:"score"`
	Availability struct {
		Available string `json:"available"`
	} `json:"availability"`
	Grading struct {
		Type           string `json:"type"` // Attempts, Calculated, Manual
		Due            string `json:"due"`
		AttemptsAllowed int   `json:"attemptsAllowed"`
		ScoringModel   string `json:"scoringModel"` // Last, Highest, Lowest, First, Average
		AnonymousGrading struct {
			Type     string `json:"type"`
			ReleaseAfter string `json:"releaseAfter"`
		} `json:"anonymousGrading"`
	} `json:"grading"`
}

// BlackboardColumnsResponse represents the paginated response for gradebook columns.
type BlackboardColumnsResponse struct {
	Results []BlackboardGradebookColumn `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// GetGradebookColumns fetches the gradebook columns for a course.
func (c *BlackboardAPIClient) GetGradebookColumns(ctx context.Context, courseID string) ([]BlackboardGradebookColumn, error) {
	url := fmt.Sprintf("%s/learn/api/public/v2/courses/%s/gradebook/columns", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gradebook columns: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response BlackboardColumnsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode gradebook columns: %w", err)
	}

	return response.Results, nil
}

// BlackboardTerm represents an academic term in Blackboard.
type BlackboardTerm struct {
	ID           string `json:"id"`
	ExternalID   string `json:"externalId"`
	DataSourceID string `json:"dataSourceId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Availability struct {
		Available string `json:"available"`
		Duration  struct {
			Type  string `json:"type"`
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"duration"`
	} `json:"availability"`
}

// GetTerm fetches a specific term by ID.
func (c *BlackboardAPIClient) GetTerm(ctx context.Context, termID string) (*BlackboardTerm, error) {
	url := fmt.Sprintf("%s/learn/api/public/v1/terms/%s", c.baseURL, termID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch term: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var term BlackboardTerm
	if err := json.NewDecoder(resp.Body).Decode(&term); err != nil {
		return nil, fmt.Errorf("failed to decode term: %w", err)
	}

	return &term, nil
}

// BlackboardAnnouncement represents an announcement in Blackboard.
type BlackboardAnnouncement struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	Position     int    `json:"position"`
	Availability struct {
		Duration struct {
			Type  string `json:"type"`
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"duration"`
	} `json:"availability"`
	ShowAtLogin    bool `json:"showAtLogin"`
	ShowInCourses  bool `json:"showInCourses"`
	Creator        string `json:"creator"`
}

// BlackboardAnnouncementsResponse represents the paginated response for announcements.
type BlackboardAnnouncementsResponse struct {
	Results []BlackboardAnnouncement `json:"results"`
	Paging  struct {
		NextPage string `json:"nextPage"`
	} `json:"paging"`
}

// GetCourseAnnouncements fetches announcements for a specific course.
func (c *BlackboardAPIClient) GetCourseAnnouncements(ctx context.Context, courseID string) ([]BlackboardAnnouncement, error) {
	url := fmt.Sprintf("%s/learn/api/public/v1/courses/%s/announcements", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch announcements: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response BlackboardAnnouncementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode announcements: %w", err)
	}

	return response.Results, nil
}

// GetCourseMembership fetches the current user's membership in a course.
func (c *BlackboardAPIClient) GetCourseMembership(ctx context.Context, courseID string) (*BlackboardCourseMembership, error) {
	url := fmt.Sprintf("%s/learn/api/public/v1/courses/%s/users/me", c.baseURL, courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch membership: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var membership BlackboardCourseMembership
	if err := json.NewDecoder(resp.Body).Decode(&membership); err != nil {
		return nil, fmt.Errorf("failed to decode membership: %w", err)
	}

	return &membership, nil
}

// IsBlackboardStudent checks if the user has a student role based on institution roles.
func IsBlackboardStudent(user *User) bool {
	roles := strings.Split(user.Extra["institution_roles"], ",")
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if strings.Contains(strings.ToLower(role), "student") {
			return true
		}
	}
	return false
}

// IsBlackboardInstructor checks if the user has an instructor role based on institution roles.
func IsBlackboardInstructor(user *User) bool {
	roles := strings.Split(user.Extra["institution_roles"], ",")
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if strings.Contains(strings.ToLower(role), "instructor") ||
			strings.Contains(strings.ToLower(role), "faculty") ||
			strings.Contains(strings.ToLower(role), "teacher") {
			return true
		}
	}
	return false
}

// IsBlackboardAdmin checks if the user has an admin role based on system roles.
func IsBlackboardAdmin(user *User) bool {
	systemRoles := strings.Split(user.Extra["system_roles"], ",")
	for _, role := range systemRoles {
		role = strings.TrimSpace(role)
		if strings.Contains(strings.ToLower(role), "admin") ||
			strings.Contains(strings.ToLower(role), "support") {
			return true
		}
	}
	return false
}

// IsBlackboardStaff checks if the user is staff (non-student).
func IsBlackboardStaff(user *User) bool {
	roles := strings.Split(user.Extra["institution_roles"], ",")
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if strings.Contains(strings.ToLower(role), "staff") ||
			strings.Contains(strings.ToLower(role), "employee") {
			return true
		}
	}
	return false
}

// GetBlackboardInstitutionRoles returns the user's institution roles as a slice.
func GetBlackboardInstitutionRoles(user *User) []string {
	rolesStr := user.Extra["institution_roles"]
	if rolesStr == "" {
		return nil
	}
	return strings.Split(rolesStr, ",")
}

// GetBlackboardSystemRoles returns the user's system roles as a slice.
func GetBlackboardSystemRoles(user *User) []string {
	rolesStr := user.Extra["system_roles"]
	if rolesStr == "" {
		return nil
	}
	return strings.Split(rolesStr, ",")
}

// HasBlackboardInstitutionRole checks if the user has a specific institution role.
func HasBlackboardInstitutionRole(user *User, roleID string) bool {
	roles := GetBlackboardInstitutionRoles(user)
	for _, role := range roles {
		if role == roleID {
			return true
		}
	}
	return false
}

// HasBlackboardSystemRole checks if the user has a specific system role.
func HasBlackboardSystemRole(user *User, roleID string) bool {
	roles := GetBlackboardSystemRoles(user)
	for _, role := range roles {
		if role == roleID {
			return true
		}
	}
	return false
}

// GetBlackboardCourseRole converts a course role string to BlackboardUserRole.
func GetBlackboardCourseRole(roleID string) BlackboardUserRole {
	switch roleID {
	case "Student":
		return BlackboardUserRoleStudent
	case "Instructor":
		return BlackboardUserRoleInstructor
	case "TeachingAssistant":
		return BlackboardUserRoleTeachingAssistant
	case "CourseBuilder":
		return BlackboardUserRoleCourseBuilder
	case "Grader":
		return BlackboardUserRoleGrader
	case "Guest":
		return BlackboardUserRoleGuest
	default:
		return BlackboardUserRole(roleID)
	}
}

// IsBlackboardCourseInstructor checks if a membership has instructor privileges.
func IsBlackboardCourseInstructor(membership *BlackboardCourseMembership) bool {
	return membership.CourseRoleID == "Instructor"
}

// IsBlackboardCourseStudent checks if a membership is a student.
func IsBlackboardCourseStudent(membership *BlackboardCourseMembership) bool {
	return membership.CourseRoleID == "Student"
}

// IsBlackboardCourseTA checks if a membership is a teaching assistant.
func IsBlackboardCourseTA(membership *BlackboardCourseMembership) bool {
	return membership.CourseRoleID == "TeachingAssistant"
}

// HasBlackboardCourseGradingRights checks if a membership can grade.
func HasBlackboardCourseGradingRights(membership *BlackboardCourseMembership) bool {
	return membership.CourseRoleID == "Instructor" ||
		membership.CourseRoleID == "TeachingAssistant" ||
		membership.CourseRoleID == "Grader"
}

// GetBlackboardStudentID returns the user's student ID.
func GetBlackboardStudentID(user *User) string {
	return user.Extra["student_id"]
}

// GetBlackboardExternalID returns the user's external ID (often SIS ID).
func GetBlackboardExternalID(user *User) string {
	return user.Extra["external_id"]
}
