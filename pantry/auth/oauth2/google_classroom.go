// auth/oauth2/google_classroom.go
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
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleClassroomRole represents the role of a user in Google Classroom.
type GoogleClassroomRole string

const (
	GoogleClassroomRoleStudent GoogleClassroomRole = "student"
	GoogleClassroomRoleTeacher GoogleClassroomRole = "teacher"
	GoogleClassroomRoleUnknown GoogleClassroomRole = "unknown"
)

// GoogleClassroomConfig holds configuration for Google Classroom OAuth2 authentication.
type GoogleClassroomConfig struct {
	// ClientID is the Google OAuth2 client ID.
	ClientID string

	// ClientSecret is the Google OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Google.
	// Example: "https://school.app/auth/google-classroom/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default includes identity scopes plus classroom.courses.readonly and classroom.rosters.readonly
	// Additional scopes can be added for coursework, announcements, etc.
	Scopes []string

	// FetchCourses controls whether to fetch the user's courses during authentication.
	// If true, courses are fetched and stored in User.Extra["courses_json"].
	// Default: false (to minimize API calls during login)
	FetchCourses bool

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

// GoogleClassroom creates a new OAuth2 provider configured for Google Classroom authentication.
//
// Google Classroom uses Google OAuth2 with additional scopes for accessing classroom data.
// This provider automatically includes scopes for viewing courses and rosters.
//
// Usage in BuildHandler:
//
//	classroomAuth, err := oauth2.GoogleClassroom(oauth2.GoogleClassroomConfig{
//	    ClientID:     appCfg.GoogleClientID,
//	    ClientSecret: appCfg.GoogleClientSecret,
//	    RedirectURL:  "https://school.app/auth/google-classroom/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	    FetchCourses: true,  // Optional: fetch courses during login
//	}, logger)
//
//	r.Get("/auth/google-classroom/login", classroomAuth.LoginHandler())
//	r.Get("/auth/google-classroom/callback", classroomAuth.CallbackHandler())
//	r.Get("/auth/google-classroom/logout", classroomAuth.LogoutHandler())
//
//	r.Group(func(r chi.Router) {
//	    r.Use(classroomAuth.RequireAuth("/auth/google-classroom/login"))
//	    r.Mount("/classroom", classroom.Routes(deps, logger))
//	})
//
// The User.Extra map contains Google Classroom-specific fields:
//   - "is_teacher": "true" if user teaches any courses, "false" otherwise
//   - "is_student": "true" if user is enrolled in any courses, "false" otherwise
//   - "courses_json": JSON array of courses (if FetchCourses is true)
//   - "teacher_course_count": Number of courses where user is a teacher
//   - "student_course_count": Number of courses where user is a student
//
// The access token stored in User.AccessToken can be used to make additional
// Classroom API calls for coursework, announcements, etc.
func GoogleClassroom(cfg GoogleClassroomConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/google-classroom: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/google-classroom: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/google-classroom: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/google-classroom: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/google-classroom: StateStore is required")
	}

	// Default scopes for Google Classroom
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/classroom.courses.readonly",
			"https://www.googleapis.com/auth/classroom.rosters.readonly",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	// Create a custom user info fetcher that includes Classroom data
	fetchUserInfo := createGoogleClassroomUserInfoFetcher(cfg.FetchCourses, logger)

	providerCfg := &Config{
		ProviderName:  "google-classroom",
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

// GoogleClassroomCourse represents a Google Classroom course.
type GoogleClassroomCourse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Section        string `json:"section,omitempty"`
	Room           string `json:"room,omitempty"`
	DescriptionHeading string `json:"descriptionHeading,omitempty"`
	CourseState    string `json:"courseState"`
	EnrollmentCode string `json:"enrollmentCode,omitempty"`
	// Role indicates the user's role in this course
	Role GoogleClassroomRole `json:"role"`
}

// classroomCoursesResponse represents the response from the Classroom courses.list API.
type classroomCoursesResponse struct {
	Courses       []classroomCourse `json:"courses"`
	NextPageToken string            `json:"nextPageToken"`
}

type classroomCourse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Section            string `json:"section"`
	Room               string `json:"room"`
	DescriptionHeading string `json:"descriptionHeading"`
	CourseState        string `json:"courseState"`
	EnrollmentCode     string `json:"enrollmentCode"`
	OwnerId            string `json:"ownerId"`
}

