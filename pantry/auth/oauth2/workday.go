// auth/oauth2/workday.go
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

// WorkdayUserType represents the type of user in Workday Student.
type WorkdayUserType string

const (
	WorkdayUserTypeStudent    WorkdayUserType = "student"
	WorkdayUserTypeProspect   WorkdayUserType = "prospect"
	WorkdayUserTypeApplicant  WorkdayUserType = "applicant"
	WorkdayUserTypeFaculty    WorkdayUserType = "faculty"
	WorkdayUserTypeStaff      WorkdayUserType = "staff"
	WorkdayUserTypeAdvisor    WorkdayUserType = "advisor"
	WorkdayUserTypeInstructor WorkdayUserType = "instructor"
	WorkdayUserTypeAlumni     WorkdayUserType = "alumni"
)

// WorkdayConfig holds configuration for Workday Student OAuth2 authentication.
type WorkdayConfig struct {
	// ClientID is the Workday OAuth2 client ID.
	ClientID string

	// ClientSecret is the Workday OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL for OAuth2 authentication.
	// Example: "https://university.app/auth/workday/callback"
	RedirectURL string

	// TenantURL is the Workday tenant URL for your institution.
	// Example: "https://wd5-impl-services1.workday.com/ccx/service/tenantname"
	// or "https://university.workday.com"
	TenantURL string

	// TenantName is the Workday tenant name.
	// Example: "university_student"
	TenantName string

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

// Workday creates a new OAuth2 provider configured for Workday Student authentication.
//
// Workday Student is a cloud-based student information system and ERP used by
// colleges and universities. It provides comprehensive student lifecycle management
// including admissions, student records, academic advising, financial aid, and billing.
//
// Workday uses OAuth2 with OIDC for authentication and provides access to
// student, faculty, and staff data through its REST API.
//
// Usage in BuildHandler:
//
//	workdayAuth, err := oauth2.Workday(oauth2.WorkdayConfig{
//	    ClientID:     appCfg.WorkdayClientID,
//	    ClientSecret: appCfg.WorkdayClientSecret,
//	    RedirectURL:  "https://university.app/auth/workday/callback",
//	    TenantURL:    appCfg.WorkdayTenantURL,
//	    TenantName:   appCfg.WorkdayTenantName,
//	    SessionStore: sessionStore,
//	    StateStore:   stateStore,
//	    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
//	        if oauth2.IsWorkdayStudent(user) {
//	            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
//	        } else if oauth2.IsWorkdayFaculty(user) {
//	            http.Redirect(w, r, "/faculty/dashboard", http.StatusTemporaryRedirect)
//	        } else {
//	            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
//	        }
//	    },
//	}, logger)
//
//	r.Get("/auth/workday/login", workdayAuth.LoginHandler())
//	r.Get("/auth/workday/callback", workdayAuth.CallbackHandler())
//	r.Get("/auth/workday/logout", workdayAuth.LogoutHandler())
//
// The User.Extra map contains Workday-specific fields:
//   - "user_type": student, prospect, applicant, faculty, staff, advisor, instructor, alumni
//   - "workday_id": Workday Worker ID or Student ID
//   - "wid": Workday internal ID (WID)
//   - "student_id": Student ID (students only)
//   - "employee_id": Employee ID (faculty/staff only)
//   - "academic_level": Academic level (undergraduate, graduate, etc.)
//   - "academic_period": Current academic period
//   - "program": Academic program (students only)
//   - "department": Department (faculty/staff only)
//   - "primary_position": Primary job position (faculty/staff only)
func Workday(cfg WorkdayConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/workday: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/workday: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/workday: RedirectURL is required")
	}
	if cfg.TenantURL == "" {
		return nil, errors.New("oauth2/workday: TenantURL is required")
	}
	if cfg.TenantName == "" {
		return nil, errors.New("oauth2/workday: TenantName is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/workday: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/workday: StateStore is required")
	}

	tenantURL := strings.TrimSuffix(cfg.TenantURL, "/")

	// Build Workday OAuth2 endpoints
	authURL := fmt.Sprintf("%s/authorize", tenantURL)
	tokenURL := fmt.Sprintf("%s/token", tenantURL)

	endpoint := oauth2.Endpoint{
		AuthURL:  authURL,
		TokenURL: tokenURL,
	}

	// Default scopes
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

	// Create a fetcher that captures the tenant URL and name
	fetchUserInfo := createWorkdayUserInfoFetcher(tenantURL, cfg.TenantName)

	providerCfg := &Config{
		ProviderName:  "workday",
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

// workdayUserInfo represents user information from Workday.
type workdayUserInfo struct {
	Sub              string   `json:"sub"`               // Subject identifier
	Name             string   `json:"name"`              // Full name
	GivenName        string   `json:"given_name"`        // First name
	FamilyName       string   `json:"family_name"`       // Last name
	MiddleName       string   `json:"middle_name"`       // Middle name
	PreferredName    string   `json:"preferred_name"`    // Preferred/chosen name
	Email            string   `json:"email"`             // Email address
	EmailVerified    bool     `json:"email_verified"`    // Email verification status
	WorkdayID        string   `json:"workday_id"`        // Worker ID or Student ID
	WID              string   `json:"wid"`               // Workday internal ID
	Roles            []string `json:"roles"`             // User roles
	UserTypes        []string `json:"user_types"`        // User types
	IsStudent        bool     `json:"is_student"`        // Is a student
	IsEmployee       bool     `json:"is_employee"`       // Is an employee
	IsFaculty        bool     `json:"is_faculty"`        // Is faculty
	// Student-specific
	StudentID        string `json:"student_id"`        // Student ID
	AcademicLevel    string `json:"academic_level"`    // Undergraduate, Graduate, etc.
	AcademicPeriod   string `json:"academic_period"`   // Current academic period
	Program          string `json:"program"`           // Academic program
	ExpectedGradDate string `json:"expected_grad_date"` // Expected graduation date
	// Employee-specific
	EmployeeID      string `json:"employee_id"`      // Employee ID
	Department      string `json:"department"`       // Department
	PrimaryPosition string `json:"primary_position"` // Primary job position
	SupervisorID    string `json:"supervisor_id"`    // Supervisor's worker ID
	Location        string `json:"location"`         // Work location
}

// createWorkdayUserInfoFetcher creates a UserInfoFetcher bound to a specific tenant.
func createWorkdayUserInfoFetcher(tenantURL, tenantName string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Workday userinfo endpoint
		userinfoURL := fmt.Sprintf("%s/userinfo", tenantURL)

		resp, err := client.Get(userinfoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info workdayUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		// Build display name
		name := info.PreferredName
		if name == "" {
			name = info.Name
		}
		if name == "" {
			name = info.GivenName
			if info.FamilyName != "" {
				name += " " + info.FamilyName
			}
		}

		// Determine user type
		userType := determineWorkdayUserType(&info)

		// Build roles string
		rolesStr := strings.Join(info.Roles, ",")

		// Use Sub as ID, or Workday ID if available
		userID := info.Sub
		if info.WorkdayID != "" {
			userID = info.WorkdayID
		}

		return &User{
			ID:            userID,
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
			Name:          name,
			Picture:       "",
			Raw: map[string]any{
				"sub":               info.Sub,
				"name":              info.Name,
				"given_name":        info.GivenName,
				"family_name":       info.FamilyName,
				"middle_name":       info.MiddleName,
				"preferred_name":    info.PreferredName,
				"email":             info.Email,
				"email_verified":    info.EmailVerified,
				"workday_id":        info.WorkdayID,
				"wid":               info.WID,
				"roles":             info.Roles,
				"user_types":        info.UserTypes,
				"is_student":        info.IsStudent,
				"is_employee":       info.IsEmployee,
				"is_faculty":        info.IsFaculty,
				"student_id":        info.StudentID,
				"academic_level":    info.AcademicLevel,
				"academic_period":   info.AcademicPeriod,
				"program":           info.Program,
				"expected_grad_date": info.ExpectedGradDate,
				"employee_id":       info.EmployeeID,
				"department":        info.Department,
				"primary_position":  info.PrimaryPosition,
				"location":          info.Location,
			},
			Extra: map[string]string{
				"user_type":         string(userType),
				"workday_id":        info.WorkdayID,
				"wid":               info.WID,
				"roles":             rolesStr,
				"student_id":        info.StudentID,
				"academic_level":    info.AcademicLevel,
				"academic_period":   info.AcademicPeriod,
				"program":           info.Program,
				"expected_grad_date": info.ExpectedGradDate,
				"employee_id":       info.EmployeeID,
				"department":        info.Department,
				"primary_position":  info.PrimaryPosition,
				"supervisor_id":     info.SupervisorID,
				"location":          info.Location,
				"first_name":        info.GivenName,
				"last_name":         info.FamilyName,
				"preferred_name":    info.PreferredName,
			},
		}, nil
	}
}

// determineWorkdayUserType determines the user type from Workday info.
func determineWorkdayUserType(info *workdayUserInfo) WorkdayUserType {
	// Check explicit flags first
	if info.IsStudent && info.StudentID != "" {
		return WorkdayUserTypeStudent
	}
	if info.IsFaculty {
		return WorkdayUserTypeFaculty
	}
	if info.IsEmployee {
		return WorkdayUserTypeStaff
	}

	// Check user types
	for _, ut := range info.UserTypes {
		utLower := strings.ToLower(ut)
		switch {
		case strings.Contains(utLower, "student"):
			return WorkdayUserTypeStudent
		case strings.Contains(utLower, "prospect"):
			return WorkdayUserTypeProspect
		case strings.Contains(utLower, "applicant"):
			return WorkdayUserTypeApplicant
		case strings.Contains(utLower, "faculty"):
			return WorkdayUserTypeFaculty
		case strings.Contains(utLower, "instructor"):
			return WorkdayUserTypeInstructor
		case strings.Contains(utLower, "advisor"):
			return WorkdayUserTypeAdvisor
		case strings.Contains(utLower, "staff") || strings.Contains(utLower, "employee"):
			return WorkdayUserTypeStaff
		case strings.Contains(utLower, "alumni"):
			return WorkdayUserTypeAlumni
		}
	}

	// Check roles
	for _, role := range info.Roles {
		roleLower := strings.ToLower(role)
		if strings.Contains(roleLower, "student") {
			return WorkdayUserTypeStudent
		}
		if strings.Contains(roleLower, "faculty") || strings.Contains(roleLower, "instructor") {
			return WorkdayUserTypeFaculty
		}
		if strings.Contains(roleLower, "advisor") {
			return WorkdayUserTypeAdvisor
		}
	}

	// Check if has student ID
	if info.StudentID != "" {
		return WorkdayUserTypeStudent
	}

	// Check if has employee ID
	if info.EmployeeID != "" {
		return WorkdayUserTypeStaff
	}

	return WorkdayUserTypeStaff // Default
}

// IsWorkdayStudent returns true if the user is a student.
func IsWorkdayStudent(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeStudent)
}

// IsWorkdayProspect returns true if the user is a prospect.
func IsWorkdayProspect(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeProspect)
}

