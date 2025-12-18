# PowerSchool Authentication

*K-12 SIS with student/teacher/parent/admin roles.*

PowerSchool is one of the most widely used Student Information Systems (SIS) in K-12 education. It provides OAuth2/OIDC authentication and access to student, teacher, parent, and administrator data.

**Important**: Each school district has their own PowerSchool server, so you must configure the server URL for your specific district.

---

## 1. Create PowerSchool Plugin

1. Log in to your PowerSchool Admin portal
2. Go to **System → System Settings → Plugin Management Configuration**
3. Click **Install** and create a new plugin
4. Enable **OAuth 2.0** for the plugin
5. Set your **Redirect URI**: `https://school.app/auth/powerschool/callback`
6. Save your **Client ID** and **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    PowerSchoolClientID     string `conf:"powerschool_client_id"`
    PowerSchoolClientSecret string `conf:"powerschool_client_secret"`
    PowerSchoolRedirectURL  string `conf:"powerschool_redirect_url"`
    PowerSchoolServerURL    string `conf:"powerschool_server_url"` // e.g., "https://district.powerschool.com"
}
```

## 3. Wire Up in BuildHandler

```go
// Create PowerSchool OAuth provider
powerSchoolAuth, err := oauth2.PowerSchool(oauth2.PowerSchoolConfig{
    ClientID:     appCfg.PowerSchoolClientID,
    ClientSecret: appCfg.PowerSchoolClientSecret,
    RedirectURL:  appCfg.PowerSchoolRedirectURL,
    ServerURL:    appCfg.PowerSchoolServerURL, // Required: district's PowerSchool URL
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        switch oauth2.GetPowerSchoolUserType(user) {
        case oauth2.PowerSchoolUserTypeStudent:
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        case oauth2.PowerSchoolUserTypeTeacher:
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        case oauth2.PowerSchoolUserTypeParent:
            http.Redirect(w, r, "/parent/dashboard", http.StatusTemporaryRedirect)
        default:
            http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// PowerSchool auth routes
r.Get("/auth/powerschool/login", powerSchoolAuth.LoginHandler())
r.Get("/auth/powerschool/callback", powerSchoolAuth.CallbackHandler())
r.Get("/auth/powerschool/logout", powerSchoolAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains PowerSchool-specific fields:

| Field | Description |
|-------|-------------|
| `user_type` | `student`, `teacher`, `admin`, or `parent` |
| `dcid` | PowerSchool internal database ID |
| `student_dcid` | Student DCID (for parents, links to their student) |
| `school_id` | Current school ID |
| `district_id` | District ID |
| `grade_level` | Student's grade level (students only) |
| `student_number` | Student number/ID (students only) |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user type
if oauth2.IsPowerSchoolStudent(user) { ... }
if oauth2.IsPowerSchoolTeacher(user) { ... }
if oauth2.IsPowerSchoolAdmin(user) { ... }
if oauth2.IsPowerSchoolParent(user) { ... }

// Get user type
userType := oauth2.GetPowerSchoolUserType(user)  // PowerSchoolUserType enum
```

## API Client

For additional PowerSchool API calls after authentication:

```go
func getStudentScheduleHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Create PowerSchool API client
    psClient := oauth2.NewPowerSchoolAPIClient(
        user.AccessToken,
        "https://district.powerschool.com",
    )

    // Get student DCID from user info
    studentDCID, _ := strconv.ParseInt(user.Extra["dcid"], 10, 64)

    // Fetch student's class sections
    sections, err := psClient.GetStudentSections(r.Context(), studentDCID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(sections)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetStudent(ctx, dcid)` | Get student record |
| `GetStudentSections(ctx, dcid)` | Get student's class sections |
| `GetSchool(ctx, schoolID)` | Get school information |

## Package Reference

### Types

| Type | Description |
|------|-------------|
| `oauth2.PowerSchoolConfig` | PowerSchool-specific configuration |
| `oauth2.PowerSchoolUserType` | User type enum (student, teacher, admin, parent) |
| `oauth2.PowerSchoolAPIClient` | API client for PowerSchool |
| `oauth2.PowerSchoolStudent` | Student record |
| `oauth2.PowerSchoolSection` | Class section |
| `oauth2.PowerSchoolSchool` | School record |

### Functions

| Function | Description |
|----------|-------------|
| `oauth2.PowerSchool(cfg, logger)` | Create PowerSchool OAuth2 provider |
| `oauth2.NewPowerSchoolAPIClient(token, url)` | Create API client |
| `oauth2.IsPowerSchoolStudent(user)` | Check if user is a student |
| `oauth2.IsPowerSchoolTeacher(user)` | Check if user is a teacher |
| `oauth2.IsPowerSchoolAdmin(user)` | Check if user is an admin |
| `oauth2.IsPowerSchoolParent(user)` | Check if user is a parent |
| `oauth2.GetPowerSchoolUserType(user)` | Get user type enum |

---

[← Back to OAuth2 Providers](./README.md)