// createGoogleClassroomUserInfoFetcher creates a UserInfoFetcher that includes Classroom data.
func createGoogleClassroomUserInfoFetcher(fetchCourses bool, logger *zap.Logger) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

		// First, fetch basic user info
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user info: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code from userinfo: %d", resp.StatusCode)
		}

		var info googleUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode user info: %w", err)
		}

		user := &User{
			ID:            info.ID,
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
			Name:          info.Name,
			Picture:       info.Picture,
			Raw: map[string]any{
				"id":             info.ID,
				"email":          info.Email,
				"verified_email": info.EmailVerified,
				"name":           info.Name,
				"given_name":     info.GivenName,
				"family_name":    info.FamilyName,
				"picture":        info.Picture,
				"locale":         info.Locale,
			},
			Extra: map[string]string{
				"first_name": info.GivenName,
				"last_name":  info.FamilyName,
			},
		}

		// Fetch Classroom courses to determine role
		teacherCourses, studentCourses, err := fetchClassroomCourses(ctx, client, info.ID)
		if err != nil {
			// Log the error but don't fail authentication
			if logger != nil {
				logger.Warn("failed to fetch Classroom courses",
					zap.String("user_id", info.ID),
					zap.Error(err),
				)
			}
			user.Extra["is_teacher"] = "unknown"
			user.Extra["is_student"] = "unknown"
			user.Extra["teacher_course_count"] = "0"
			user.Extra["student_course_count"] = "0"
		} else {
			user.Extra["is_teacher"] = fmt.Sprintf("%t", len(teacherCourses) > 0)
			user.Extra["is_student"] = fmt.Sprintf("%t", len(studentCourses) > 0)
			user.Extra["teacher_course_count"] = fmt.Sprintf("%d", len(teacherCourses))
			user.Extra["student_course_count"] = fmt.Sprintf("%d", len(studentCourses))

			// Optionally include full course data
			if fetchCourses {
				allCourses := make([]GoogleClassroomCourse, 0, len(teacherCourses)+len(studentCourses))
				for _, c := range teacherCourses {
					allCourses = append(allCourses, GoogleClassroomCourse{
						ID:                 c.ID,
						Name:               c.Name,
						Section:            c.Section,
						Room:               c.Room,
						DescriptionHeading: c.DescriptionHeading,
						CourseState:        c.CourseState,
						EnrollmentCode:     c.EnrollmentCode,
						Role:               GoogleClassroomRoleTeacher,
					})
				}
				for _, c := range studentCourses {
					allCourses = append(allCourses, GoogleClassroomCourse{
						ID:                 c.ID,
						Name:               c.Name,
						Section:            c.Section,
						Room:               c.Room,
						DescriptionHeading: c.DescriptionHeading,
						CourseState:        c.CourseState,
						Role:               GoogleClassroomRoleStudent,
					})
				}
				coursesJSON, _ := json.Marshal(allCourses)
				user.Extra["courses_json"] = string(coursesJSON)
			}
		}

		return user, nil
	}
}

// fetchClassroomCourses fetches the user's courses from Google Classroom API.
// Returns separate lists of courses where the user is a teacher vs student.
func fetchClassroomCourses(ctx context.Context, client *http.Client, userID string) ([]classroomCourse, []classroomCourse, error) {
	var teacherCourses, studentCourses []classroomCourse

	// Fetch courses where user is a teacher
	teacherResp, err := client.Get("https://classroom.googleapis.com/v1/courses?teacherId=me&courseStates=ACTIVE")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch teacher courses: %w", err)
	}
	defer teacherResp.Body.Close()

	if teacherResp.StatusCode == http.StatusOK {
		var resp classroomCoursesResponse
		if err := json.NewDecoder(teacherResp.Body).Decode(&resp); err == nil {
			teacherCourses = resp.Courses
		}
	}

	// Fetch courses where user is a student
	studentResp, err := client.Get("https://classroom.googleapis.com/v1/courses?studentId=me&courseStates=ACTIVE")
	if err != nil {
		return teacherCourses, nil, fmt.Errorf("failed to fetch student courses: %w", err)
	}
	defer studentResp.Body.Close()

	if studentResp.StatusCode == http.StatusOK {
		var resp classroomCoursesResponse
		if err := json.NewDecoder(studentResp.Body).Decode(&resp); err == nil {
			studentCourses = resp.Courses
		}
	}

	return teacherCourses, studentCourses, nil
}

// IsGoogleClassroomTeacher returns true if the user is a teacher in any Google Classroom course.
func IsGoogleClassroomTeacher(user *User) bool {
	return user.Extra["is_teacher"] == "true"
}

// IsGoogleClassroomStudent returns true if the user is a student in any Google Classroom course.
func IsGoogleClassroomStudent(user *User) bool {
	return user.Extra["is_student"] == "true"
}

// GetGoogleClassroomRole returns the primary role of the user in Google Classroom.
// If the user is both a teacher and student, returns Teacher.
// If role could not be determined, returns Unknown.
func GetGoogleClassroomRole(user *User) GoogleClassroomRole {
	if user.Extra["is_teacher"] == "true" {
		return GoogleClassroomRoleTeacher
	}
	if user.Extra["is_student"] == "true" {
		return GoogleClassroomRoleStudent
	}
	return GoogleClassroomRoleUnknown
}

