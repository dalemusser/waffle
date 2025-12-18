# Blackboard Authentication

*Enterprise LMS for higher education.*

Blackboard Learn is a widely used Learning Management System in higher education. It provides REST API authentication for accessing courses, grades, content, and more.

---

## 1. Register Blackboard Application

1. Go to the [Blackboard Developer Portal](https://developer.blackboard.com/)
2. Create a new application
3. Configure your application:
   - **Application Name**: Your app name
   - **Description**: App description
   - **Domain**: Your application domain
   - **Redirect URI**: `https://university.app/auth/blackboard/callback`
4. Note your **Application ID** and **Secret**
5. Request access to your institution's Blackboard instance

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    BlackboardClientID     string `conf:"blackboard_client_id"`
    BlackboardClientSecret string `conf:"blackboard_client_secret"`
    BlackboardRedirectURL  string `conf:"blackboard_redirect_url"`
    BlackboardBaseURL      string `conf:"blackboard_base_url"` // e.g., "https://blackboard.university.edu"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Blackboard OAuth provider
bbAuth, err := oauth2.Blackboard(oauth2.BlackboardConfig{
    ClientID:     appCfg.BlackboardClientID,
    ClientSecret: appCfg.BlackboardClientSecret,
    RedirectURL:  appCfg.BlackboardRedirectURL,
    BaseURL:      appCfg.BlackboardBaseURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsBlackboardInstructor(user) {
            http.Redirect(w, r, "/instructor/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Blackboard auth routes
r.Get("/auth/blackboard/login", bbAuth.LoginHandler())
r.Get("/auth/blackboard/callback", bbAuth.CallbackHandler())
r.Get("/auth/blackboard/logout", bbAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Blackboard-specific fields:

| Field | Description |
|-------|-------------|
| `bb_user_id` | Blackboard user ID |
| `user_uuid` | User UUID |
| `external_id` | External/SIS ID |
| `student_id` | Student ID |
| `institution_role` | Primary institution role |
| `system_roles` | Comma-separated system roles |
| `availability` | Account availability status |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user roles
if oauth2.IsBlackboardInstructor(user) { ... }
if oauth2.IsBlackboardStudent(user) { ... }
if oauth2.IsBlackboardTA(user) { ... }
if oauth2.IsBlackboardAdmin(user) { ... }
if oauth2.IsBlackboardGuest(user) { ... }

// Get user identifiers
bbUserID := oauth2.GetBlackboardUserID(user)
externalID := oauth2.GetBlackboardExternalID(user)
studentID := oauth2.GetBlackboardStudentID(user)

// Get roles
institutionRole := oauth2.GetBlackboardInstitutionRole(user)
systemRoles := oauth2.GetBlackboardSystemRoles(user)
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) CourseContent(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    courseID := chi.URLParam(r, "courseID")

    bbClient := oauth2.NewBlackboardAPIClient(user.AccessToken, h.Config.BlackboardBaseURL)

    contents, err := bbClient.GetCourseContents(r.Context(), courseID)
    if err != nil {
        http.Error(w, "Failed to fetch content", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(contents)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetCourses(ctx, userID)` | Get user's courses |
| `GetCourse(ctx, courseID)` | Get course details |
| `GetCourseContents(ctx, courseID)` | Get course content items |
| `GetCourseMemberships(ctx, courseID)` | Get course roster |
| `GetCourseGrades(ctx, courseID)` | Get course gradebook |
| `GetAssignments(ctx, courseID)` | Get course assignments |
| `GetAnnouncements(ctx, courseID)` | Get course announcements |
| `GetCalendarItems(ctx, courseID)` | Get calendar items |

## Important Notes

- Each institution has its own Blackboard instance URL
- Blackboard Ultra and Original have different UI but same API
- Application must be approved for each institution
- REST APIs require 3LO (three-legged OAuth) for user context
- Access tokens expire in 1 hour; use refresh tokens

---

[← Back to OAuth2 Providers](./README.md)
