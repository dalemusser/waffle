// auth/oauth2/banner.go
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

// BannerUserType represents the type of user in Ellucian Banner.
type BannerUserType string

const (
	BannerUserTypeStudent  BannerUserType = "student"
	BannerUserTypeFaculty  BannerUserType = "faculty"
	BannerUserTypeStaff    BannerUserType = "staff"
	BannerUserTypeAdvisor  BannerUserType = "advisor"
	BannerUserTypeAlumni   BannerUserType = "alumni"
	BannerUserTypeApplicant BannerUserType = "applicant"
)

// BannerConfig holds configuration for Ellucian Banner OAuth2 authentication.
type BannerConfig struct {
	// ClientID is the Banner OAuth2 client ID.
	ClientID string

	// ClientSecret is the Banner OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL for OAuth2 authentication.
	// Example: "https://university.app/auth/banner/callback"
	RedirectURL string

	// BaseURL is the Banner API base URL for your institution.
	// Example: "https://banner.university.edu" or "https://university.elluciancloud.com"
	// This is required as each institution has their own Banner instance.
	BaseURL string

	// Tenant is the Ellucian Cloud tenant ID (for cloud deployments).
	// Leave empty for on-premise Banner installations.
	Tenant string

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

// Banner creates a new OAuth2 provider configured for Ellucian Banner authentication.
//
// Ellucian Banner is one of the most widely used enterprise resource planning (ERP)
// and student information systems in higher education. It provides comprehensive
// functionality for student records, financial aid, HR, and finance.
//
// Banner can be deployed on-premise or via Ellucian Cloud (SaaS). The OAuth2
// endpoints differ based on deployment type.
//
// Usage in BuildHandler:
//
//	bannerAuth, err := oauth2.Banner(oauth2.BannerConfig{
//	    ClientID:     appCfg.BannerClientID,
//	    ClientSecret: appCfg.BannerClientSecret,
//	    RedirectURL:  "https://university.app/auth/banner/callback",
//	    BaseURL:      appCfg.BannerBaseURL, // e.g., "https://banner.university.edu"
//	    SessionStore: sessionStore,
//	    StateStore:   stateStore,
//	    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
//	        if oauth2.IsBannerStudent(user) {
//	            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
//	        } else if oauth2.IsBannerFaculty(user) {
//	            http.Redirect(w, r, "/faculty/dashboard", http.StatusTemporaryRedirect)
//	        } else {
//	            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
//	        }
//	    },
//	}, logger)
//
//	r.Get("/auth/banner/login", bannerAuth.LoginHandler())
//	r.Get("/auth/banner/callback", bannerAuth.CallbackHandler())
//	r.Get("/auth/banner/logout", bannerAuth.LogoutHandler())
//
// The User.Extra map contains Banner-specific fields:
//   - "user_type": student, faculty, staff, advisor, alumni, or applicant
//   - "banner_id": Banner ID (SPRIDEN_ID)
//   - "pidm": Banner PIDM (internal identifier)
//   - "institution_id": Institution identifier
//   - "primary_role": Primary role in the institution
//   - "roles": Comma-separated list of all roles
//   - "department": Department (faculty/staff)
//   - "college": College (students)
//   - "major": Major/program (students)
//   - "class_level": Class level - FR, SO, JR, SR, GR (students)
func Banner(cfg BannerConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/banner: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/banner: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/banner: RedirectURL is required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("oauth2/banner: BaseURL is required (e.g., https://banner.university.edu)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/banner: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/banner: StateStore is required")
	}

	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	// Build Banner OAuth2 endpoint
	// Ellucian Cloud uses different endpoints than on-premise
	var authURL, tokenURL string
	if cfg.Tenant != "" {
		// Ellucian Cloud (SaaS)
		authURL = fmt.Sprintf("https://login.elluciancloud.com/%s/oauth2/authorize", cfg.Tenant)
		tokenURL = fmt.Sprintf("https://login.elluciancloud.com/%s/oauth2/token", cfg.Tenant)
	} else {
		// On-premise Banner
		authURL = baseURL + "/api/oauth2/authorize"
		tokenURL = baseURL + "/api/oauth2/token"
	}

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

	// Create a fetcher that captures the base URL
	fetchUserInfo := createBannerUserInfoFetcher(baseURL, cfg.Tenant)

	providerCfg := &Config{
		ProviderName:  "banner",
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

// bannerUserInfo represents user information from Banner.
type bannerUserInfo struct {
	Sub           string   `json:"sub"`             // Subject identifier
	Name          string   `json:"name"`            // Full name
	GivenName     string   `json:"given_name"`      // First name
	FamilyName    string   `json:"family_name"`     // Last name
	MiddleName    string   `json:"middle_name"`     // Middle name
	Email         string   `json:"email"`           // Email address
	EmailVerified bool     `json:"email_verified"`  // Email verification status
	BannerID      string   `json:"banner_id"`       // Banner ID (SPRIDEN_ID)
	PIDM          string   `json:"pidm"`            // Banner PIDM
	InstitutionID string   `json:"institution_id"`  // Institution ID
	Roles         []string `json:"roles"`           // User roles
	PrimaryRole   string   `json:"primary_role"`    // Primary role
	// Student-specific
	College      string `json:"college"`       // College
	Major        string `json:"major"`         // Major/program
	ClassLevel   string `json:"class_level"`   // FR, SO, JR, SR, GR
	EnrollStatus string `json:"enroll_status"` // Enrollment status
	// Faculty/Staff-specific
	Department string `json:"department"` // Department
	Title      string `json:"title"`      // Job title
	EmployeeID string `json:"employee_id"` // Employee ID
}

// createBannerUserInfoFetcher creates a UserInfoFetcher bound to a specific base URL.
func createBannerUserInfoFetcher(baseURL, tenant string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Determine userinfo endpoint
		var userinfoURL string
		if tenant != "" {
			userinfoURL = fmt.Sprintf("https://login.elluciancloud.com/%s/oauth2/userinfo", tenant)
		} else {
			userinfoURL = baseURL + "/api/oauth2/userinfo"
		}

		resp, err := client.Get(userinfoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info bannerUserInfo
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

		// Determine user type from roles
		userType := determineBannerUserType(info.Roles, info.PrimaryRole)

		// Build roles string
		rolesStr := strings.Join(info.Roles, ",")

		// Use Sub as ID, or Banner ID if available
		userID := info.Sub
		if info.BannerID != "" {
			userID = info.BannerID
		}

		return &User{
			ID:            userID,
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
			Name:          name,
			Picture:       "",
			Raw: map[string]any{
				"sub":            info.Sub,
				"name":           info.Name,
				"given_name":     info.GivenName,
				"family_name":    info.FamilyName,
				"middle_name":    info.MiddleName,
				"email":          info.Email,
				"email_verified": info.EmailVerified,
				"banner_id":      info.BannerID,
				"pidm":           info.PIDM,
				"institution_id": info.InstitutionID,
				"roles":          info.Roles,
				"primary_role":   info.PrimaryRole,
				"college":        info.College,
				"major":          info.Major,
				"class_level":    info.ClassLevel,
				"department":     info.Department,
				"title":          info.Title,
			},
			Extra: map[string]string{
				"user_type":      string(userType),
				"banner_id":      info.BannerID,
				"pidm":           info.PIDM,
				"institution_id": info.InstitutionID,
				"primary_role":   info.PrimaryRole,
				"roles":          rolesStr,
				"college":        info.College,
				"major":          info.Major,
				"class_level":    info.ClassLevel,
				"enroll_status":  info.EnrollStatus,
				"department":     info.Department,
				"title":          info.Title,
				"employee_id":    info.EmployeeID,
				"first_name":     info.GivenName,
				"last_name":      info.FamilyName,
			},
		}, nil
	}
}

// determineBannerUserType determines the user type from Banner roles.
func determineBannerUserType(roles []string, primaryRole string) BannerUserType {
	// Check primary role first
	switch strings.ToLower(primaryRole) {
	case "student":
		return BannerUserTypeStudent
	case "faculty":
		return BannerUserTypeFaculty
	case "staff", "employee":
		return BannerUserTypeStaff
	case "advisor":
		return BannerUserTypeAdvisor
	case "alumni":
		return BannerUserTypeAlumni
	case "applicant":
		return BannerUserTypeApplicant
	}

	// Check all roles
	for _, role := range roles {
		roleLower := strings.ToLower(role)
		if strings.Contains(roleLower, "student") {
			return BannerUserTypeStudent
		}
		if strings.Contains(roleLower, "faculty") || strings.Contains(roleLower, "instructor") {
			return BannerUserTypeFaculty
		}
		if strings.Contains(roleLower, "staff") || strings.Contains(roleLower, "employee") {
			return BannerUserTypeStaff
		}
		if strings.Contains(roleLower, "advisor") {
			return BannerUserTypeAdvisor
		}
		if strings.Contains(roleLower, "alumni") {
			return BannerUserTypeAlumni
		}
		if strings.Contains(roleLower, "applicant") {
			return BannerUserTypeApplicant
		}
	}

	return BannerUserTypeStaff // Default
}

// IsBannerStudent returns true if the user is a student.
func IsBannerStudent(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeStudent)
}

// IsBannerFaculty returns true if the user is faculty.
func IsBannerFaculty(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeFaculty)
}

