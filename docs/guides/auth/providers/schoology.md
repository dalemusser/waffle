# Schoology Authentication

*K-12 LMS with course and grade management.*

Schoology is a popular K-12 Learning Management System that provides course management, gradebook, and collaboration tools. It uses OAuth 1.0a for API authentication.

**Note**: Schoology uses OAuth 1.0a, not OAuth 2.0. The WAFFLE implementation handles this transparently.

---

## 1. Create Schoology API Credentials

1. Log in to Schoology as an **Administrator**
2. Go to **Tools → API** (or `your-school.schoology.com/api`)
3. Click **Request API Credentials**
4. Configure your application:
   - **App Name**: Your application name
   - **Organization**: Your organization
   - **Callback URL**: `https://school.app/auth/schoology/callback`
5. Note your **Consumer Key** and **Consumer Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    SchoologyConsumerKey    string `conf:"schoology_consumer_key"`
    SchoologyConsumerSecret string `conf:"schoology_consumer_secret"`
    SchoologyRedirectURL    string `conf:"schoology_redirect_url"`
    SchoologyDomain         string `conf:"schoology_domain"` // e.g., "school.schoology.com"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Schoology OAuth provider
schoologyAuth, err := oauth2.Schoology(oauth2.SchoologyConfig{
    ConsumerKey:    appCfg.SchoologyConsumerKey,
    ConsumerSecret: appCfg.SchoologyConsumerSecret,
    RedirectURL:    appCfg.SchoologyRedirectURL,
    Domain:         appCfg.SchoologyDomain,
    SessionStore:   sessionStore,
    StateStore:     stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsSchoologyTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsSchoologyStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/parent/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Schoology auth routes
r.Get("/auth/schoology/login", schoologyAuth.LoginHandler())
r.Get("/auth/schoology/callback", schoologyAuth.CallbackHandler())
r.Get("/auth/schoology/logout", schoologyAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Schoology-specific fields:

| Field | Description |
|-------|-------------|
| `schoology_uid` | Schoology user ID |
| `school_uid` | School/building ID |
| `building_id` | Building ID |
| `school_nid` | School node ID |
| `name_title` | Title (Mr., Mrs., etc.) |
| `name_first` | First name |
| `name_last` | Last name |
| `primary_email` | Primary email address |
| `role_title` | Role title (e.g., "Student", "Teacher") |
| `type` | Account type |
| `tz_name` | Timezone name |
| `grad_year` | Graduation year (students) |

## Helper Functions

```go
// Check user roles
if oauth2.IsSchoologyTeacher(user) { ... }
if oauth2.IsSchoologyStudent(user) { ... }
if oauth2.IsSchoologyParent(user) { ... }
if oauth2.IsSchoologyAdmin(user) { ... }

// Get user identifiers
schoologyUID := oauth2.GetSchoologyUID(user)
schoolUID := oauth2.GetSchoologySchoolUID(user)
buildingID := oauth2.GetSchoologyBuildingID(user)

// Get student-specific info
gradYear := oauth2.GetSchoologyGradYear(user)
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) UserCourses(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    schoologyClient := oauth2.NewSchoologyAPIClient(
        h.Config.SchoologyConsumerKey,
        h.Config.SchoologyConsumerSecret,
        user.AccessToken,
        user.Extra["access_token_secret"],
        h.Config.SchoologyDomain,
    )

    courses, err := schoologyClient.GetUserCourses(r.Context())
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
| `GetUserCourses(ctx)` | Get user's courses |
| `GetCourse(ctx, courseID)` | Get course details |
| `GetCourseSection(ctx, sectionID)` | Get section details |
| `GetSectionEnrollments(ctx, sectionID)` | Get section roster |
| `GetSectionAssignments(ctx, sectionID)` | Get section assignments |
| `GetSectionGrades(ctx, sectionID)` | Get section gradebook |
| `GetUserGroups(ctx)` | Get user's groups |
| `GetSchoolInfo(ctx, schoolID)` | Get school information |

## Important Notes

- Schoology uses OAuth 1.0a, not OAuth 2.0
- The `access_token_secret` is stored in `User.Extra` for API calls
- Each school district has their own Schoology domain
- API rate limits apply (check Schoology documentation)
- Parent accounts can access linked student data

---

[← Back to OAuth2 Providers](./README.md)