// IsWorkdayApplicant returns true if the user is an applicant.
func IsWorkdayApplicant(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeApplicant)
}

// IsWorkdayFaculty returns true if the user is faculty.
func IsWorkdayFaculty(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeFaculty)
}

// IsWorkdayStaff returns true if the user is staff.
func IsWorkdayStaff(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeStaff)
}

// IsWorkdayAdvisor returns true if the user is an advisor.
func IsWorkdayAdvisor(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeAdvisor)
}

// IsWorkdayInstructor returns true if the user is an instructor.
func IsWorkdayInstructor(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeInstructor)
}

// IsWorkdayAlumni returns true if the user is alumni.
func IsWorkdayAlumni(user *User) bool {
	return user.Extra["user_type"] == string(WorkdayUserTypeAlumni)
}

// IsWorkdayEmployee returns true if the user is faculty, staff, or instructor.
func IsWorkdayEmployee(user *User) bool {
	userType := user.Extra["user_type"]
	return userType == string(WorkdayUserTypeFaculty) ||
		userType == string(WorkdayUserTypeStaff) ||
		userType == string(WorkdayUserTypeInstructor) ||
		userType == string(WorkdayUserTypeAdvisor)
}

// GetWorkdayUserType returns the Workday user type from the User.
func GetWorkdayUserType(user *User) WorkdayUserType {
	return WorkdayUserType(user.Extra["user_type"])
}