// IsBannerStaff returns true if the user is staff.
func IsBannerStaff(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeStaff)
}

// IsBannerAdvisor returns true if the user is an advisor.
func IsBannerAdvisor(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeAdvisor)
}

// IsBannerAlumni returns true if the user is alumni.
func IsBannerAlumni(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeAlumni)
}

// IsBannerApplicant returns true if the user is an applicant.
func IsBannerApplicant(user *User) bool {
	return user.Extra["user_type"] == string(BannerUserTypeApplicant)
}

// IsBannerEmployee returns true if the user is faculty or staff.
func IsBannerEmployee(user *User) bool {
	return IsBannerFaculty(user) || IsBannerStaff(user)
}

// GetBannerUserType returns the Banner user type from the User.
func GetBannerUserType(user *User) BannerUserType {
	return BannerUserType(user.Extra["user_type"])
}

// GetBannerID returns the Banner ID (SPRIDEN_ID).
func GetBannerID(user *User) string {
	return user.Extra["banner_id"]
}

// GetBannerPIDM returns the Banner PIDM.
func GetBannerPIDM(user *User) string {
	return user.Extra["pidm"]
}

// GetBannerRoles returns the user's roles.
func GetBannerRoles(user *User) []string {
	roles := user.Extra["roles"]
	if roles == "" {
		return nil
	}
	return strings.Split(roles, ",")
}