// GetGoogleClassroomCourses parses and returns the user's courses from User.Extra.
// Returns nil if courses were not fetched during authentication (FetchCourses was false).
func GetGoogleClassroomCourses(user *User) ([]GoogleClassroomCourse, error) {
	coursesJSON, ok := user.Extra["courses_json"]
	if !ok || coursesJSON == "" {
		return nil, nil
	}

	var courses []GoogleClassroomCourse
	if err := json.Unmarshal([]byte(coursesJSON), &courses); err != nil {
		return nil, fmt.Errorf("failed to parse courses: %w", err)
	}
	return courses, nil
}

// GoogleClassroomClient provides helper methods for making Google Classroom API calls.
// Create one using the user's access token after authentication.
type GoogleClassroomClient struct {
	client *http.Client
}

// NewGoogleClassroomClient creates a new Classroom API client using the user's access token.
func NewGoogleClassroomClient(ctx context.Context, accessToken string) *GoogleClassroomClient {
	token := &oauth2.Token{AccessToken: accessToken}
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	return &GoogleClassroomClient{client: client}
}

// GetCourses fetches all active courses for the authenticated user.
func (c *GoogleClassroomClient) GetCourses(ctx context.Context) ([]GoogleClassroomCourse, error) {
	teacherCourses, studentCourses, err := fetchClassroomCourses(ctx, c.client, "me")
	if err != nil {
		return nil, err
	}

	allCourses := make([]GoogleClassroomCourse, 0, len(teacherCourses)+len(studentCourses))
	for _, course := range teacherCourses {
		allCourses = append(allCourses, GoogleClassroomCourse{
			ID:                 course.ID,
			Name:               course.Name,
			Section:            course.Section,
			Room:               course.Room,
			DescriptionHeading: course.DescriptionHeading,
			CourseState:        course.CourseState,
			EnrollmentCode:     course.EnrollmentCode,
			Role:               GoogleClassroomRoleTeacher,
		})
	}
	for _, course := range studentCourses {
		allCourses = append(allCourses, GoogleClassroomCourse{
			ID:                 course.ID,
			Name:               course.Name,
			Section:            course.Section,
			Room:               course.Room,
			DescriptionHeading: course.DescriptionHeading,
			CourseState:        course.CourseState,
			Role:               GoogleClassroomRoleStudent,
		})
	}

	return allCourses, nil
}

// GetCourseStudents fetches all students in a specific course.
func (c *GoogleClassroomClient) GetCourseStudents(ctx context.Context, courseID string) ([]GoogleClassroomStudent, error) {
	url := fmt.Sprintf("https://classroom.googleapis.com/v1/courses/%s/students", courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Students []struct {
			UserID  string `json:"userId"`
			Profile struct {
				ID           string `json:"id"`
				Name         struct {
					GivenName  string `json:"givenName"`
					FamilyName string `json:"familyName"`
					FullName   string `json:"fullName"`
				} `json:"name"`
				EmailAddress string `json:"emailAddress"`
				PhotoUrl     string `json:"photoUrl"`
			} `json:"profile"`
		} `json:"students"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}

	students := make([]GoogleClassroomStudent, len(result.Students))
	for i, s := range result.Students {
		students[i] = GoogleClassroomStudent{
			UserID:    s.UserID,
			Email:     s.Profile.EmailAddress,
			Name:      s.Profile.Name.FullName,
			FirstName: s.Profile.Name.GivenName,
			LastName:  s.Profile.Name.FamilyName,
			PhotoURL:  s.Profile.PhotoUrl,
		}
	}

	return students, nil
}

// GetCourseTeachers fetches all teachers in a specific course.
func (c *GoogleClassroomClient) GetCourseTeachers(ctx context.Context, courseID string) ([]GoogleClassroomTeacher, error) {
	url := fmt.Sprintf("https://classroom.googleapis.com/v1/courses/%s/teachers", courseID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teachers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Teachers []struct {
			UserID  string `json:"userId"`
			Profile struct {
				ID           string `json:"id"`
				Name         struct {
					GivenName  string `json:"givenName"`
					FamilyName string `json:"familyName"`
					FullName   string `json:"fullName"`
				} `json:"name"`
				EmailAddress string `json:"emailAddress"`
				PhotoUrl     string `json:"photoUrl"`
			} `json:"profile"`
		} `json:"teachers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode teachers: %w", err)
	}

	teachers := make([]GoogleClassroomTeacher, len(result.Teachers))
	for i, t := range result.Teachers {
		teachers[i] = GoogleClassroomTeacher{
			UserID:    t.UserID,
			Email:     t.Profile.EmailAddress,
			Name:      t.Profile.Name.FullName,
			FirstName: t.Profile.Name.GivenName,
			LastName:  t.Profile.Name.FamilyName,
			PhotoURL:  t.Profile.PhotoUrl,
		}
	}

	return teachers, nil
}

// GoogleClassroomStudent represents a student in a Google Classroom course.
type GoogleClassroomStudent struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PhotoURL  string `json:"photoUrl"`
}

// GoogleClassroomTeacher represents a teacher in a Google Classroom course.
type GoogleClassroomTeacher struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PhotoURL  string `json:"photoUrl"`
}
