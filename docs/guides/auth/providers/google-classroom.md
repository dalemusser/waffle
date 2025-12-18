# Google Classroom Authentication

*Google OAuth2 with Classroom API access for courses, rosters, and assignments.*

Google Classroom uses the same Google OAuth2 flow but with additional scopes to access courses, rosters, and assignments.

---

## 1. Enable Google Classroom API

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your project (or create one)
3. Go to **APIs & Services → Library**
4. Search for "Google Classroom API" and **Enable** it
5. Go to **APIs & Services → Credentials**
6. Create OAuth credentials (or use existing Google OAuth credentials)
7. Add `https://yourapp.com/auth/google-classroom/callback` as a redirect URI

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    // Google Classroom uses the same credentials as Google OAuth
    GoogleClientID     string `conf:"google_client_id"`
    GoogleClientSecret string `conf:"google_client_secret"`
    GoogleClassroomRedirectURL string `conf:"google_classroom_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Google Classroom OAuth provider
classroomAuth, err := oauth2.GoogleClassroom(oauth2.GoogleClassroomConfig{
    ClientID:     appCfg.GoogleClientID,
    ClientSecret: appCfg.GoogleClientSecret,
    RedirectURL:  appCfg.GoogleClassroomRedirectURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    FetchCourses: true,  // Fetch courses during login
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Route based on role
        if oauth2.IsGoogleClassroomTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Google Classroom auth routes
r.Get("/auth/google-classroom/login", classroomAuth.LoginHandler())
r.Get("/auth/google-classroom/callback", classroomAuth.CallbackHandler())
r.Get("/auth/google-classroom/logout", classroomAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Google Classroom-specific fields:

| Field | Description |
|-------|-------------|
| `is_teacher` | `"true"` if user teaches any courses |
| `is_student` | `"true"` if user is enrolled in any courses |
| `teacher_course_count` | Number of courses where user is a teacher |
| `student_course_count` | Number of courses where user is a student |
| `courses_json` | JSON array of courses (if `FetchCourses: true`) |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user role
if oauth2.IsGoogleClassroomTeacher(user) { ... }
if oauth2.IsGoogleClassroomStudent(user) { ... }

// Get primary role (Teacher if both, Unknown if neither)
role := oauth2.GetGoogleClassroomRole(user)  // GoogleClassroomRole enum

// Get courses (if FetchCourses was true)
courses, err := oauth2.GetGoogleClassroomCourses(user)
for _, course := range courses {
    fmt.Printf("Course: %s (Role: %s)\n", course.Name, course.Role)
}
```

## API Client

For additional Classroom API calls after authentication:

```go
func getCoursesHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Create Classroom API client with user's token
    client := oauth2.NewGoogleClassroomClient(r.Context(), user.AccessToken)

    // Fetch courses
    courses, err := client.GetCourses(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(courses)
}

func getCourseRosterHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    courseID := chi.URLParam(r, "courseID")

    client := oauth2.NewGoogleClassroomClient(r.Context(), user.AccessToken)

    // Fetch students and teachers
    students, _ := client.GetCourseStudents(r.Context(), courseID)
    teachers, _ := client.GetCourseTeachers(r.Context(), courseID)

    json.NewEncoder(w).Encode(map[string]any{
        "students": students,
        "teachers": teachers,
    })
}
```

## Additional Scopes

For coursework, announcements, or other features, add scopes:

```go
classroomAuth, err := oauth2.GoogleClassroom(oauth2.GoogleClassroomConfig{
    // ... other config
    Scopes: []string{
        "openid",
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
        "https://www.googleapis.com/auth/classroom.courses.readonly",
        "https://www.googleapis.com/auth/classroom.rosters.readonly",
        // Add for coursework access:
        "https://www.googleapis.com/auth/classroom.coursework.students.readonly",
        "https://www.googleapis.com/auth/classroom.coursework.me.readonly",
        // Add for announcements:
        "https://www.googleapis.com/auth/classroom.announcements.readonly",
    },
}, logger)
```

## Package Reference

### Types

| Type | Description |
|------|-------------|
| `oauth2.GoogleClassroomConfig` | Google Classroom-specific configuration |
| `oauth2.GoogleClassroomRole` | Role enum (teacher, student, unknown) |
| `oauth2.GoogleClassroomCourse` | Course information |
| `oauth2.GoogleClassroomClient` | Classroom API client |

### Functions

| Function | Description |
|----------|-------------|
| `oauth2.GoogleClassroom(cfg, logger)` | Create Google Classroom OAuth2 provider |
| `oauth2.IsGoogleClassroomTeacher(user)` | Check if user is a teacher |
| `oauth2.IsGoogleClassroomStudent(user)` | Check if user is a student |
| `oauth2.GetGoogleClassroomRole(user)` | Get role enum |
| `oauth2.GetGoogleClassroomCourses(user)` | Get courses from User.Extra |
| `oauth2.NewGoogleClassroomClient(ctx, token)` | Create Classroom API client |

---

[← Back to OAuth2 Providers](./README.md)