// HasBannerRole returns true if the user has the specified role.
func HasBannerRole(user *User, role string) bool {
	roles := GetBannerRoles(user)
	roleLower := strings.ToLower(role)
	for _, r := range roles {
		if strings.ToLower(r) == roleLower || strings.Contains(strings.ToLower(r), roleLower) {
			return true
		}
	}
	return false
}

// GetBannerClassLevel returns the student's class level (FR, SO, JR, SR, GR).
func GetBannerClassLevel(user *User) string {
	return user.Extra["class_level"]
}

// GetBannerMajor returns the student's major/program.
func GetBannerMajor(user *User) string {
	return user.Extra["major"]
}

// GetBannerCollege returns the student's college.
func GetBannerCollege(user *User) string {
	return user.Extra["college"]
}

// GetBannerDepartment returns the employee's department.
func GetBannerDepartment(user *User) string {
	return user.Extra["department"]
}

// BannerAPIClient provides methods to call Banner API endpoints.
// Use this after authentication to fetch additional data.
type BannerAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewBannerAPIClient creates a client for calling Banner API.
// Requires a valid OAuth2 access token and the Banner base URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	bannerClient := oauth2.NewBannerAPIClient(user.AccessToken, "https://banner.university.edu")
//	courses, err := bannerClient.GetStudentCourses(r.Context(), oauth2.GetBannerPIDM(user), "202310")
func NewBannerAPIClient(accessToken, baseURL string) *BannerAPIClient {
	return &BannerAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// BannerCourse represents a course from Banner.
type BannerCourse struct {
	CRN            string  `json:"crn"`
	Subject        string  `json:"subject"`
	CourseNumber   string  `json:"course_number"`
	SequenceNumber string  `json:"sequence_number"`
	Title          string  `json:"title"`
	Credits        float64 `json:"credits"`
	GradeMode      string  `json:"grade_mode"`
	TermCode       string  `json:"term_code"`
	Campus         string  `json:"campus"`
	ScheduleType   string  `json:"schedule_type"`
	InstructorName string  `json:"instructor_name"`
}

// GetStudentCourses fetches a student's course registrations for a term.
func (c *BannerAPIClient) GetStudentCourses(ctx context.Context, pidm, termCode string) ([]BannerCourse, error) {
	url := fmt.Sprintf("%s/api/student-registrations/v1/students/%s/registrations?term=%s", c.baseURL, pidm, termCode)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []BannerCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return courses, nil
}

// BannerGrade represents a grade from Banner.
type BannerGrade struct {
	TermCode        string  `json:"term_code"`
	CRN             string  `json:"crn"`
	Subject         string  `json:"subject"`
	CourseNumber    string  `json:"course_number"`
	Title           string  `json:"title"`
	Credits         float64 `json:"credits"`
	Grade           string  `json:"grade"`
	QualityPoints   float64 `json:"quality_points"`
	GradeDate       string  `json:"grade_date"`
	IncompleteDate  string  `json:"incomplete_date"`
}

// GetStudentGrades fetches a student's grades.
func (c *BannerAPIClient) GetStudentGrades(ctx context.Context, pidm string, termCode string) ([]BannerGrade, error) {
	url := fmt.Sprintf("%s/api/student-grades/v1/students/%s/grades", c.baseURL, pidm)
	if termCode != "" {
		url += "?term=" + termCode
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var grades []BannerGrade
	if err := json.NewDecoder(resp.Body).Decode(&grades); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return grades, nil
}

// BannerStudentRecord represents a student's academic record.
type BannerStudentRecord struct {
	PIDM             string  `json:"pidm"`
	BannerID         string  `json:"banner_id"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Email            string  `json:"email"`
	College          string  `json:"college"`
	Major            string  `json:"major"`
	ClassLevel       string  `json:"class_level"`
	CumulativeGPA    float64 `json:"cumulative_gpa"`
	TotalCredits     float64 `json:"total_credits"`
	EnrollmentStatus string  `json:"enrollment_status"`
	AdmitTerm        string  `json:"admit_term"`
	ExpectedGradTerm string  `json:"expected_grad_term"`
}

// GetStudentRecord fetches a student's academic record.
func (c *BannerAPIClient) GetStudentRecord(ctx context.Context, pidm string) (*BannerStudentRecord, error) {
	url := fmt.Sprintf("%s/api/students/v1/students/%s", c.baseURL, pidm)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var record BannerStudentRecord
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode student record: %w", err)
	}

	return &record, nil
}

// BannerSection represents a class section from Banner.
type BannerSection struct {
	CRN              string  `json:"crn"`
	TermCode         string  `json:"term_code"`
	Subject          string  `json:"subject"`
	CourseNumber     string  `json:"course_number"`
	SequenceNumber   string  `json:"sequence_number"`
	Title            string  `json:"title"`
	Credits          float64 `json:"credits"`
	MaxEnrollment    int     `json:"max_enrollment"`
	ActualEnrollment int     `json:"actual_enrollment"`
	SeatsAvailable   int     `json:"seats_available"`
	Campus           string  `json:"campus"`
	ScheduleType     string  `json:"schedule_type"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
}

// GetFacultySections fetches sections taught by a faculty member.
func (c *BannerAPIClient) GetFacultySections(ctx context.Context, pidm, termCode string) ([]BannerSection, error) {
	url := fmt.Sprintf("%s/api/faculty/v1/faculty/%s/sections?term=%s", c.baseURL, pidm, termCode)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch faculty sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sections []BannerSection
	if err := json.NewDecoder(resp.Body).Decode(&sections); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return sections, nil
}

// BannerRoster represents a class roster entry.
type BannerRoster struct {
	PIDM         string `json:"pidm"`
	BannerID     string `json:"banner_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Major        string `json:"major"`
	ClassLevel   string `json:"class_level"`
	RegisterDate string `json:"register_date"`
	Status       string `json:"status"`
}

// GetSectionRoster fetches the roster for a section.
func (c *BannerAPIClient) GetSectionRoster(ctx context.Context, termCode, crn string) ([]BannerRoster, error) {
	url := fmt.Sprintf("%s/api/sections/v1/sections/%s/%s/roster", c.baseURL, termCode, crn)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var roster []BannerRoster
	if err := json.NewDecoder(resp.Body).Decode(&roster); err != nil {
		return nil, fmt.Errorf("failed to decode roster: %w", err)
	}

	return roster, nil
}

// BannerTerm represents an academic term from Banner.
type BannerTerm struct {
	TermCode    string `json:"term_code"`
	Description string `json:"description"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	AcadYear    string `json:"acad_year"`
}

// GetTerms fetches available academic terms.
func (c *BannerAPIClient) GetTerms(ctx context.Context) ([]BannerTerm, error) {
	url := fmt.Sprintf("%s/api/academic-terms/v1/terms", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch terms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var terms []BannerTerm
	if err := json.NewDecoder(resp.Body).Decode(&terms); err != nil {
		return nil, fmt.Errorf("failed to decode terms: %w", err)
	}

	return terms, nil
}

// GetCurrentTerm fetches the current academic term.
func (c *BannerAPIClient) GetCurrentTerm(ctx context.Context) (*BannerTerm, error) {
	url := fmt.Sprintf("%s/api/academic-terms/v1/terms/current", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current term: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var term BannerTerm
	if err := json.NewDecoder(resp.Body).Decode(&term); err != nil {
		return nil, fmt.Errorf("failed to decode term: %w", err)
	}

	return &term, nil
}

// BannerHold represents a student hold from Banner.
type BannerHold struct {
	HoldType    string `json:"hold_type"`
	HoldCode    string `json:"hold_code"`
	Description string `json:"description"`
	FromDate    string `json:"from_date"`
	ToDate      string `json:"to_date"`
	Reason      string `json:"reason"`
	OriginCode  string `json:"origin_code"`
}

// GetStudentHolds fetches a student's holds.
func (c *BannerAPIClient) GetStudentHolds(ctx context.Context, pidm string) ([]BannerHold, error) {
	url := fmt.Sprintf("%s/api/students/v1/students/%s/holds", c.baseURL, pidm)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var holds []BannerHold
	if err := json.NewDecoder(resp.Body).Decode(&holds); err != nil {
		return nil, fmt.Errorf("failed to decode holds: %w", err)
	}

	return holds, nil
}