// GetWorkdayID returns the Workday ID (Worker ID or Student ID).
func GetWorkdayID(user *User) string {
	return user.Extra["workday_id"]
}

// GetWorkdayWID returns the Workday internal ID (WID).
func GetWorkdayWID(user *User) string {
	return user.Extra["wid"]
}

// GetWorkdayStudentID returns the student ID.
func GetWorkdayStudentID(user *User) string {
	return user.Extra["student_id"]
}

// GetWorkdayEmployeeID returns the employee ID.
func GetWorkdayEmployeeID(user *User) string {
	return user.Extra["employee_id"]
}

// GetWorkdayAcademicLevel returns the academic level (undergraduate, graduate, etc.).
func GetWorkdayAcademicLevel(user *User) string {
	return user.Extra["academic_level"]
}

// GetWorkdayProgram returns the academic program.
func GetWorkdayProgram(user *User) string {
	return user.Extra["program"]
}

// GetWorkdayDepartment returns the department.
func GetWorkdayDepartment(user *User) string {
	return user.Extra["department"]
}

// GetWorkdayPosition returns the primary position.
func GetWorkdayPosition(user *User) string {
	return user.Extra["primary_position"]
}

// GetWorkdayRoles returns the user's roles.
func GetWorkdayRoles(user *User) []string {
	roles := user.Extra["roles"]
	if roles == "" {
		return nil
	}
	return strings.Split(roles, ",")
}

