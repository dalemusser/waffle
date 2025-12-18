// auth/oauth2/skyward.go
package oauth2

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

// SkywardUserType represents a user's type in Skyward.
type SkywardUserType string

const (
	// SkywardUserTypeStudent represents a student in Skyward.
	SkywardUserTypeStudent SkywardUserType = "student"

	// SkywardUserTypeStaff represents a staff member (teacher, admin, etc.) in Skyward.
	SkywardUserTypeStaff SkywardUserType = "staff"

	// SkywardUserTypeFamily represents a family member (parent/guardian) in Skyward.
	SkywardUserTypeFamily SkywardUserType = "family"

	// SkywardUserTypeUnknown represents an unknown user type.
	SkywardUserTypeUnknown SkywardUserType = "unknown"
)

// SkywardConfig holds configuration for Skyward OAuth2 authentication.
type SkywardConfig struct {
	// ClientID is the Skyward OAuth2 client ID.
	ClientID string

	// ClientSecret is the Skyward OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Skyward.
	// Example: "https://myapp.com/auth/skyward/callback"
	RedirectURL string

	// DistrictURL is your district's Skyward base URL.
	// Example: "https://skyward.iscorp.com/districtname" or "https://sis.district.k12.state.us"
	// Do NOT include trailing slashes.
	DistrictURL string

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

// Skyward creates a new OAuth2 provider configured for Skyward SIS authentication.
//
// Skyward is a widely used Student Information System (SIS) serving thousands of school
// districts across the United States. It provides comprehensive student data management
// including grades, attendance, scheduling, and family/student/staff portals.
//
// Skyward offers two main product lines:
//   - Skyward SMS 2.0: Legacy system still used by many districts
//   - Skyward Qmlativ: Modern cloud-based platform
//
// Setup in Skyward:
//  1. Contact your Skyward administrator or Skyward support
//  2. Request API access for your application
//  3. Provide your Redirect URI for OAuth2 callback
//  4. Receive your Client ID and Client Secret
//  5. Note your district's Skyward URL
//
// Important: Each school district has their own Skyward instance with its own URL.
//
// Usage in BuildHandler:
//
//	skywardAuth, err := oauth2.Skyward(oauth2.SkywardConfig{
//	    ClientID:     appCfg.SkywardClientID,
//	    ClientSecret: appCfg.SkywardClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/skyward/callback",
//	    DistrictURL:  appCfg.SkywardDistrictURL, // e.g., "https://skyward.iscorp.com/districtname"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/skyward/login", skywardAuth.LoginHandler())
//	r.Get("/auth/skyward/callback", skywardAuth.CallbackHandler())
//	r.Get("/auth/skyward/logout", skywardAuth.LogoutHandler())
//
// The User.Extra map contains Skyward-specific fields:
//   - "skyward_id": Skyward user ID
//   - "name_id": Name ID (unique identifier)
//   - "user_type": "student", "staff", or "family"
//   - "district_id": District identifier
//   - "entity_id": Entity (building/school) ID
//   - "entity_name": Entity (building/school) name
//   - "first_name": User's first name
//   - "last_name": User's last name
//   - "middle_name": User's middle name
//   - "student_id": Student ID number (students only)
//   - "grade_level": Grade level (students only)
//   - "graduation_year": Expected graduation year (students only)
//   - "employee_id": Employee ID (staff only)
//   - "position": Job position/title (staff only)
//   - "department": Department (staff only)
//   - "student_name_ids": Comma-separated student name IDs (family only)
func Skyward(cfg SkywardConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/skyward: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/skyward: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/skyward: RedirectURL is required")
	}
	if cfg.DistrictURL == "" {
		return nil, errors.New("oauth2/skyward: DistrictURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/skyward: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/skyward: StateStore is required")
	}

	// Clean up district URL - remove trailing slashes
	districtURL := strings.TrimSuffix(cfg.DistrictURL, "/")

	// Build OAuth2 endpoint URLs
	// Skyward uses /api/oauth2/ for OAuth2 endpoints
	baseOAuthURL := districtURL + "/api/oauth2"

	endpoint := oauth2.Endpoint{
		AuthURL:  baseOAuthURL + "/authorize",
		TokenURL: baseOAuthURL + "/token",
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

	// Create a fetcher that uses the Skyward API
	fetchUserInfo := createSkywardUserInfoFetcher(districtURL)

	providerCfg := &Config{
		ProviderName:  "skyward",
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

// skywardUserInfo represents the response from Skyward user info endpoint.
type skywardUserInfo struct {
	SkywardID    string `json:"skywardId"`
	NameID       int64  `json:"nameId"`
	Username     string `json:"username"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	MiddleName   string `json:"middleName"`
	Email        string `json:"email"`
	UserType     string `json:"userType"` // student, staff, family
	DistrictID   int64  `json:"districtId"`
	EntityID     int64  `json:"entityId"`
	EntityName   string `json:"entityName"`

	// Student-specific fields
	StudentID      string `json:"studentId,omitempty"`
	GradeLevel     string `json:"gradeLevel,omitempty"`
	GraduationYear int    `json:"graduationYear,omitempty"`
	Homeroom       string `json:"homeroom,omitempty"`

	// Staff-specific fields
	EmployeeID string `json:"employeeId,omitempty"`
	Position   string `json:"position,omitempty"`
	Department string `json:"department,omitempty"`

	// Family-specific fields
	StudentNameIDs []int64 `json:"studentNameIds,omitempty"`
}

// createSkywardUserInfoFetcher creates a UserInfoFetcher for Skyward.
func createSkywardUserInfoFetcher(districtURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from Skyward API
		resp, err := client.Get(districtURL + "/api/v1/user/me")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info skywardUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		// Build display name
		name := strings.TrimSpace(info.FirstName + " " + info.LastName)
		if name == "" {
			name = info.Username
		}

		// Build extra fields
		extra := map[string]string{
			"skyward_id":  info.SkywardID,
			"name_id":     fmt.Sprintf("%d", info.NameID),
			"user_type":   info.UserType,
			"district_id": fmt.Sprintf("%d", info.DistrictID),
			"entity_id":   fmt.Sprintf("%d", info.EntityID),
			"entity_name": info.EntityName,
			"first_name":  info.FirstName,
			"last_name":   info.LastName,
			"middle_name": info.MiddleName,
		}

		// Add user-type specific fields
		switch info.UserType {
		case "student":
			extra["student_id"] = info.StudentID
			extra["grade_level"] = info.GradeLevel
			if info.GraduationYear > 0 {
				extra["graduation_year"] = fmt.Sprintf("%d", info.GraduationYear)
			}
			extra["homeroom"] = info.Homeroom
		case "staff":
			extra["employee_id"] = info.EmployeeID
			extra["position"] = info.Position
			extra["department"] = info.Department
		case "family":
			if len(info.StudentNameIDs) > 0 {
				ids := make([]string, len(info.StudentNameIDs))
				for i, id := range info.StudentNameIDs {
					ids[i] = fmt.Sprintf("%d", id)
				}
				extra["student_name_ids"] = strings.Join(ids, ",")
			}
		}

		return &User{
			ID:            fmt.Sprintf("%d", info.NameID),
			Email:         info.Email,
			EmailVerified: info.Email != "",
			Name:          name,
			Picture:       "", // Skyward doesn't typically provide profile pictures via API
			Raw: map[string]any{
				"skywardId":      info.SkywardID,
				"nameId":         info.NameID,
				"username":       info.Username,
				"firstName":      info.FirstName,
				"lastName":       info.LastName,
				"middleName":     info.MiddleName,
				"email":          info.Email,
				"userType":       info.UserType,
				"districtId":     info.DistrictID,
				"entityId":       info.EntityID,
				"entityName":     info.EntityName,
				"studentId":      info.StudentID,
				"gradeLevel":     info.GradeLevel,
				"graduationYear": info.GraduationYear,
				"homeroom":       info.Homeroom,
				"employeeId":     info.EmployeeID,
				"position":       info.Position,
				"department":     info.Department,
				"studentNameIds": info.StudentNameIDs,
			},
			Extra: extra,
		}, nil
	}
}

// SkywardAPIClient provides methods to call Skyward API endpoints.
// Use this after authentication for additional Skyward API operations.
type SkywardAPIClient struct {
	client      *http.Client
	districtURL string
}

// NewSkywardAPIClient creates a client for calling Skyward API.
// Requires a valid OAuth2 access token and the district URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	skywardClient := oauth2.NewSkywardAPIClient(user.AccessToken, "https://skyward.iscorp.com/districtname")
//	grades, err := skywardClient.GetStudentGrades(r.Context(), nameID)
func NewSkywardAPIClient(accessToken, districtURL string) *SkywardAPIClient {
	// Clean up district URL
	districtURL = strings.TrimSuffix(districtURL, "/")

	return &SkywardAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		districtURL: districtURL,
	}
}

// SkywardStudent represents a student record in Skyward.
type SkywardStudent struct {
	NameID         int64  `json:"nameId"`
	StudentID      string `json:"studentId"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	MiddleName     string `json:"middleName"`
	GradeLevel     string `json:"gradeLevel"`
	GraduationYear int    `json:"graduationYear"`
	EntityID       int64  `json:"entityId"`
	EntityName     string `json:"entityName"`
	Homeroom       string `json:"homeroom"`
	Counselor      string `json:"counselor"`
	EnrollmentDate string `json:"enrollmentDate"`
	Active         bool   `json:"active"`
}

// SkywardCourse represents a course/class in Skyward.
type SkywardCourse struct {
	SectionID      int64  `json:"sectionId"`
	CourseID       string `json:"courseId"`
	CourseName     string `json:"courseName"`
	CourseNumber   string `json:"courseNumber"`
	SectionNumber  string `json:"sectionNumber"`
	Period         string `json:"period"`
	Room           string `json:"room"`
	TeacherNameID  int64  `json:"teacherNameId"`
	TeacherName    string `json:"teacherName"`
	TermID         int64  `json:"termId"`
	TermName       string `json:"termName"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	Credits        float64 `json:"credits"`
	Department     string `json:"department"`
}

// SkywardGrade represents a grade entry in Skyward.
type SkywardGrade struct {
	SectionID       int64   `json:"sectionId"`
	CourseName      string  `json:"courseName"`
	AssignmentID    int64   `json:"assignmentId"`
	AssignmentName  string  `json:"assignmentName"`
	Category        string  `json:"category"`
	Score           float64 `json:"score"`
	PointsPossible  float64 `json:"pointsPossible"`
	Percent         float64 `json:"percent"`
	LetterGrade     string  `json:"letterGrade"`
	DateDue         string  `json:"dateDue"`
	DateAssigned    string  `json:"dateAssigned"`
	Comments        string  `json:"comments"`
	Missing         bool    `json:"missing"`
	Late            bool    `json:"late"`
	Exempt          bool    `json:"exempt"`
	GradingPeriod   string  `json:"gradingPeriod"`
}

// SkywardGradeSummary represents a course grade summary in Skyward.
type SkywardGradeSummary struct {
	SectionID     int64   `json:"sectionId"`
	CourseName    string  `json:"courseName"`
	CourseNumber  string  `json:"courseNumber"`
	TeacherName   string  `json:"teacherName"`
	Period        string  `json:"period"`
	CurrentGrade  float64 `json:"currentGrade"`
	LetterGrade   string  `json:"letterGrade"`
	GradingPeriod string  `json:"gradingPeriod"`
	Credits       float64 `json:"credits"`
	GradePoints   float64 `json:"gradePoints"`
}

// SkywardAttendance represents an attendance record in Skyward.
type SkywardAttendance struct {
	AttendanceID  int64  `json:"attendanceId"`
	Date          string `json:"date"`
	Period        string `json:"period"`
	Status        string `json:"status"` // Present, Absent, Tardy, etc.
	StatusCode    string `json:"statusCode"`
	Reason        string `json:"reason"`
	Comments      string `json:"comments"`
	SectionID     int64  `json:"sectionId,omitempty"`
	CourseName    string `json:"courseName,omitempty"`
	Excused       bool   `json:"excused"`
}

// SkywardScheduleDay represents a day in the student's schedule.
type SkywardScheduleDay struct {
	DayName  string          `json:"dayName"`
	DayType  string          `json:"dayType"`
	Periods  []SkywardPeriod `json:"periods"`
}

// SkywardPeriod represents a period in the schedule.
type SkywardPeriod struct {
	Period     string `json:"period"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	CourseID   string `json:"courseId"`
	CourseName string `json:"courseName"`
	Room       string `json:"room"`
	TeacherName string `json:"teacherName"`
}

// SkywardFee represents a fee/balance in Skyward.
type SkywardFee struct {
	FeeID       int64   `json:"feeId"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	DateCharged string  `json:"dateCharged"`
	DueDate     string  `json:"dueDate"`
	Category    string  `json:"category"`
	Paid        bool    `json:"paid"`
	Balance     float64 `json:"balance"`
}

// GetStudentSchedule fetches a student's class schedule.
func (c *SkywardAPIClient) GetStudentSchedule(ctx context.Context, nameID int64) ([]SkywardCourse, error) {
	url := fmt.Sprintf("%s/api/v1/students/%d/schedule", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []SkywardCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode schedule: %w", err)
	}

	return courses, nil
}

// GetStudentGrades fetches a student's detailed assignment grades.
func (c *SkywardAPIClient) GetStudentGrades(ctx context.Context, nameID int64) ([]SkywardGrade, error) {
	url := fmt.Sprintf("%s/api/v1/students/%d/grades", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var grades []SkywardGrade
	if err := json.NewDecoder(resp.Body).Decode(&grades); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return grades, nil
}

// GetStudentGradeSummary fetches a student's grade summary by course.
func (c *SkywardAPIClient) GetStudentGradeSummary(ctx context.Context, nameID int64) ([]SkywardGradeSummary, error) {
	url := fmt.Sprintf("%s/api/v1/students/%d/grades/summary", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grade summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var summary []SkywardGradeSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode grade summary: %w", err)
	}

	return summary, nil
}

// GetStudentAttendance fetches a student's attendance records.
func (c *SkywardAPIClient) GetStudentAttendance(ctx context.Context, nameID int64) ([]SkywardAttendance, error) {
	url := fmt.Sprintf("%s/api/v1/students/%d/attendance", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attendance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var attendance []SkywardAttendance
	if err := json.NewDecoder(resp.Body).Decode(&attendance); err != nil {
		return nil, fmt.Errorf("failed to decode attendance: %w", err)
	}

	return attendance, nil
}

// GetFamilyStudents fetches the students associated with a family account.
func (c *SkywardAPIClient) GetFamilyStudents(ctx context.Context, familyNameID int64) ([]SkywardStudent, error) {
	url := fmt.Sprintf("%s/api/v1/family/%d/students", c.districtURL, familyNameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var students []SkywardStudent
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return students, nil
}

// GetStudentFees fetches a student's fees and balances.
func (c *SkywardAPIClient) GetStudentFees(ctx context.Context, nameID int64) ([]SkywardFee, error) {
	url := fmt.Sprintf("%s/api/v1/students/%d/fees", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fees: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var fees []SkywardFee
	if err := json.NewDecoder(resp.Body).Decode(&fees); err != nil {
		return nil, fmt.Errorf("failed to decode fees: %w", err)
	}

	return fees, nil
}

// SkywardStaffSection represents a section taught by a staff member.
type SkywardStaffSection struct {
	SectionID       int64  `json:"sectionId"`
	CourseID        string `json:"courseId"`
	CourseName      string `json:"courseName"`
	CourseNumber    string `json:"courseNumber"`
	SectionNumber   string `json:"sectionNumber"`
	Period          string `json:"period"`
	Room            string `json:"room"`
	EnrollmentCount int    `json:"enrollmentCount"`
	TermID          int64  `json:"termId"`
	TermName        string `json:"termName"`
	Department      string `json:"department"`
}

// GetStaffSections fetches the sections/classes taught by a staff member.
func (c *SkywardAPIClient) GetStaffSections(ctx context.Context, nameID int64) ([]SkywardStaffSection, error) {
	url := fmt.Sprintf("%s/api/v1/staff/%d/sections", c.districtURL, nameID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sections []SkywardStaffSection
	if err := json.NewDecoder(resp.Body).Decode(&sections); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return sections, nil
}

// SkywardSectionRoster represents a student in a section roster.
type SkywardSectionRoster struct {
	NameID        int64  `json:"nameId"`
	StudentID     string `json:"studentId"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	GradeLevel    string `json:"gradeLevel"`
	Email         string `json:"email"`
	CurrentGrade  float64 `json:"currentGrade"`
	LetterGrade   string `json:"letterGrade"`
	EnrollmentDate string `json:"enrollmentDate"`
	Active        bool   `json:"active"`
}

// GetSectionRoster fetches the roster for a specific section.
func (c *SkywardAPIClient) GetSectionRoster(ctx context.Context, sectionID int64) ([]SkywardSectionRoster, error) {
	url := fmt.Sprintf("%s/api/v1/sections/%d/roster", c.districtURL, sectionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var roster []SkywardSectionRoster
	if err := json.NewDecoder(resp.Body).Decode(&roster); err != nil {
		return nil, fmt.Errorf("failed to decode roster: %w", err)
	}

	return roster, nil
}

// SkywardEntity represents a school/building in Skyward.
type SkywardEntity struct {
	EntityID     int64  `json:"entityId"`
	Name         string `json:"name"`
	Number       string `json:"number"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Zip          string `json:"zip"`
	Phone        string `json:"phone"`
	Principal    string `json:"principal"`
	GradeLow     string `json:"gradeLow"`
	GradeHigh    string `json:"gradeHigh"`
	DistrictID   int64  `json:"districtId"`
	DistrictName string `json:"districtName"`
}

// GetEntity fetches information about a specific entity (school/building).
func (c *SkywardAPIClient) GetEntity(ctx context.Context, entityID int64) (*SkywardEntity, error) {
	url := fmt.Sprintf("%s/api/v1/entities/%d", c.districtURL, entityID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var entity SkywardEntity
	if err := json.NewDecoder(resp.Body).Decode(&entity); err != nil {
		return nil, fmt.Errorf("failed to decode entity: %w", err)
	}

	return &entity, nil
}

// SkywardGradingPeriod represents a grading period in Skyward.
type SkywardGradingPeriod struct {
	PeriodID   int64  `json:"periodId"`
	Name       string `json:"name"`
	ShortName  string `json:"shortName"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	IsCurrent  bool   `json:"isCurrent"`
	SchoolYear string `json:"schoolYear"`
}

// GetGradingPeriods fetches the grading periods for an entity.
func (c *SkywardAPIClient) GetGradingPeriods(ctx context.Context, entityID int64) ([]SkywardGradingPeriod, error) {
	url := fmt.Sprintf("%s/api/v1/entities/%d/grading-periods", c.districtURL, entityID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grading periods: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var periods []SkywardGradingPeriod
	if err := json.NewDecoder(resp.Body).Decode(&periods); err != nil {
		return nil, fmt.Errorf("failed to decode grading periods: %w", err)
	}

	return periods, nil
}

// IsSkywardStudent checks if the user is a student.
func IsSkywardStudent(user *User) bool {
	return user.Extra["user_type"] == "student"
}

// IsSkywardStaff checks if the user is a staff member.
func IsSkywardStaff(user *User) bool {
	return user.Extra["user_type"] == "staff"
}

// IsSkywardFamily checks if the user is a family member (parent/guardian).
func IsSkywardFamily(user *User) bool {
	return user.Extra["user_type"] == "family"
}

// GetSkywardUserType returns the user's type as an enum.
func GetSkywardUserType(user *User) SkywardUserType {
	switch user.Extra["user_type"] {
	case "student":
		return SkywardUserTypeStudent
	case "staff":
		return SkywardUserTypeStaff
	case "family":
		return SkywardUserTypeFamily
	default:
		return SkywardUserTypeUnknown
	}
}

// GetSkywardNameID returns the user's name ID.
func GetSkywardNameID(user *User) string {
	return user.Extra["name_id"]
}

// GetSkywardEntityID returns the user's current entity (school/building) ID.
func GetSkywardEntityID(user *User) string {
	return user.Extra["entity_id"]
}

// GetSkywardDistrictID returns the user's district ID.
func GetSkywardDistrictID(user *User) string {
	return user.Extra["district_id"]
}

// GetSkywardStudentID returns the student ID number (students only).
func GetSkywardStudentID(user *User) string {
	return user.Extra["student_id"]
}

// GetSkywardGradeLevel returns the grade level (students only).
func GetSkywardGradeLevel(user *User) string {
	return user.Extra["grade_level"]
}

// GetSkywardEmployeeID returns the employee ID (staff only).
func GetSkywardEmployeeID(user *User) string {
	return user.Extra["employee_id"]
}

// GetSkywardStudentNameIDs returns the name IDs of students linked to a family account.
func GetSkywardStudentNameIDs(user *User) []string {
	ids := user.Extra["student_name_ids"]
	if ids == "" {
		return nil
	}
	return strings.Split(ids, ",")
}
