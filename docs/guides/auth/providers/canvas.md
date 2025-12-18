# Canvas Authentication

*Popular LMS used in K-12 and higher education.*

Canvas by Instructure is one of the most widely used Learning Management Systems (LMS) in education. It provides OAuth2 authentication and comprehensive APIs for courses, assignments, grades, and more.

---

## 1. Create Canvas Developer Key

1. Log in to your Canvas instance as an **Admin**
2. Go to **Admin → Developer Keys**
3. Click **+ Developer Key → API Key**
4. Configure your application:
   - **Key Name**: Your application name
   - **Redirect URI**: `https://school.app/auth/canvas/callback`
   - **Enforce Scopes**: Enable and select required scopes
5. Save and note your **Client ID** and **Client Secret**
6. Set the key state to **ON**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    CanvasClientID     string `conf:"canvas_client_id"`
    CanvasClientSecret string `conf:"canvas_client_secret"`
    CanvasRedirectURL  string `conf:"canvas_redirect_url"`
    CanvasBaseURL      string `conf:"canvas_base_url"` // e.g., "https://canvas.instructure.com"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Canvas OAuth provider
canvasAuth, err := oauth2.Canvas(oauth2.CanvasConfig{
    ClientID:     appCfg.CanvasClientID,
    ClientSecret: appCfg.CanvasClientSecret,
    RedirectURL:  appCfg.CanvasRedirectURL,
    BaseURL:      appCfg.CanvasBaseURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsCanvasTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsCanvasStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Canvas auth routes
r.Get("/auth/canvas/login", canvasAuth.LoginHandler())
r.Get("/auth/canvas/callback", canvasAuth.CallbackHandler())
r.Get("/auth/canvas/logout", canvasAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Canvas-specific fields:

| Field | Description |
|-------|-------------|
| `canvas_user_id` | Canvas user ID |
| `primary_email` | User's primary email |
| `login_id` | User's login ID (SIS ID) |
| `integration_id` | Integration ID |
| `sis_user_id` | SIS user ID |
| `locale` | User's locale preference |
| `time_zone` | User's time zone |
| `avatar_url` | Profile avatar URL |
| `enrollments` | JSON array of course enrollments |

## Helper Functions

```go
// Check user roles (based on enrollments)
if oauth2.IsCanvasTeacher(user) { ... }
if oauth2.IsCanvasStudent(user) { ... }
if oauth2.IsCanvasTA(user) { ... }
if oauth2.IsCanvasDesigner(user) { ... }
if oauth2.IsCanvasObserver(user) { ... }
if oauth2.IsCanvasAdmin(user) { ... }

// Get user identifiers
canvasID := oauth2.GetCanvasUserID(user)
sisID := oauth2.GetCanvasSISUserID(user)
loginID := oauth2.GetCanvasLoginID(user)

// Get enrollments
enrollments := oauth2.GetCanvasEnrollments(user)
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) CourseAssignments(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    courseID := chi.URLParam(r, "courseID")

    canvasClient := oauth2.NewCanvasAPIClient(user.AccessToken, h.Config.CanvasBaseURL)

    assignments, err := canvasClient.GetCourseAssignments(r.Context(), courseID)
    if err != nil {
        http.Error(w, "Failed to fetch assignments", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(assignments)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetCourses(ctx)` | Get user's courses |
| `GetCourse(ctx, courseID)` | Get specific course |
| `GetCourseAssignments(ctx, courseID)` | Get course assignments |
| `GetCourseStudents(ctx, courseID)` | Get course roster |
| `GetUserSubmissions(ctx, courseID, userID)` | Get student submissions |
| `GetCourseGrades(ctx, courseID)` | Get course grades |
| `GetModules(ctx, courseID)` | Get course modules |
| `GetAnnouncements(ctx, courseID)` | Get course announcements |

## Scopes

Common Canvas OAuth scopes:

| Scope | Description |
|-------|-------------|
| `url:GET\|/api/v1/users/:user_id/profile` | Read user profile |
| `url:GET\|/api/v1/courses` | List courses |
| `url:GET\|/api/v1/courses/:course_id/assignments` | List assignments |
| `url:GET\|/api/v1/courses/:course_id/students` | List students |
| `url:GET\|/api/v1/courses/:course_id/enrollments` | List enrollments |

## Important Notes

- Each Canvas instance has its own URL (e.g., `https://school.instructure.com`)
- Canvas Cloud instances use `instructure.com` domain
- Self-hosted Canvas instances use custom domains
- Developer keys can be scoped to limit API access
- Refresh tokens are long-lived; access tokens expire in 1 hour

---

[← Back to OAuth2 Providers](./README.md)
