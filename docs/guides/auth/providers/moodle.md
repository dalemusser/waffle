# Moodle Authentication

*Open-source LMS with OAuth2 plugin support.*

Moodle is the world's most widely used open-source Learning Management System. It supports OAuth2 authentication through plugins and provides comprehensive APIs for managing courses, users, and content.

---

## 1. Configure Moodle OAuth2 Server

1. Log in to Moodle as an **Administrator**
2. Go to **Site administration → Server → OAuth 2 services**
3. Click **Create new custom service**
4. Configure:
   - **Name**: Your application name
   - **Client ID**: Generate or specify
   - **Client Secret**: Generate a secure secret
   - **Service base URL**: Your Moodle URL
   - **Redirect URI**: `https://school.app/auth/moodle/callback`
5. Enable required scopes
6. Save and note credentials

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    MoodleClientID     string `conf:"moodle_client_id"`
    MoodleClientSecret string `conf:"moodle_client_secret"`
    MoodleRedirectURL  string `conf:"moodle_redirect_url"`
    MoodleBaseURL      string `conf:"moodle_base_url"` // e.g., "https://moodle.school.edu"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Moodle OAuth provider
moodleAuth, err := oauth2.Moodle(oauth2.MoodleConfig{
    ClientID:     appCfg.MoodleClientID,
    ClientSecret: appCfg.MoodleClientSecret,
    RedirectURL:  appCfg.MoodleRedirectURL,
    BaseURL:      appCfg.MoodleBaseURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsMoodleTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Moodle auth routes
r.Get("/auth/moodle/login", moodleAuth.LoginHandler())
r.Get("/auth/moodle/callback", moodleAuth.CallbackHandler())
r.Get("/auth/moodle/logout", moodleAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Moodle-specific fields:

| Field | Description |
|-------|-------------|
| `moodle_user_id` | Moodle user ID |
| `username` | Moodle username |
| `idnumber` | External ID number |
| `institution` | Institution name |
| `department` | Department |
| `city` | City |
| `country` | Country code |
| `timezone` | User timezone |
| `lang` | Preferred language |
| `theme` | Preferred theme |
| `first_name` | First name |
| `last_name` | Last name |

## Helper Functions

```go
// Check user roles (based on system roles)
if oauth2.IsMoodleTeacher(user) { ... }
if oauth2.IsMoodleStudent(user) { ... }
if oauth2.IsMoodleManager(user) { ... }
if oauth2.IsMoodleAdmin(user) { ... }
if oauth2.IsMoodleCourseCreator(user) { ... }

// Get user identifiers
moodleID := oauth2.GetMoodleUserID(user)
username := oauth2.GetMoodleUsername(user)
idNumber := oauth2.GetMoodleIDNumber(user)
```

## API Client

For post-authentication API calls using Moodle Web Services:

```go
func (h *Handler) UserCourses(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    moodleClient := oauth2.NewMoodleAPIClient(user.AccessToken, h.Config.MoodleBaseURL)

    userID, _ := strconv.ParseInt(oauth2.GetMoodleUserID(user), 10, 64)
    courses, err := moodleClient.GetUserCourses(r.Context(), userID)
    if err != nil {
        http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(courses)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetUserCourses(ctx, userID)` | Get user's enrolled courses |
| `GetCourse(ctx, courseID)` | Get course details |
| `GetCourseContents(ctx, courseID)` | Get course sections and modules |
| `GetEnrolledUsers(ctx, courseID)` | Get course participants |
| `GetUserGrades(ctx, courseID, userID)` | Get user's grades |
| `GetAssignments(ctx, courseID)` | Get course assignments |
| `GetForums(ctx, courseID)` | Get course forums |
| `GetCalendarEvents(ctx, courseID)` | Get calendar events |

## Required Moodle Web Services

Ensure these web service functions are enabled:

| Function | Description |
|----------|-------------|
| `core_webservice_get_site_info` | Get site info and user data |
| `core_user_get_users_by_field` | Get user details |
| `core_enrol_get_users_courses` | Get user's courses |
| `core_course_get_contents` | Get course content |
| `core_enrol_get_enrolled_users` | Get enrolled users |
| `gradereport_user_get_grade_items` | Get user grades |
| `mod_assign_get_assignments` | Get assignments |

## Important Notes

- Moodle is self-hosted; each institution has its own URL
- OAuth2 requires the OAuth2 server plugin (included in Moodle 3.3+)
- Web Services must be enabled in Site administration
- External services need appropriate function permissions
- Token expiration is configurable in Moodle settings

---

[← Back to OAuth2 Providers](./README.md)
