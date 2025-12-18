// auth/oauth2/edfi.go
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

// EdFiUserType represents the type of user in Ed-Fi.
type EdFiUserType string

const (
	EdFiUserTypeStudent EdFiUserType = "student"
	EdFiUserTypeStaff   EdFiUserType = "staff"
	EdFiUserTypeTeacher EdFiUserType = "teacher"
	EdFiUserTypeAdmin   EdFiUserType = "admin"
	EdFiUserTypeParent  EdFiUserType = "parent"
)

// EdFiConfig holds configuration for Ed-Fi OAuth2 authentication.
type EdFiConfig struct {
	// ClientID is the Ed-Fi OAuth2 client ID (API key).
	ClientID string

	// ClientSecret is the Ed-Fi OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL for OAuth2 authentication.
	// Example: "https://school.app/auth/edfi/callback"
	RedirectURL string

	// ODSURL is the Ed-Fi ODS (Operational Data Store) API URL for your district.
	// Example: "https://api.ed-fi.org/v5.3/api" or "https://district.ed-fi.org/ods"
	// This is required as each district has their own Ed-Fi ODS instance.
	ODSURL string

	// AuthURL is the Ed-Fi authorization server URL. If empty, uses ODSURL + "/oauth/authorize".
	// Some Ed-Fi deployments have separate auth servers.
	AuthURL string

	// TokenURL is the Ed-Fi token endpoint URL. If empty, uses ODSURL + "/oauth/token".
	// Some Ed-Fi deployments have separate auth servers.
	TokenURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: empty (Ed-Fi typically uses key-based access rather than scopes)
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

// EdFi creates a new OAuth2 provider configured for Ed-Fi authentication.
//
// Ed-Fi is a widely-adopted K-12 data standard that provides a common data model
// and API specifications for education data exchange. Many state education agencies
// and school districts use Ed-Fi as their data interoperability standard.
//
// Important: Each district/state has their own Ed-Fi ODS (Operational Data Store)
// instance, so you must configure the ODSURL for your specific deployment.
//
// Usage in BuildHandler:
//
//	edfiAuth, err := oauth2.EdFi(oauth2.EdFiConfig{
//	    ClientID:     appCfg.EdFiClientID,
//	    ClientSecret: appCfg.EdFiClientSecret,
//	    RedirectURL:  "https://school.app/auth/edfi/callback",
//	    ODSURL:       appCfg.EdFiODSURL, // e.g., "https://api.ed-fi.org/v5.3/api"
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/edfi/login", edfiAuth.LoginHandler())
//	r.Get("/auth/edfi/callback", edfiAuth.CallbackHandler())
//	r.Get("/auth/edfi/logout", edfiAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(edfiAuth.RequireAuth("/auth/edfi/login"))
//	    r.Mount("/dashboard", dashboard.Routes(deps, logger))
//	})
//
// The User.Extra map contains Ed-Fi-specific fields:
//   - "user_type": student, staff, teacher, admin, or parent
//   - "unique_id": Ed-Fi unique identifier (studentUniqueId or staffUniqueId)
//   - "local_education_agency_id": LEA ID
//   - "school_id": School ID (if associated with a single school)
//   - "school_ids": Comma-separated list of school IDs (if multiple)
//   - "staff_classification": Staff classification descriptor (staff only)
//   - "grade_level": Grade level descriptor (students only)
func EdFi(cfg EdFiConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/edfi: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/edfi: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/edfi: RedirectURL is required")
	}
	if cfg.ODSURL == "" {
		return nil, errors.New("oauth2/edfi: ODSURL is required (e.g., https://api.ed-fi.org/v5.3/api)")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/edfi: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/edfi: StateStore is required")
	}

	// Build Ed-Fi OAuth2 endpoint
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = strings.TrimSuffix(cfg.ODSURL, "/") + "/oauth/authorize"
	}

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = strings.TrimSuffix(cfg.ODSURL, "/") + "/oauth/token"
	}

	endpoint := oauth2.Endpoint{
		AuthURL:  authURL,
		TokenURL: tokenURL,
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint:     endpoint,
	}

	// Create a fetcher that captures the ODS URL
	fetchUserInfo := createEdFiUserInfoFetcher(cfg.ODSURL)

	providerCfg := &Config{
		ProviderName:  "edfi",
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

// edfiTokenInfo represents the token/identity information from Ed-Fi.
type edfiTokenInfo struct {
	// Standard OAuth2 fields
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`

	// Ed-Fi identity fields (may vary by implementation)
	StaffUniqueId             string   `json:"staffUniqueId"`
	StudentUniqueId           string   `json:"studentUniqueId"`
	ParentUniqueId            string   `json:"parentUniqueId"`
	Email                     string   `json:"email"`
	FirstName                 string   `json:"firstName"`
	LastSurname               string   `json:"lastSurname"`
	MiddleName                string   `json:"middleName"`
	LocalEducationAgencyId    string   `json:"localEducationAgencyId"`
	SchoolId                  string   `json:"schoolId"`
	SchoolIds                 []string `json:"schoolIds"`
	StaffClassification       string   `json:"staffClassification"`
	GradeLevelDescriptor      string   `json:"gradeLevelDescriptor"`
	IsTeacher                 bool     `json:"isTeacher"`
	IsAdministrator           bool     `json:"isAdministrator"`
	AssociatedOrganizationIds []string `json:"associatedOrganizationIds"`
}

// createEdFiUserInfoFetcher creates a UserInfoFetcher bound to a specific ODS URL.
func createEdFiUserInfoFetcher(odsURL string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// Try to get identity information from the token or identity endpoint
		// Ed-Fi implementations vary, so we try multiple approaches
		baseURL := strings.TrimSuffix(odsURL, "/")

		var info edfiTokenInfo

		// First try: Ed-Fi identity/me endpoint
		resp, err := client.Get(baseURL + "/identity/me")
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
				return nil, fmt.Errorf("failed to decode identity response: %w", err)
			}
		} else {
			// Second try: Ed-Fi oauth/token endpoint with token introspection
			if resp != nil {
				resp.Body.Close()
			}

			// Try userinfo endpoint
			resp, err = client.Get(baseURL + "/oauth/userinfo")
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
					return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
				}
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				// If no identity endpoint works, create minimal user from token
				// The application can use the API client to fetch additional data
				info = edfiTokenInfo{}
			}
		}

		// Determine user type and unique ID
		userType := determineEdFiUserType(&info)
		uniqueID := determineEdFiUniqueId(&info)

		// Build display name
		name := info.FirstName
		if info.LastSurname != "" {
			if name != "" {
				name += " "
			}
			name += info.LastSurname
		}

		// Build school IDs string
		schoolIDs := ""
		if len(info.SchoolIds) > 0 {
			schoolIDs = strings.Join(info.SchoolIds, ",")
		} else if info.SchoolId != "" {
			schoolIDs = info.SchoolId
		}

		// Build User ID
		userID := uniqueID
		if userID == "" {
			userID = info.Email
		}

		return &User{
			ID:            userID,
			Email:         info.Email,
			EmailVerified: info.Email != "",
			Name:          name,
			Picture:       "",
			Raw: map[string]any{
				"staffUniqueId":             info.StaffUniqueId,
				"studentUniqueId":           info.StudentUniqueId,
				"parentUniqueId":            info.ParentUniqueId,
				"email":                     info.Email,
				"firstName":                 info.FirstName,
				"lastSurname":               info.LastSurname,
				"middleName":                info.MiddleName,
				"localEducationAgencyId":    info.LocalEducationAgencyId,
				"schoolId":                  info.SchoolId,
				"schoolIds":                 info.SchoolIds,
				"staffClassification":       info.StaffClassification,
				"gradeLevelDescriptor":      info.GradeLevelDescriptor,
				"isTeacher":                 info.IsTeacher,
				"isAdministrator":           info.IsAdministrator,
				"associatedOrganizationIds": info.AssociatedOrganizationIds,
			},
			Extra: map[string]string{
				"user_type":                 string(userType),
				"unique_id":                 uniqueID,
				"staff_unique_id":           info.StaffUniqueId,
				"student_unique_id":         info.StudentUniqueId,
				"parent_unique_id":          info.ParentUniqueId,
				"local_education_agency_id": info.LocalEducationAgencyId,
				"school_id":                 info.SchoolId,
				"school_ids":                schoolIDs,
				"staff_classification":      info.StaffClassification,
				"grade_level":               info.GradeLevelDescriptor,
				"first_name":                info.FirstName,
				"last_name":                 info.LastSurname,
			},
		}, nil
	}
}

// determineEdFiUserType determines the user type from Ed-Fi identity info.
func determineEdFiUserType(info *edfiTokenInfo) EdFiUserType {
	if info.IsAdministrator {
		return EdFiUserTypeAdmin
	}
	if info.IsTeacher {
		return EdFiUserTypeTeacher
	}
	if info.StaffUniqueId != "" {
		// Check staff classification for admin roles
		classification := strings.ToLower(info.StaffClassification)
		if strings.Contains(classification, "admin") ||
			strings.Contains(classification, "principal") ||
			strings.Contains(classification, "superintendent") {
			return EdFiUserTypeAdmin
		}
		if strings.Contains(classification, "teacher") ||
			strings.Contains(classification, "instructor") {
			return EdFiUserTypeTeacher
		}
		return EdFiUserTypeStaff
	}
	if info.StudentUniqueId != "" {
		return EdFiUserTypeStudent
	}
	if info.ParentUniqueId != "" {
		return EdFiUserTypeParent
	}
	return EdFiUserTypeStaff // Default to staff for authenticated API users
}

// determineEdFiUniqueId extracts the appropriate unique ID from Ed-Fi identity info.
func determineEdFiUniqueId(info *edfiTokenInfo) string {
	if info.StudentUniqueId != "" {
		return info.StudentUniqueId
	}
	if info.StaffUniqueId != "" {
		return info.StaffUniqueId
	}
	if info.ParentUniqueId != "" {
		return info.ParentUniqueId
	}
	return ""
}

// IsEdFiStudent returns true if the user is a student.
func IsEdFiStudent(user *User) bool {
	return user.Extra["user_type"] == string(EdFiUserTypeStudent)
}

// IsEdFiStaff returns true if the user is a staff member (any type).
func IsEdFiStaff(user *User) bool {
	userType := user.Extra["user_type"]
	return userType == string(EdFiUserTypeStaff) ||
		userType == string(EdFiUserTypeTeacher) ||
		userType == string(EdFiUserTypeAdmin)
}

// IsEdFiTeacher returns true if the user is a teacher.
func IsEdFiTeacher(user *User) bool {
	return user.Extra["user_type"] == string(EdFiUserTypeTeacher)
}

// IsEdFiAdmin returns true if the user is an administrator.
func IsEdFiAdmin(user *User) bool {
	return user.Extra["user_type"] == string(EdFiUserTypeAdmin)
}

// IsEdFiParent returns true if the user is a parent.
func IsEdFiParent(user *User) bool {
	return user.Extra["user_type"] == string(EdFiUserTypeParent)
}

// GetEdFiUserType returns the Ed-Fi user type from the User.
func GetEdFiUserType(user *User) EdFiUserType {
	return EdFiUserType(user.Extra["user_type"])
}

// GetEdFiUniqueId returns the Ed-Fi unique ID (studentUniqueId, staffUniqueId, or parentUniqueId).
func GetEdFiUniqueId(user *User) string {
	return user.Extra["unique_id"]
}

// GetEdFiSchoolIds returns the school IDs associated with the user.
func GetEdFiSchoolIds(user *User) []string {
	schoolIds := user.Extra["school_ids"]
	if schoolIds == "" {
		return nil
	}
	return strings.Split(schoolIds, ",")
}

// GetEdFiLEAId returns the Local Education Agency ID.
func GetEdFiLEAId(user *User) string {
	return user.Extra["local_education_agency_id"]
}

// EdFiAPIClient provides methods to call Ed-Fi ODS API endpoints.
// Use this after authentication to fetch additional data like students, staff,
// schools, enrollments, grades, attendance, etc.
type EdFiAPIClient struct {
	client *http.Client
	odsURL string
}

// NewEdFiAPIClient creates a client for calling Ed-Fi ODS API.
// Requires a valid OAuth2 access token and the Ed-Fi ODS URL.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	edfiClient := oauth2.NewEdFiAPIClient(user.AccessToken, "https://api.ed-fi.org/v5.3/api")
//	students, err := edfiClient.GetStudents(r.Context(), nil)
func NewEdFiAPIClient(accessToken, odsURL string) *EdFiAPIClient {
	return &EdFiAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		odsURL: strings.TrimSuffix(odsURL, "/"),
	}
}

// EdFiStudent represents a student record from Ed-Fi.
type EdFiStudent struct {
	ID                       string `json:"id"` // Ed-Fi resource ID
	StudentUniqueId          string `json:"studentUniqueId"`
	FirstName                string `json:"firstName"`
	LastSurname              string `json:"lastSurname"`
	MiddleName               string `json:"middleName"`
	BirthDate                string `json:"birthDate"`
	BirthSexDescriptor       string `json:"birthSexDescriptor"`
	HispanicLatinoEthnicity  bool   `json:"hispanicLatinoEthnicity"`
	ElectronicMailAddress    string `json:"electronicMailAddress"`
	PersonalTitlePrefix      string `json:"personalTitlePrefix"`
	GenerationCodeSuffix     string `json:"generationCodeSuffix"`
	StudentIdentificationCode string `json:"studentIdentificationCode"`
}

// GetStudents fetches students from the Ed-Fi ODS.
// Optional query parameters can filter results (e.g., schoolId, limit, offset).
func (c *EdFiAPIClient) GetStudents(ctx context.Context, params map[string]string) ([]EdFiStudent, error) {
	url := c.odsURL + "/data/v3/ed-fi/students"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var students []EdFiStudent
	if err := json.NewDecoder(resp.Body).Decode(&students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	return students, nil
}

// GetStudent fetches a single student by Ed-Fi resource ID.
func (c *EdFiAPIClient) GetStudent(ctx context.Context, id string) (*EdFiStudent, error) {
	url := c.odsURL + "/data/v3/ed-fi/students/" + id

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var student EdFiStudent
	if err := json.NewDecoder(resp.Body).Decode(&student); err != nil {
		return nil, fmt.Errorf("failed to decode student: %w", err)
	}

	return &student, nil
}

// EdFiStaff represents a staff record from Ed-Fi.
type EdFiStaff struct {
	ID                    string `json:"id"` // Ed-Fi resource ID
	StaffUniqueId         string `json:"staffUniqueId"`
	FirstName             string `json:"firstName"`
	LastSurname           string `json:"lastSurname"`
	MiddleName            string `json:"middleName"`
	BirthDate             string `json:"birthDate"`
	SexDescriptor         string `json:"sexDescriptor"`
	HispanicLatinoEthnicity bool  `json:"hispanicLatinoEthnicity"`
	ElectronicMailAddress string `json:"electronicMailAddress"`
	HighestCompletedLevelOfEducationDescriptor string `json:"highestCompletedLevelOfEducationDescriptor"`
	YearsOfPriorProfessionalExperience         int    `json:"yearsOfPriorProfessionalExperience"`
}

// GetStaff fetches staff members from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetStaff(ctx context.Context, params map[string]string) ([]EdFiStaff, error) {
	url := c.odsURL + "/data/v3/ed-fi/staffs"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var staff []EdFiStaff
	if err := json.NewDecoder(resp.Body).Decode(&staff); err != nil {
		return nil, fmt.Errorf("failed to decode staff: %w", err)
	}

	return staff, nil
}

// GetStaffMember fetches a single staff member by Ed-Fi resource ID.
func (c *EdFiAPIClient) GetStaffMember(ctx context.Context, id string) (*EdFiStaff, error) {
	url := c.odsURL + "/data/v3/ed-fi/staffs/" + id

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staff member: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var staff EdFiStaff
	if err := json.NewDecoder(resp.Body).Decode(&staff); err != nil {
		return nil, fmt.Errorf("failed to decode staff member: %w", err)
	}

	return &staff, nil
}

// EdFiSchool represents a school record from Ed-Fi.
type EdFiSchool struct {
	ID                           string `json:"id"` // Ed-Fi resource ID
	SchoolId                     int64  `json:"schoolId"`
	NameOfInstitution            string `json:"nameOfInstitution"`
	ShortNameOfInstitution       string `json:"shortNameOfInstitution"`
	SchoolTypeDescriptor         string `json:"schoolTypeDescriptor"`
	LocalEducationAgencyReference struct {
		LocalEducationAgencyId int64 `json:"localEducationAgencyId"`
	} `json:"localEducationAgencyReference"`
	GradeLevels []struct {
		GradeLevelDescriptor string `json:"gradeLevelDescriptor"`
	} `json:"gradeLevels"`
	WebSite string `json:"webSite"`
}

// GetSchools fetches schools from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetSchools(ctx context.Context, params map[string]string) ([]EdFiSchool, error) {
	url := c.odsURL + "/data/v3/ed-fi/schools"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schools []EdFiSchool
	if err := json.NewDecoder(resp.Body).Decode(&schools); err != nil {
		return nil, fmt.Errorf("failed to decode schools: %w", err)
	}

	return schools, nil
}

// GetSchool fetches a single school by Ed-Fi resource ID.
func (c *EdFiAPIClient) GetSchool(ctx context.Context, id string) (*EdFiSchool, error) {
	url := c.odsURL + "/data/v3/ed-fi/schools/" + id

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch school: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var school EdFiSchool
	if err := json.NewDecoder(resp.Body).Decode(&school); err != nil {
		return nil, fmt.Errorf("failed to decode school: %w", err)
	}

	return &school, nil
}

// EdFiLocalEducationAgency represents an LEA (school district) from Ed-Fi.
type EdFiLocalEducationAgency struct {
	ID                          string `json:"id"` // Ed-Fi resource ID
	LocalEducationAgencyId      int64  `json:"localEducationAgencyId"`
	NameOfInstitution           string `json:"nameOfInstitution"`
	ShortNameOfInstitution      string `json:"shortNameOfInstitution"`
	LocalEducationAgencyCategoryDescriptor string `json:"localEducationAgencyCategoryDescriptor"`
	WebSite                     string `json:"webSite"`
}

// GetLocalEducationAgencies fetches LEAs (school districts) from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetLocalEducationAgencies(ctx context.Context, params map[string]string) ([]EdFiLocalEducationAgency, error) {
	url := c.odsURL + "/data/v3/ed-fi/localEducationAgencies"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch LEAs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var leas []EdFiLocalEducationAgency
	if err := json.NewDecoder(resp.Body).Decode(&leas); err != nil {
		return nil, fmt.Errorf("failed to decode LEAs: %w", err)
	}

	return leas, nil
}

// EdFiStudentSchoolAssociation represents a student's enrollment in a school.
type EdFiStudentSchoolAssociation struct {
	ID                       string `json:"id"` // Ed-Fi resource ID
	EntryDate                string `json:"entryDate"`
	ExitWithdrawDate         string `json:"exitWithdrawDate"`
	EntryGradeLevelDescriptor string `json:"entryGradeLevelDescriptor"`
	EntryTypeDescriptor      string `json:"entryTypeDescriptor"`
	ExitWithdrawTypeDescriptor string `json:"exitWithdrawTypeDescriptor"`
	StudentReference struct {
		StudentUniqueId string `json:"studentUniqueId"`
	} `json:"studentReference"`
	SchoolReference struct {
		SchoolId int64 `json:"schoolId"`
	} `json:"schoolReference"`
}

// GetStudentSchoolAssociations fetches student enrollments from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetStudentSchoolAssociations(ctx context.Context, params map[string]string) ([]EdFiStudentSchoolAssociation, error) {
	url := c.odsURL + "/data/v3/ed-fi/studentSchoolAssociations"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student enrollments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var associations []EdFiStudentSchoolAssociation
	if err := json.NewDecoder(resp.Body).Decode(&associations); err != nil {
		return nil, fmt.Errorf("failed to decode student enrollments: %w", err)
	}

	return associations, nil
}

// EdFiSection represents a class section from Ed-Fi.
type EdFiSection struct {
	ID                       string `json:"id"` // Ed-Fi resource ID
	SectionIdentifier        string `json:"sectionIdentifier"`
	SectionName              string `json:"sectionName"`
	AvailableCreditConversion float64 `json:"availableCreditConversion"`
	AvailableCredits         float64 `json:"availableCredits"`
	InstructionalTimePlanned int    `json:"instructionalTimePlanned"`
	CourseOfferingReference struct {
		LocalCourseCode string `json:"localCourseCode"`
		SchoolId        int64  `json:"schoolId"`
		SchoolYear      int    `json:"schoolYear"`
		SessionName     string `json:"sessionName"`
	} `json:"courseOfferingReference"`
	ClassPeriods []struct {
		ClassPeriodReference struct {
			ClassPeriodName string `json:"classPeriodName"`
			SchoolId        int64  `json:"schoolId"`
		} `json:"classPeriodReference"`
	} `json:"classPeriods"`
}

// GetSections fetches class sections from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetSections(ctx context.Context, params map[string]string) ([]EdFiSection, error) {
	url := c.odsURL + "/data/v3/ed-fi/sections"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sections []EdFiSection
	if err := json.NewDecoder(resp.Body).Decode(&sections); err != nil {
		return nil, fmt.Errorf("failed to decode sections: %w", err)
	}

	return sections, nil
}

// EdFiStaffSectionAssociation represents a staff member's assignment to a section.
type EdFiStaffSectionAssociation struct {
	ID                          string `json:"id"` // Ed-Fi resource ID
	BeginDate                   string `json:"beginDate"`
	EndDate                     string `json:"endDate"`
	ClassroomPositionDescriptor string `json:"classroomPositionDescriptor"`
	HighlyQualifiedTeacher      bool   `json:"highlyQualifiedTeacher"`
	PercentageContribution      float64 `json:"percentageContribution"`
	StaffReference struct {
		StaffUniqueId string `json:"staffUniqueId"`
	} `json:"staffReference"`
	SectionReference struct {
		LocalCourseCode   string `json:"localCourseCode"`
		SchoolId          int64  `json:"schoolId"`
		SchoolYear        int    `json:"schoolYear"`
		SectionIdentifier string `json:"sectionIdentifier"`
		SessionName       string `json:"sessionName"`
	} `json:"sectionReference"`
}

// GetStaffSectionAssociations fetches staff section assignments from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetStaffSectionAssociations(ctx context.Context, params map[string]string) ([]EdFiStaffSectionAssociation, error) {
	url := c.odsURL + "/data/v3/ed-fi/staffSectionAssociations"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staff section assignments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var associations []EdFiStaffSectionAssociation
	if err := json.NewDecoder(resp.Body).Decode(&associations); err != nil {
		return nil, fmt.Errorf("failed to decode staff section assignments: %w", err)
	}

	return associations, nil
}

// EdFiStudentSectionAssociation represents a student's enrollment in a section.
type EdFiStudentSectionAssociation struct {
	ID                       string `json:"id"` // Ed-Fi resource ID
	BeginDate                string `json:"beginDate"`
	EndDate                  string `json:"endDate"`
	HomeroomIndicator        bool   `json:"homeroomIndicator"`
	AttemptStatusDescriptor  string `json:"attemptStatusDescriptor"`
	RepeatIdentifierDescriptor string `json:"repeatIdentifierDescriptor"`
	StudentReference struct {
		StudentUniqueId string `json:"studentUniqueId"`
	} `json:"studentReference"`
	SectionReference struct {
		LocalCourseCode   string `json:"localCourseCode"`
		SchoolId          int64  `json:"schoolId"`
		SchoolYear        int    `json:"schoolYear"`
		SectionIdentifier string `json:"sectionIdentifier"`
		SessionName       string `json:"sessionName"`
	} `json:"sectionReference"`
}

// GetStudentSectionAssociations fetches student section enrollments from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetStudentSectionAssociations(ctx context.Context, params map[string]string) ([]EdFiStudentSectionAssociation, error) {
	url := c.odsURL + "/data/v3/ed-fi/studentSectionAssociations"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student section enrollments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var associations []EdFiStudentSectionAssociation
	if err := json.NewDecoder(resp.Body).Decode(&associations); err != nil {
		return nil, fmt.Errorf("failed to decode student section enrollments: %w", err)
	}

	return associations, nil
}

// EdFiGrade represents a student's grade in a section.
type EdFiGrade struct {
	ID                       string  `json:"id"` // Ed-Fi resource ID
	GradeTypeDescriptor      string  `json:"gradeTypeDescriptor"`
	LetterGradeEarned        string  `json:"letterGradeEarned"`
	NumericGradeEarned       float64 `json:"numericGradeEarned"`
	DiagnosticStatement      string  `json:"diagnosticStatement"`
	PerformanceBaseConversionDescriptor string `json:"performanceBaseConversionDescriptor"`
	StudentSectionAssociationReference struct {
		BeginDate         string `json:"beginDate"`
		LocalCourseCode   string `json:"localCourseCode"`
		SchoolId          int64  `json:"schoolId"`
		SchoolYear        int    `json:"schoolYear"`
		SectionIdentifier string `json:"sectionIdentifier"`
		SessionName       string `json:"sessionName"`
		StudentUniqueId   string `json:"studentUniqueId"`
	} `json:"studentSectionAssociationReference"`
	GradingPeriodReference struct {
		GradingPeriodDescriptor string `json:"gradingPeriodDescriptor"`
		PeriodSequence          int    `json:"periodSequence"`
		SchoolId                int64  `json:"schoolId"`
		SchoolYear              int    `json:"schoolYear"`
	} `json:"gradingPeriodReference"`
}

// GetGrades fetches student grades from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetGrades(ctx context.Context, params map[string]string) ([]EdFiGrade, error) {
	url := c.odsURL + "/data/v3/ed-fi/grades"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var grades []EdFiGrade
	if err := json.NewDecoder(resp.Body).Decode(&grades); err != nil {
		return nil, fmt.Errorf("failed to decode grades: %w", err)
	}

	return grades, nil
}

// EdFiStudentSchoolAttendanceEvent represents a student attendance event.
type EdFiStudentSchoolAttendanceEvent struct {
	ID                           string `json:"id"` // Ed-Fi resource ID
	EventDate                    string `json:"eventDate"`
	AttendanceEventCategoryDescriptor string `json:"attendanceEventCategoryDescriptor"`
	AttendanceEventReason        string `json:"attendanceEventReason"`
	EducationalEnvironmentDescriptor string `json:"educationalEnvironmentDescriptor"`
	EventDuration                float64 `json:"eventDuration"`
	StudentReference struct {
		StudentUniqueId string `json:"studentUniqueId"`
	} `json:"studentReference"`
	SchoolReference struct {
		SchoolId int64 `json:"schoolId"`
	} `json:"schoolReference"`
	SessionReference struct {
		SchoolId    int64  `json:"schoolId"`
		SchoolYear  int    `json:"schoolYear"`
		SessionName string `json:"sessionName"`
	} `json:"sessionReference"`
}

// GetStudentSchoolAttendanceEvents fetches student attendance events from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetStudentSchoolAttendanceEvents(ctx context.Context, params map[string]string) ([]EdFiStudentSchoolAttendanceEvent, error) {
	url := c.odsURL + "/data/v3/ed-fi/studentSchoolAttendanceEvents"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attendance events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var events []EdFiStudentSchoolAttendanceEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode attendance events: %w", err)
	}

	return events, nil
}

// EdFiCourse represents a course from Ed-Fi.
type EdFiCourse struct {
	ID                       string `json:"id"` // Ed-Fi resource ID
	CourseCode               string `json:"courseCode"`
	CourseTitle              string `json:"courseTitle"`
	CourseDescription        string `json:"courseDescription"`
	NumberOfParts            int    `json:"numberOfParts"`
	MaxCompletionsForCredit  int    `json:"maxCompletionsForCredit"`
	MinimumAvailableCredits  float64 `json:"minimumAvailableCredits"`
	MaximumAvailableCredits  float64 `json:"maximumAvailableCredits"`
	AcademicSubjectDescriptor string `json:"academicSubjectDescriptor"`
	CourseLevelCharacteristics []struct {
		CourseLevelCharacteristicDescriptor string `json:"courseLevelCharacteristicDescriptor"`
	} `json:"courseLevelCharacteristics"`
	EducationOrganizationReference struct {
		EducationOrganizationId int64 `json:"educationOrganizationId"`
	} `json:"educationOrganizationReference"`
}

// GetCourses fetches courses from the Ed-Fi ODS.
func (c *EdFiAPIClient) GetCourses(ctx context.Context, params map[string]string) ([]EdFiCourse, error) {
	url := c.odsURL + "/data/v3/ed-fi/courses"
	if len(params) > 0 {
		url += "?" + buildQueryString(params)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var courses []EdFiCourse
	if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
		return nil, fmt.Errorf("failed to decode courses: %w", err)
	}

	return courses, nil
}

// buildQueryString creates a URL query string from a map of parameters.
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	var parts []string
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "&")
}