// WorkdayAPIClient provides methods to call Workday API endpoints.
// Use this after authentication to fetch additional data.
type WorkdayAPIClient struct {
	client     *http.Client
	tenantURL  string
	tenantName string
}

// NewWorkdayAPIClient creates a client for calling Workday API.
// Requires a valid OAuth2 access token and the Workday tenant configuration.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	wdClient := oauth2.NewWorkdayAPIClient(user.AccessToken, tenantURL, tenantName)
//	courses, err := wdClient.GetStudentCourses(r.Context(), oauth2.GetWorkdayStudentID(user))
func NewWorkdayAPIClient(accessToken, tenantURL, tenantName string) *WorkdayAPIClient {
	return &WorkdayAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		tenantURL:  strings.TrimSuffix(tenantURL, "/"),
		tenantName: tenantName,
	}
}

// WorkdayCourse represents a course from Workday Student.
type WorkdayCourse struct {
	WID              string  `json:"wid"`
	CourseID         string  `json:"course_id"`
	CourseTitle      string  `json:"course_title"`
	Subject          string  `json:"subject"`
	CourseNumber     string  `json:"course_number"`
	SectionNumber    string  `json:"section_number"`
	Credits          float64 `json:"credits"`
	AcademicPeriod   string  `json:"academic_period"`
	Status           string  `json:"status"`
	Grade            string  `json:"grade"`
	InstructorName   string  `json:"instructor_name"`
	MeetingPatterns  string  `json:"meeting_patterns"`
	Location         string  `json:"location"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
}

// GetStudentCourses fetches a student's course enrollments.
func (c *WorkdayAPIClient) GetStudentCourses(ctx context.Context, studentID string) ([]WorkdayCourse, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/students/%s/courses", c.tenantURL, c.tenantName, studentID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdayCourse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return result.Data, nil
}

// WorkdayStudentRecord represents a student record from Workday.
type WorkdayStudentRecord struct {
	WID              string  `json:"wid"`
	StudentID        string  `json:"student_id"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	PreferredName    string  `json:"preferred_name"`
	Email            string  `json:"email"`
	AcademicLevel    string  `json:"academic_level"`
	Program          string  `json:"program"`
	Major            string  `json:"major"`
	Minor            string  `json:"minor"`
	AdmitDate        string  `json:"admit_date"`
	ExpectedGradDate string  `json:"expected_grad_date"`
	CumulativeGPA    float64 `json:"cumulative_gpa"`
	TotalCredits     float64 `json:"total_credits"`
	EnrollmentStatus string  `json:"enrollment_status"`
}

// GetStudentRecord fetches a student's academic record.
func (c *WorkdayAPIClient) GetStudentRecord(ctx context.Context, studentID string) (*WorkdayStudentRecord, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/students/%s", c.tenantURL, c.tenantName, studentID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var record WorkdayStudentRecord
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode student record: %w", err)
	}

	return &record, nil
}

// WorkdaySection represents a course section from Workday.
type WorkdaySection struct {
	WID              string  `json:"wid"`
	CourseID         string  `json:"course_id"`
	SectionNumber    string  `json:"section_number"`
	CourseTitle      string  `json:"course_title"`
	Subject          string  `json:"subject"`
	CourseNumber     string  `json:"course_number"`
	Credits          float64 `json:"credits"`
	AcademicPeriod   string  `json:"academic_period"`
	MaxEnrollment    int     `json:"max_enrollment"`
	ActualEnrollment int     `json:"actual_enrollment"`
	WaitlistCount    int     `json:"waitlist_count"`
	MeetingPatterns  string  `json:"meeting_patterns"`
	Location         string  `json:"location"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
}

// GetInstructorSections fetches sections taught by an instructor.
func (c *WorkdayAPIClient) GetInstructorSections(ctx context.Context, instructorID string) ([]WorkdaySection, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/instructors/%s/sections", c.tenantURL, c.tenantName, instructorID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdaySection `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return result.Data, nil
}

