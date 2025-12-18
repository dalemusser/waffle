// auth/oauth2/gg4l.go
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

// GG4LUserType represents a user's type in GG4L.
type GG4LUserType string

const (
	// GG4LUserTypeStudent represents a student in GG4L.
	GG4LUserTypeStudent GG4LUserType = "student"

	// GG4LUserTypeTeacher represents a teacher in GG4L.
	GG4LUserTypeTeacher GG4LUserType = "teacher"

	// GG4LUserTypeAdmin represents an administrator in GG4L.
	GG4LUserTypeAdmin GG4LUserType = "admin"

	// GG4LUserTypeParent represents a parent/guardian in GG4L.
	GG4LUserTypeParent GG4LUserType = "parent"

	// GG4LUserTypeStaff represents other staff in GG4L.
	GG4LUserTypeStaff GG4LUserType = "staff"

	// GG4LUserTypeUnknown represents an unknown user type.
	GG4LUserTypeUnknown GG4LUserType = "unknown"
)

// GG4LConfig holds configuration for GG4L OAuth2 authentication.
type GG4LConfig struct {
	// ClientID is the GG4L OAuth2 client ID.
	ClientID string

	// ClientSecret is the GG4L OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with GG4L.
	// Example: "https://myapp.com/auth/gg4l/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: openid, profile, email
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

// GG4L creates a new OAuth2 provider configured for GG4L (Global Grid for Learning) authentication.
//
// GG4L (Global Grid for Learning) is an education data platform that provides SSO (Single Sign-On)
// and rostering services for K-12 schools. It enables secure data exchange between schools and
// educational applications using industry standards like OneRoster and SIF.
//
// GG4L Connect provides:
//   - Single Sign-On for students, teachers, parents, and staff
//   - OneRoster-compliant rostering data
//   - SIF (Schools Interoperability Framework) integration
//   - Secure data transport between SIS and learning applications
//
// Setup in GG4L:
//  1. Register as a vendor at https://www.gg4l.com/vendors
//  2. Create an application in the GG4L Developer Portal
//  3. Configure your Redirect URI
//  4. Obtain your Client ID and Client Secret
//  5. Configure the scopes your application needs
//
// Usage in BuildHandler:
//
//	gg4lAuth, err := oauth2.GG4L(oauth2.GG4LConfig{
//	    ClientID:     appCfg.GG4LClientID,
//	    ClientSecret: appCfg.GG4LClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/gg4l/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/gg4l/login", gg4lAuth.LoginHandler())
//	r.Get("/auth/gg4l/callback", gg4lAuth.CallbackHandler())
//	r.Get("/auth/gg4l/logout", gg4lAuth.LogoutHandler())
//
// The User.Extra map contains GG4L-specific fields:
//   - "gg4l_id": GG4L user ID
//   - "source_id": Source system ID (from SIS)
//   - "user_type": "student", "teacher", "admin", "parent", or "staff"
//   - "district_id": District identifier
//   - "district_name": District name
//   - "school_ids": Comma-separated list of school IDs
//   - "school_names": Comma-separated list of school names
//   - "first_name": User's first name
//   - "last_name": User's last name
//   - "middle_name": User's middle name
//   - "grade_level": Grade level (students only)
//   - "graduation_year": Expected graduation year (students only)
//   - "student_ids": Comma-separated student IDs (parents only)
//   - "org_unit_ids": Comma-separated organizational unit IDs
func GG4L(cfg GG4LConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/gg4l: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/gg4l: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/gg4l: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/gg4l: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/gg4l: StateStore is required")
	}

	// GG4L OAuth2 endpoints
	endpoint := oauth2.Endpoint{
		AuthURL:  "https://sso.gg4l.com/oauth/authorize",
		TokenURL: "https://sso.gg4l.com/oauth/token",
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

	// Create a fetcher that uses the GG4L API
	fetchUserInfo := createGG4LUserInfoFetcher()

	providerCfg := &Config{
		ProviderName:  "gg4l",
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

// gg4lUserInfo represents the response from GG4L user info endpoint.
type gg4lUserInfo struct {
	ID           string   `json:"id"`
	SourceID     string   `json:"sourceId"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	MiddleName   string   `json:"middleName"`
	UserType     string   `json:"userType"` // student, teacher, admin, parent, staff
	DistrictID   string   `json:"districtId"`
	DistrictName string   `json:"districtName"`
	SchoolIDs    []string `json:"schoolIds"`
	SchoolNames  []string `json:"schoolNames"`
	OrgUnitIDs   []string `json:"orgUnitIds"`
	Picture      string   `json:"picture"`

	// Student-specific fields
	GradeLevel     string `json:"gradeLevel,omitempty"`
	GraduationYear int    `json:"graduationYear,omitempty"`

	// Parent-specific fields
	StudentIDs []string `json:"studentIds,omitempty"`
}

// createGG4LUserInfoFetcher creates a UserInfoFetcher for GG4L.
func createGG4LUserInfoFetcher() UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Fetch user info from GG4L API
		resp, err := client.Get("https://sso.gg4l.com/api/v1/userinfo")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var info gg4lUserInfo
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
			"gg4l_id":       info.ID,
			"source_id":     info.SourceID,
			"user_type":     info.UserType,
			"district_id":   info.DistrictID,
			"district_name": info.DistrictName,
			"first_name":    info.FirstName,
			"last_name":     info.LastName,
			"middle_name":   info.MiddleName,
		}

		if len(info.SchoolIDs) > 0 {
			extra["school_ids"] = strings.Join(info.SchoolIDs, ",")
		}
		if len(info.SchoolNames) > 0 {
			extra["school_names"] = strings.Join(info.SchoolNames, ",")
		}
		if len(info.OrgUnitIDs) > 0 {
			extra["org_unit_ids"] = strings.Join(info.OrgUnitIDs, ",")
		}

		// Add user-type specific fields
		switch info.UserType {
		case "student":
			extra["grade_level"] = info.GradeLevel
			if info.GraduationYear > 0 {
				extra["graduation_year"] = fmt.Sprintf("%d", info.GraduationYear)
			}
		case "parent":
			if len(info.StudentIDs) > 0 {
				extra["student_ids"] = strings.Join(info.StudentIDs, ",")
			}
		}

		return &User{
			ID:            info.ID,
			Email:         info.Email,
			EmailVerified: info.Email != "",
			Name:          name,
			Picture:       info.Picture,
			Raw: map[string]any{
				"id":             info.ID,
				"sourceId":       info.SourceID,
				"username":       info.Username,
				"email":          info.Email,
				"firstName":      info.FirstName,
				"lastName":       info.LastName,
				"middleName":     info.MiddleName,
				"userType":       info.UserType,
				"districtId":     info.DistrictID,
				"districtName":   info.DistrictName,
				"schoolIds":      info.SchoolIDs,
				"schoolNames":    info.SchoolNames,
				"orgUnitIds":     info.OrgUnitIDs,
				"gradeLevel":     info.GradeLevel,
				"graduationYear": info.GraduationYear,
				"studentIds":     info.StudentIDs,
			},
			Extra: extra,
		}, nil
	}
}

// GG4LAPIClient provides methods to call GG4L API endpoints.
// Use this after authentication for additional GG4L API operations.
type GG4LAPIClient struct {
	client *http.Client
}

// NewGG4LAPIClient creates a client for calling GG4L API.
// Requires a valid OAuth2 access token.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	gg4lClient := oauth2.NewGG4LAPIClient(user.AccessToken)
//	schools, err := gg4lClient.GetSchools(r.Context())
func NewGG4LAPIClient(accessToken string) *GG4LAPIClient {
	return &GG4LAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
	}
}

// GG4LSchool represents a school in GG4L.
type GG4LSchool struct {
	ID           string `json:"id"`
	SourceID     string `json:"sourceId"`
	Name         string `json:"name"`
	SchoolNumber string `json:"schoolNumber"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Zip          string `json:"zip"`
	Phone        string `json:"phone"`
	Principal    string `json:"principal"`
	GradeLow     string `json:"gradeLow"`
	GradeHigh    string `json:"gradeHigh"`
	DistrictID   string `json:"districtId"`
	DistrictName string `json:"districtName"`
}

// GG4LDistrict represents a district in GG4L.
type GG4LDistrict struct {
	ID       string `json:"id"`
	SourceID string `json:"sourceId"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Phone    string `json:"phone"`
}

// GG4LStudent represents a student in GG4L.
type GG4LStudent struct {
	ID             string   `json:"id"`
	SourceID       string   `json:"sourceId"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	MiddleName     string   `json:"middleName"`
	Email          string   `json:"email"`
	GradeLevel     string   `json:"gradeLevel"`
	GraduationYear int      `json:"graduationYear"`
	StudentNumber  string   `json:"studentNumber"`
	SchoolIDs      []string `json:"schoolIds"`
	OrgUnitIDs     []string `json:"orgUnitIds"`
	Active         bool     `json:"active"`
}

// GG4LTeacher represents a teacher in GG4L.
type GG4LTeacher struct {
	ID             string   `json:"id"`
	SourceID       string   `json:"sourceId"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	MiddleName     string   `json:"middleName"`
	Email          string   `json:"email"`
	EmployeeNumber string   `json:"employeeNumber"`
	Title          string   `json:"title"`
	SchoolIDs      []string `json:"schoolIds"`
	OrgUnitIDs     []string `json:"orgUnitIds"`
	Active         bool     `json:"active"`
}

// GG4LClass represents a class/section in GG4L.
type GG4LClass struct {
	ID           string   `json:"id"`
	SourceID     string   `json:"sourceId"`
	Title        string   `json:"title"`
	CourseID     string   `json:"courseId"`
	CourseTitle  string   `json:"courseTitle"`
	CourseCode   string   `json:"courseCode"`
	SchoolID     string   `json:"schoolId"`
	SchoolName   string   `json:"schoolName"`
	TeacherIDs   []string `json:"teacherIds"`
	Period       string   `json:"period"`
	Room         string   `json:"room"`
	Subject      string   `json:"subject"`
	GradeLevel   string   `json:"gradeLevel"`
	TermID       string   `json:"termId"`
	TermName     string   `json:"termName"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
}

// GG4LEnrollment represents a student's enrollment in a class.
type GG4LEnrollment struct {
	ID         string `json:"id"`
	StudentID  string `json:"studentId"`
	ClassID    string `json:"classId"`
	Role       string `json:"role"` // student, teacher, aide
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Primary    bool   `json:"primary"`
	Active     bool   `json:"active"`
}

// GG4LOrgUnit represents an organizational unit in GG4L.
type GG4LOrgUnit struct {
	ID          string `json:"id"`
	SourceID    string `json:"sourceId"`
	Name        string `json:"name"`
	Type        string `json:"type"` // school, department, grade, etc.
	ParentID    string `json:"parentId"`
	DistrictID  string `json:"districtId"`
}

// GetSchools fetches schools the user has access to.
func (c *GG4LAPIClient) GetSchools(ctx context.Context) ([]GG4LSchool, error) {
	resp, err := c.client.Get("https://api.gg4l.com/v1/schools")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schools []GG4LSchool
	if err := json.NewDecoder(resp.Body).Decode(&schools); err != nil {
		return nil, fmt.Errorf("failed to decode schools: %w", err)
	}

	return schools, nil
}

// GetSchool fetches a specific school by ID.
func (c *GG4LAPIClient) GetSchool(ctx context.Context, schoolID string) (*GG4LSchool, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/schools/%s", schoolID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch school: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var school GG4LSchool
	if err := json.NewDecoder(resp.Body).Decode(&school); err != nil {
		return nil, fmt.Errorf("failed to decode school: %w", err)
	}

	return &school, nil
}

// GetDistrict fetches the district information.
func (c *GG4LAPIClient) GetDistrict(ctx context.Context, districtID string) (*GG4LDistrict, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/districts/%s", districtID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch district: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var district GG4LDistrict
	if err := json.NewDecoder(resp.Body).Decode(&district); err != nil {
		return nil, fmt.Errorf("failed to decode district: %w", err)
	}

	return &district, nil
}

// GetStudents fetches students for a school.
func (c *GG4LAPIClient) GetStudents(ctx context.Context, schoolID string) ([]GG4LStudent, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/schools/%s/students", schoolID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var students []GG4LStudent
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return students, nil
}

// GetTeachers fetches teachers for a school.
func (c *GG4LAPIClient) GetTeachers(ctx context.Context, schoolID string) ([]GG4LTeacher, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/schools/%s/teachers", schoolID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teachers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var teachers []GG4LTeacher
	if err := json.NewDecoder(resp.Body).Decode(&teachers); err != nil {
		return nil, fmt.Errorf("failed to decode teachers: %w", err)
	}

	return teachers, nil
}

// GetClasses fetches classes for a school.
func (c *GG4LAPIClient) GetClasses(ctx context.Context, schoolID string) ([]GG4LClass, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/schools/%s/classes", schoolID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch classes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var classes []GG4LClass
	if err := json.NewDecoder(resp.Body).Decode(&classes); err != nil {
		return nil, fmt.Errorf("failed to decode classes: %w", err)
	}

	return classes, nil
}

// GetClassEnrollments fetches enrollments for a class.
func (c *GG4LAPIClient) GetClassEnrollments(ctx context.Context, classID string) ([]GG4LEnrollment, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/classes/%s/enrollments", classID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrollments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var enrollments []GG4LEnrollment
	if err := json.NewDecoder(resp.Body).Decode(&enrollments); err != nil {
		return nil, fmt.Errorf("failed to decode enrollments: %w", err)
	}

	return enrollments, nil
}

// GetStudentClasses fetches classes for a specific student.
func (c *GG4LAPIClient) GetStudentClasses(ctx context.Context, studentID string) ([]GG4LClass, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/students/%s/classes", studentID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student classes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var classes []GG4LClass
	if err := json.NewDecoder(resp.Body).Decode(&classes); err != nil {
		return nil, fmt.Errorf("failed to decode classes: %w", err)
	}

	return classes, nil
}

// GetTeacherClasses fetches classes for a specific teacher.
func (c *GG4LAPIClient) GetTeacherClasses(ctx context.Context, teacherID string) ([]GG4LClass, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/teachers/%s/classes", teacherID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teacher classes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var classes []GG4LClass
	if err := json.NewDecoder(resp.Body).Decode(&classes); err != nil {
		return nil, fmt.Errorf("failed to decode classes: %w", err)
	}

	return classes, nil
}

// GetOrgUnits fetches organizational units for a district.
func (c *GG4LAPIClient) GetOrgUnits(ctx context.Context, districtID string) ([]GG4LOrgUnit, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/districts/%s/orgunits", districtID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch org units: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var orgUnits []GG4LOrgUnit
	if err := json.NewDecoder(resp.Body).Decode(&orgUnits); err != nil {
		return nil, fmt.Errorf("failed to decode org units: %w", err)
	}

	return orgUnits, nil
}

// GetParentStudents fetches students associated with a parent.
func (c *GG4LAPIClient) GetParentStudents(ctx context.Context, parentID string) ([]GG4LStudent, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://api.gg4l.com/v1/parents/%s/students", parentID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var students []GG4LStudent
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return students, nil
}

// IsGG4LStudent checks if the user is a student.
func IsGG4LStudent(user *User) bool {
	return user.Extra["user_type"] == "student"
}

// IsGG4LTeacher checks if the user is a teacher.
func IsGG4LTeacher(user *User) bool {
	return user.Extra["user_type"] == "teacher"
}

// IsGG4LAdmin checks if the user is an administrator.
func IsGG4LAdmin(user *User) bool {
	return user.Extra["user_type"] == "admin"
}

// IsGG4LParent checks if the user is a parent/guardian.
func IsGG4LParent(user *User) bool {
	return user.Extra["user_type"] == "parent"
}

// IsGG4LStaff checks if the user is staff (non-teaching).
func IsGG4LStaff(user *User) bool {
	return user.Extra["user_type"] == "staff"
}

// GetGG4LUserType returns the user's type as an enum.
func GetGG4LUserType(user *User) GG4LUserType {
	switch user.Extra["user_type"] {
	case "student":
		return GG4LUserTypeStudent
	case "teacher":
		return GG4LUserTypeTeacher
	case "admin":
		return GG4LUserTypeAdmin
	case "parent":
		return GG4LUserTypeParent
	case "staff":
		return GG4LUserTypeStaff
	default:
		return GG4LUserTypeUnknown
	}
}

// GetGG4LID returns the GG4L user ID.
func GetGG4LID(user *User) string {
	return user.Extra["gg4l_id"]
}

// GetGG4LSourceID returns the source system ID (from SIS).
func GetGG4LSourceID(user *User) string {
	return user.Extra["source_id"]
}

// GetGG4LDistrictID returns the user's district ID.
func GetGG4LDistrictID(user *User) string {
	return user.Extra["district_id"]
}

// GetGG4LDistrictName returns the user's district name.
func GetGG4LDistrictName(user *User) string {
	return user.Extra["district_name"]
}

// GetGG4LSchoolIDs returns the user's school IDs.
func GetGG4LSchoolIDs(user *User) []string {
	ids := user.Extra["school_ids"]
	if ids == "" {
		return nil
	}
	return strings.Split(ids, ",")
}

// GetGG4LSchoolNames returns the user's school names.
func GetGG4LSchoolNames(user *User) []string {
	names := user.Extra["school_names"]
	if names == "" {
		return nil
	}
	return strings.Split(names, ",")
}

// GetGG4LGradeLevel returns the student's grade level.
func GetGG4LGradeLevel(user *User) string {
	return user.Extra["grade_level"]
}

// GetGG4LStudentIDs returns the student IDs linked to a parent.
func GetGG4LStudentIDs(user *User) []string {
	ids := user.Extra["student_ids"]
	if ids == "" {
		return nil
	}
	return strings.Split(ids, ",")
}

// GetGG4LOrgUnitIDs returns the user's organizational unit IDs.
func GetGG4LOrgUnitIDs(user *User) []string {
	ids := user.Extra["org_unit_ids"]
	if ids == "" {
		return nil
	}
	return strings.Split(ids, ",")
}
