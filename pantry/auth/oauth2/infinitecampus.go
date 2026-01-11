// auth/oauth2/infinitecampus.go
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

// InfiniteCampusUserType represents a user's type in Infinite Campus.
type InfiniteCampusUserType string

const (
	// InfiniteCampusUserTypeStudent represents a student in Infinite Campus.
	InfiniteCampusUserTypeStudent InfiniteCampusUserType = "student"

	// InfiniteCampusUserTypeStaff represents a staff member (teacher, admin, etc.) in Infinite Campus.
	InfiniteCampusUserTypeStaff InfiniteCampusUserType = "staff"

	// InfiniteCampusUserTypeParent represents a parent/guardian in Infinite Campus.
	InfiniteCampusUserTypeParent InfiniteCampusUserType = "parent"

	// InfiniteCampusUserTypeUnknown represents an unknown user type.
	InfiniteCampusUserTypeUnknown InfiniteCampusUserType = "unknown"
)

// InfiniteCampusConfig holds configuration for Infinite Campus OAuth2 authentication.
type InfiniteCampusConfig struct {
	// ClientID is the Infinite Campus OAuth2 client ID.
	ClientID string

	// ClientSecret is the Infinite Campus OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Infinite Campus.
	// Example: "https://myapp.com/auth/infinitecampus/callback"
	RedirectURL string

	// DistrictURL is your district's Infinite Campus base URL.
	// Example: "https://campus.district.k12.state.us" or "https://districtname.infinitecampus.org"
	// Do NOT include /campus or trailing slashes.
	DistrictURL string

	// AppName is your registered application name in Infinite Campus.
	// This is used in the OAuth2 authorization URL.
	AppName string

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

// InfiniteCampus creates a new OAuth2 provider configured for Infinite Campus authentication.
//
// Infinite Campus is one of the most widely used Student Information Systems (SIS) in K-12
// education in the United States. It provides comprehensive student data management including
// grades, attendance, scheduling, and more.
//
// Setup in Infinite Campus:
//  1. Log in to Infinite Campus as a System Administrator
//  2. Go to System Administration → Portal → Portal Options → OAuth Applications
//  3. Click Add and configure your application
//  4. Set the Redirect URI to your callback URL
//  5. Note your Client ID and Client Secret
//  6. Configure the appropriate scopes for your application
//
// Important: Each school district has their own Infinite Campus instance with its own URL.
//
// Usage in BuildHandler:
//
//	icAuth, err := oauth2.InfiniteCampus(oauth2.InfiniteCampusConfig{
//	    ClientID:     appCfg.InfiniteCampusClientID,
//	    ClientSecret: appCfg.InfiniteCampusClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/infinitecampus/callback",
//	    DistrictURL:  appCfg.InfiniteCampusDistrictURL, // e.g., "https://campus.district.k12.state.us"
//	    AppName:      appCfg.InfiniteCampusAppName,
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/infinitecampus/login", icAuth.LoginHandler())
//	r.Get("/auth/infinitecampus/callback", icAuth.CallbackHandler())
//	r.Get("/auth/infinitecampus/logout", icAuth.LogoutHandler())
//
// The User.Extra map contains Infinite Campus-specific fields:
//   - "ic_person_id": Infinite Campus person ID
//   - "ic_user_id": Infinite Campus user ID
//   - "user_type": "student", "staff", or "parent"
//   - "district_id": District identifier
//   - "calendar_id": Current calendar ID
//   - "school_id": Current school ID
//   - "school_name": Current school name
//   - "first_name": User's first name
//   - "last_name": User's last name
//   - "middle_name": User's middle name
//   - "student_number": Student number (students only)
//   - "grade_level": Grade level (students only)
//   - "graduation_year": Expected graduation year (students only)
//   - "staff_number": Staff number (staff only)
//   - "title": Job title (staff only)
func InfiniteCampus(cfg InfiniteCampusConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/infinitecampus: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/infinitecampus: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/infinitecampus: RedirectURL is required")
	}
	if cfg.DistrictURL == "" {
		return nil, errors.New("oauth2/infinitecampus: DistrictURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/infinitecampus: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/infinitecampus: StateStore is required")
	}

	// Clean up district URL - remove trailing slashes and /campus if present
	districtURL := cfg.DistrictURL
	districtURL = strings.TrimSuffix(districtURL, "/")
	districtURL = strings.TrimSuffix(districtURL, "/campus")
	districtURL = strings.TrimSuffix(districtURL, "/")

	// Build OAuth2 endpoint URLs
	// Infinite Campus uses /campus/oauth2/ for OAuth2 endpoints
	baseOAuthURL := districtURL + "/campus/api/oauth2"

	endpoint := oauth2.Endpoint{
		AuthURL:  baseOAuthURL + "/authorize",
		TokenURL: baseOAuthURL + "/token",
	}

	// Default scopes
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"user",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	// Create a fetcher that uses the Infinite Campus API
	fetchUserInfo := createInfiniteCampusUserInfoFetcher(districtURL)

	providerCfg := &Config{
		ProviderName:  "infinitecampus",
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

// infiniteCampusUserInfo represents the response from Infinite Campus user info endpoint.
type infiniteCampusUserInfo struct {
	PersonID       int64  `json:"personID"`
	UserID         int64  `json:"userID"`
	Username       string `json:"username"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	MiddleName     string `json:"middleName"`
	Email          string `json:"email"`
	UserType       string `json:"userType"` // student, staff, parent
	DistrictID     int64  `json:"districtID"`
	CalendarID     int64  `json:"calendarID"`
	SchoolID       int64  `json:"schoolID"`
	SchoolName     string `json:"schoolName"`

	// Student-specific fields
	StudentNumber  string `json:"studentNumber,omitempty"`
	GradeLevel     string `json:"gradeLevel,omitempty"`
	GraduationYear int    `json:"graduationYear,omitempty"`

	// Staff-specific fields
	StaffNumber    string `json:"staffNumber,omitempty"`
	Title          string `json:"title,omitempty"`
	Department     string `json:"department,omitempty"`

	// Parent-specific fields
	StudentPersonIDs []int64 `json:"studentPersonIDs,omitempty"`
}

// createInfiniteCampusUserInfoFetcher creates a UserInfoFetcher for Infinite Campus.
func createInfiniteCampusUserInfoFetcher(districtURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from Infinite Campus API
		resp, err := client.Get(districtURL + "/campus/api/portal/user")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info infiniteCampusUserInfo
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
			"ic_person_id": fmt.Sprintf("%d", info.PersonID),
			"ic_user_id":   fmt.Sprintf("%d", info.UserID),
			"user_type":    info.UserType,
			"district_id":  fmt.Sprintf("%d", info.DistrictID),
			"calendar_id":  fmt.Sprintf("%d", info.CalendarID),
			"school_id":    fmt.Sprintf("%d", info.SchoolID),
			"school_name":  info.SchoolName,
			"first_name":   info.FirstName,
			"last_name":    info.LastName,
			"middle_name":  info.MiddleName,
		}

		// Add user-type specific fields
		switch info.UserType {
		case "student":
			extra["student_number"] = info.StudentNumber
			extra["grade_level"] = info.GradeLevel
			if info.GraduationYear > 0 {
				extra["graduation_year"] = fmt.Sprintf("%d", info.GraduationYear)
			}
		case "staff":
			extra["staff_number"] = info.StaffNumber
			extra["title"] = info.Title
			extra["department"] = info.Department
		case "parent":
			if len(info.StudentPersonIDs) > 0 {
				ids := make([]string, len(info.StudentPersonIDs))
				for i, id := range info.StudentPersonIDs {
					ids[i] = fmt.Sprintf("%d", id)
				}
				extra["student_person_ids"] = strings.Join(ids, ",")
			}
		}

		return &User{
			ID:            fmt.Sprintf("%d", info.PersonID),
			Email:         info.Email,
			EmailVerified: info.Email != "",
			Name:          name,
			Picture:       "", // Infinite Campus doesn't typically provide profile pictures via API
			Raw: map[string]any{
				"personID":       info.PersonID,
				"userID":         info.UserID,
				"username":       info.Username,
				"firstName":      info.FirstName,
				"lastName":       info.LastName,
				"middleName":     info.MiddleName,
				"email":          info.Email,
				"userType":       info.UserType,
				"districtID":     info.DistrictID,
				"calendarID":     info.CalendarID,
				"schoolID":       info.SchoolID,
				"schoolName":     info.SchoolName,
				"studentNumber":  info.StudentNumber,
				"gradeLevel":     info.GradeLevel,
				"graduationYear": info.GraduationYear,
				"staffNumber":    info.StaffNumber,
				"title":          info.Title,
				"department":     info.Department,
				"studentPersonIDs": info.StudentPersonIDs,
			},
			Extra: extra,
		}, nil
	}
}

// InfiniteCampusAPIClient provides methods to call Infinite Campus API endpoints.
// Use this after authentication for additional Infinite Campus API operations.
type InfiniteCampusAPIClient struct {
	client      *http.Client
	districtURL string
}

// NewInfiniteCampusAPIClient creates a client for calling Infinite Campus API.
// Requires a valid OAuth2 access token and the district URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	icClient := oauth2.NewInfiniteCampusAPIClient(user.AccessToken, "https://campus.district.k12.state.us")
//	schedule, err := icClient.GetStudentSchedule(r.Context(), personID)
func NewInfiniteCampusAPIClient(accessToken, districtURL string) *InfiniteCampusAPIClient {
	// Clean up district URL
	districtURL = strings.TrimSuffix(districtURL, "/")
	districtURL = strings.TrimSuffix(districtURL, "/campus")
	districtURL = strings.TrimSuffix(districtURL, "/")

	return &InfiniteCampusAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		districtURL: districtURL,
	}
}

// InfiniteCampusStudent represents a student record in Infinite Campus.
type InfiniteCampusStudent struct {
	PersonID       int64  `json:"personID"`
	StudentNumber  string `json:"studentNumber"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	MiddleName     string `json:"middleName"`
	GradeLevel     string `json:"gradeLevel"`
	GraduationYear int    `json:"graduationYear"`
	SchoolID       int64  `json:"schoolID"`
	SchoolName     string `json:"schoolName"`
	CalendarID     int64  `json:"calendarID"`
	EnrollmentID   int64  `json:"enrollmentID"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	Active         bool   `json:"active"`
}

// InfiniteCampusCourse represents a course/section in Infinite Campus.
type InfiniteCampusCourse struct {
	SectionID      int64  `json:"sectionID"`
	CourseID       int64  `json:"courseID"`
	CourseNumber   string `json:"courseNumber"`
	CourseName     string `json:"courseName"`
	SectionNumber  string `json:"sectionNumber"`
	PeriodID       int64  `json:"periodID"`
	PeriodName     string `json:"periodName"`
	PeriodSequence int    `json:"periodSequence"`
	RoomID         int64  `json:"roomID"`
	RoomName       string `json:"roomName"`
	TeacherID      int64  `json:"teacherID"`
	TeacherName    string `json:"teacherName"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	TermID         int64  `json:"termID"`
	TermName       string `json:"termName"`
}

// InfiniteCampusGrade represents a grade entry in Infinite Campus.
type InfiniteCampusGrade struct {
	SectionID     int64   `json:"sectionID"`
	CourseName    string  `json:"courseName"`
	TaskID        int64   `json:"taskID"`
	TaskName      string  `json:"taskName"`
	Score         float64 `json:"score"`
	Percent       float64 `json:"percent"`
	LetterGrade   string  `json:"letterGrade"`
	GradePoints   float64 `json:"gradePoints"`
	Comments      string  `json:"comments"`
	DateDue       string  `json:"dateDue"`
	DateAssigned  string  `json:"dateAssigned"`
	CategoryID    int64   `json:"categoryID"`
	CategoryName  string  `json:"categoryName"`
	Missing       bool    `json:"missing"`
	Late          bool    `json:"late"`
	Exempt        bool    `json:"exempt"`
	Incomplete    bool    `json:"incomplete"`
}

// InfiniteCampusAttendance represents an attendance record in Infinite Campus.
type InfiniteCampusAttendance struct {
	AttendanceID  int64  `json:"attendanceID"`
	Date          string `json:"date"`
	PeriodID      int64  `json:"periodID"`
	PeriodName    string `json:"periodName"`
	Status        string `json:"status"` // Present, Absent, Tardy, etc.
	StatusCode    string `json:"statusCode"`
	Excuse        string `json:"excuse"`
	Comments      string `json:"comments"`
	SectionID     int64  `json:"sectionID,omitempty"`
	CourseName    string `json:"courseName,omitempty"`
}

// InfiniteCampusScheduleResponse represents the response for schedule requests.
type InfiniteCampusScheduleResponse struct {
	Courses []InfiniteCampusCourse `json:"courses"`
}

// InfiniteCampusGradesResponse represents the response for grades requests.
type InfiniteCampusGradesResponse struct {
	Grades []InfiniteCampusGrade `json:"grades"`
}

// InfiniteCampusAttendanceResponse represents the response for attendance requests.
type InfiniteCampusAttendanceResponse struct {
	Attendance []InfiniteCampusAttendance `json:"attendance"`
}

// GetStudentSchedule fetches a student's class schedule.
func (c *InfiniteCampusAPIClient) GetStudentSchedule(ctx context.Context, personID int64) ([]InfiniteCampusCourse, error) {
	url := fmt.Sprintf("%s/campus/api/portal/students/%d/schedule", c.districtURL, personID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response InfiniteCampusScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode schedule: %w", err)
	}

	return response.Courses, nil
}

// GetStudentGrades fetches a student's grades.
func (c *InfiniteCampusAPIClient) GetStudentGrades(ctx context.Context, personID int64) ([]InfiniteCampusGrade, error) {
	url := fmt.Sprintf("%s/campus/api/portal/students/%d/grades", c.districtURL, personID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response InfiniteCampusGradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return response.Grades, nil
}

// GetStudentAttendance fetches a student's attendance records.
func (c *InfiniteCampusAPIClient) GetStudentAttendance(ctx context.Context, personID int64) ([]InfiniteCampusAttendance, error) {
	url := fmt.Sprintf("%s/campus/api/portal/students/%d/attendance", c.districtURL, personID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attendance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response InfiniteCampusAttendanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode attendance: %w", err)
	}

	return response.Attendance, nil
}

// GetParentStudents fetches the students associated with a parent account.
func (c *InfiniteCampusAPIClient) GetParentStudents(ctx context.Context, parentPersonID int64) ([]InfiniteCampusStudent, error) {
	url := fmt.Sprintf("%s/campus/api/portal/parents/%d/students", c.districtURL, parentPersonID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var students []InfiniteCampusStudent
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return students, nil
}

// InfiniteCampusStaffSection represents a section taught by a staff member.
type InfiniteCampusStaffSection struct {
	SectionID       int64  `json:"sectionID"`
	CourseID        int64  `json:"courseID"`
	CourseNumber    string `json:"courseNumber"`
	CourseName      string `json:"courseName"`
	SectionNumber   string `json:"sectionNumber"`
	PeriodID        int64  `json:"periodID"`
	PeriodName      string `json:"periodName"`
	RoomID          int64  `json:"roomID"`
	RoomName        string `json:"roomName"`
	EnrollmentCount int    `json:"enrollmentCount"`
	TermID          int64  `json:"termID"`
	TermName        string `json:"termName"`
}

// GetStaffSections fetches the sections/classes taught by a staff member.
func (c *InfiniteCampusAPIClient) GetStaffSections(ctx context.Context, personID int64) ([]InfiniteCampusStaffSection, error) {
	url := fmt.Sprintf("%s/campus/api/portal/staff/%d/sections", c.districtURL, personID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sections []InfiniteCampusStaffSection
	if err := json.NewDecoder(resp.Body).Decode(&sections); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return sections, nil
}

// InfiniteCampusSectionRoster represents a student in a section roster.
type InfiniteCampusSectionRoster struct {
	PersonID      int64  `json:"personID"`
	StudentNumber string `json:"studentNumber"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	GradeLevel    string `json:"gradeLevel"`
	EnrollmentID  int64  `json:"enrollmentID"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	Active        bool   `json:"active"`
}

// GetSectionRoster fetches the roster for a specific section.
func (c *InfiniteCampusAPIClient) GetSectionRoster(ctx context.Context, sectionID int64) ([]InfiniteCampusSectionRoster, error) {
	url := fmt.Sprintf("%s/campus/api/portal/sections/%d/roster", c.districtURL, sectionID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var roster []InfiniteCampusSectionRoster
	if err := json.NewDecoder(resp.Body).Decode(&roster); err != nil {
		return nil, fmt.Errorf("failed to decode roster: %w", err)
	}

	return roster, nil
}

// InfiniteCampusSchool represents a school in Infinite Campus.
type InfiniteCampusSchool struct {
	SchoolID   int64  `json:"schoolID"`
	Name       string `json:"name"`
	Number     string `json:"number"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        string `json:"zip"`
	Phone      string `json:"phone"`
	Principal  string `json:"principal"`
	GradeLow   string `json:"gradeLow"`
	GradeHigh  string `json:"gradeHigh"`
	DistrictID int64  `json:"districtID"`
}

// GetSchool fetches information about a specific school.
func (c *InfiniteCampusAPIClient) GetSchool(ctx context.Context, schoolID int64) (*InfiniteCampusSchool, error) {
	url := fmt.Sprintf("%s/campus/api/portal/schools/%d", c.districtURL, schoolID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch school: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var school InfiniteCampusSchool
	if err := json.NewDecoder(resp.Body).Decode(&school); err != nil {
		return nil, fmt.Errorf("failed to decode school: %w", err)
	}

	return &school, nil
}

// IsInfiniteCampusStudent checks if the user is a student.
func IsInfiniteCampusStudent(user *User) bool {
	return user.Extra["user_type"] == "student"
}

// IsInfiniteCampusStaff checks if the user is a staff member.
func IsInfiniteCampusStaff(user *User) bool {
	return user.Extra["user_type"] == "staff"
}

// IsInfiniteCampusParent checks if the user is a parent/guardian.
func IsInfiniteCampusParent(user *User) bool {
	return user.Extra["user_type"] == "parent"
}

// GetInfiniteCampusUserType returns the user's type as an enum.
func GetInfiniteCampusUserType(user *User) InfiniteCampusUserType {
	switch user.Extra["user_type"] {
	case "student":
		return InfiniteCampusUserTypeStudent
	case "staff":
		return InfiniteCampusUserTypeStaff
	case "parent":
		return InfiniteCampusUserTypeParent
	default:
		return InfiniteCampusUserTypeUnknown
	}
}

// GetInfiniteCampusPersonID returns the user's person ID.
func GetInfiniteCampusPersonID(user *User) string {
	return user.Extra["ic_person_id"]
}

// GetInfiniteCampusSchoolID returns the user's current school ID.
func GetInfiniteCampusSchoolID(user *User) string {
	return user.Extra["school_id"]
}

// GetInfiniteCampusDistrictID returns the user's district ID.
func GetInfiniteCampusDistrictID(user *User) string {
	return user.Extra["district_id"]
}

// GetInfiniteCampusStudentNumber returns the student number (students only).
func GetInfiniteCampusStudentNumber(user *User) string {
	return user.Extra["student_number"]
}

// GetInfiniteCampusGradeLevel returns the grade level (students only).
func GetInfiniteCampusGradeLevel(user *User) string {
	return user.Extra["grade_level"]
}

// GetInfiniteCampusStudentPersonIDs returns the person IDs of students linked to a parent.
func GetInfiniteCampusStudentPersonIDs(user *User) []string {
	ids := user.Extra["student_person_ids"]
	if ids == "" {
		return nil
	}
	return strings.Split(ids, ",")
}