// WorkdayRoster represents a class roster entry.
type WorkdayRoster struct {
	WID           string `json:"wid"`
	StudentID     string `json:"student_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	PreferredName string `json:"preferred_name"`
	Email         string `json:"email"`
	AcademicLevel string `json:"academic_level"`
	Program       string `json:"program"`
	Major         string `json:"major"`
	EnrollDate    string `json:"enroll_date"`
	Status        string `json:"status"`
	Grade         string `json:"grade"`
}

// GetSectionRoster fetches the roster for a section.
func (c *WorkdayAPIClient) GetSectionRoster(ctx context.Context, sectionWID string) ([]WorkdayRoster, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/sections/%s/roster", c.tenantURL, c.tenantName, sectionWID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdayRoster `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode roster: %w", err)
	}

	return result.Data, nil
}

// WorkdayAcademicPeriod represents an academic period from Workday.
type WorkdayAcademicPeriod struct {
	WID         string `json:"wid"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // Semester, Quarter, Term
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	AcademicYear string `json:"academic_year"`
	IsCurrent   bool   `json:"is_current"`
}

// GetAcademicPeriods fetches available academic periods.
func (c *WorkdayAPIClient) GetAcademicPeriods(ctx context.Context) ([]WorkdayAcademicPeriod, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/academicPeriods", c.tenantURL, c.tenantName)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch academic periods: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdayAcademicPeriod `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode academic periods: %w", err)
	}

	return result.Data, nil
}

// GetCurrentAcademicPeriod fetches the current academic period.
func (c *WorkdayAPIClient) GetCurrentAcademicPeriod(ctx context.Context) (*WorkdayAcademicPeriod, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/academicPeriods/current", c.tenantURL, c.tenantName)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current period: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var period WorkdayAcademicPeriod
	if err := json.NewDecoder(resp.Body).Decode(&period); err != nil {
		return nil, fmt.Errorf("failed to decode period: %w", err)
	}

	return &period, nil
}

// WorkdayGrade represents a grade from Workday.
type WorkdayGrade struct {
	WID            string  `json:"wid"`
	CourseID       string  `json:"course_id"`
	CourseTitle    string  `json:"course_title"`
	Subject        string  `json:"subject"`
	CourseNumber   string  `json:"course_number"`
	SectionNumber  string  `json:"section_number"`
	Credits        float64 `json:"credits"`
	AcademicPeriod string  `json:"academic_period"`
	Grade          string  `json:"grade"`
	GradePoints    float64 `json:"grade_points"`
	Status         string  `json:"status"`
	GradeDate      string  `json:"grade_date"`
}

// GetStudentGrades fetches a student's grades.
func (c *WorkdayAPIClient) GetStudentGrades(ctx context.Context, studentID string, academicPeriod string) ([]WorkdayGrade, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/students/%s/grades", c.tenantURL, c.tenantName, studentID)
	if academicPeriod != "" {
		url += "?academicPeriod=" + academicPeriod
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdayGrade `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return result.Data, nil
}

// WorkdayAdvisee represents an advisee from Workday.
type WorkdayAdvisee struct {
	WID           string `json:"wid"`
	StudentID     string `json:"student_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	PreferredName string `json:"preferred_name"`
	Email         string `json:"email"`
	AcademicLevel string `json:"academic_level"`
	Program       string `json:"program"`
	Major         string `json:"major"`
	AdvisorType   string `json:"advisor_type"` // Primary, Secondary, etc.
}

// GetAdvisees fetches advisees for an advisor.
func (c *WorkdayAPIClient) GetAdvisees(ctx context.Context, advisorID string) ([]WorkdayAdvisee, error) {
	url := fmt.Sprintf("%s/ccx/api/v1/%s/advisors/%s/advisees", c.tenantURL, c.tenantName, advisorID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch advisees: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []WorkdayAdvisee `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode advisees: %w", err)
	}

	return result.Data, nil
}
