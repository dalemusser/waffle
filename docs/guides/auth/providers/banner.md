# Ellucian Banner Authentication

*Higher education ERP/SIS for colleges and universities.*

Ellucian Banner is one of the most widely used enterprise resource planning (ERP) and student information systems in higher education. It provides comprehensive functionality for student records, financial aid, HR, and finance.

Banner can be deployed on-premise or via Ellucian Cloud (SaaS).

---

## 1. Create Banner OAuth Application

1. Log in to Banner Admin
2. Navigate to the API Management section
3. Register a new OAuth2 application
4. Set your **Redirect URI**: `https://university.app/auth/banner/callback`
5. Note your **Client ID** and **Client Secret**
6. Configure appropriate scopes

For Ellucian Cloud:
1. Access the Ellucian Cloud administration portal
2. Register your application in the OAuth2 settings
3. Configure your tenant-specific endpoints

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    BannerClientID     string `conf:"banner_client_id"`
    BannerClientSecret string `conf:"banner_client_secret"`
    BannerRedirectURL  string `conf:"banner_redirect_url"`
    BannerBaseURL      string `conf:"banner_base_url"`   // e.g., "https://banner.university.edu"
    BannerTenant       string `conf:"banner_tenant"`      // For Ellucian Cloud only
}
```

## 3. Wire Up in BuildHandler

```go
// Create Banner OAuth provider
bannerAuth, err := oauth2.Banner(oauth2.BannerConfig{
    ClientID:     appCfg.BannerClientID,
    ClientSecret: appCfg.BannerClientSecret,
    RedirectURL:  appCfg.BannerRedirectURL,
    BaseURL:      appCfg.BannerBaseURL,
    Tenant:       appCfg.BannerTenant, // Optional: for Ellucian Cloud
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsBannerStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsBannerFaculty(user) {
            http.Redirect(w, r, "/faculty/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Banner auth routes
r.Get("/auth/banner/login", bannerAuth.LoginHandler())
r.Get("/auth/banner/callback", bannerAuth.CallbackHandler())
r.Get("/auth/banner/logout", bannerAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Banner-specific fields:

| Field | Description |
|-------|-------------|
| `user_type` | `student`, `faculty`, `staff`, `advisor`, `alumni`, or `applicant` |
| `banner_id` | Banner ID (SPRIDEN_ID) |
| `pidm` | Banner PIDM (internal identifier) |
| `institution_id` | Institution identifier |
| `primary_role` | Primary role in the institution |
| `roles` | Comma-separated list of all roles |
| `college` | College (students) |
| `major` | Major/program (students) |
| `class_level` | Class level - FR, SO, JR, SR, GR (students) |
| `department` | Department (faculty/staff) |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user type
if oauth2.IsBannerStudent(user) { ... }
if oauth2.IsBannerFaculty(user) { ... }
if oauth2.IsBannerStaff(user) { ... }
if oauth2.IsBannerAdvisor(user) { ... }
if oauth2.IsBannerAlumni(user) { ... }
if oauth2.IsBannerApplicant(user) { ... }
if oauth2.IsBannerEmployee(user) { ... }  // faculty or staff

// Get user type and identifiers
userType := oauth2.GetBannerUserType(user)
bannerID := oauth2.GetBannerID(user)
pidm := oauth2.GetBannerPIDM(user)
roles := oauth2.GetBannerRoles(user)

// Check for specific role
if oauth2.HasBannerRole(user, "instructor") { ... }

// Student-specific
classLevel := oauth2.GetBannerClassLevel(user)  // FR, SO, JR, SR, GR
major := oauth2.GetBannerMajor(user)
college := oauth2.GetBannerCollege(user)

// Employee-specific
department := oauth2.GetBannerDepartment(user)
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) StudentSchedule(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Create Banner API client
    bannerClient := oauth2.NewBannerAPIClient(
        user.AccessToken,
        h.Config.BannerBaseURL,
    )

    // Get current term
    currentTerm, err := bannerClient.GetCurrentTerm(r.Context())
    if err != nil {
        http.Error(w, "Failed to get current term", http.StatusInternalServerError)
        return
    }

    // Get student's courses
    pidm := oauth2.GetBannerPIDM(user)
    courses, err := bannerClient.GetStudentCourses(r.Context(), pidm, currentTerm.TermCode)
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
| `GetStudentCourses(ctx, pidm, termCode)` | Get student's course registrations |
| `GetStudentGrades(ctx, pidm, termCode)` | Get student's grades |
| `GetStudentRecord(ctx, pidm)` | Get student's academic record |
| `GetStudentHolds(ctx, pidm)` | Get student's holds |
| `GetFacultySections(ctx, pidm, termCode)` | Get sections taught by faculty |
| `GetSectionRoster(ctx, termCode, crn)` | Get class roster |
| `GetTerms(ctx)` | Get available academic terms |
| `GetCurrentTerm(ctx)` | Get current academic term |

## Data Types

| Type | Description |
|------|-------------|
| `BannerCourse` | Course registration with CRN, subject, credits, instructor |
| `BannerGrade` | Grade record with term, course, grade, quality points |
| `BannerStudentRecord` | Academic record with GPA, credits, enrollment status |
| `BannerSection` | Class section with enrollment, schedule, location |
| `BannerRoster` | Roster entry with student info, status, register date |
| `BannerTerm` | Academic term with dates and year |
| `BannerHold` | Student hold with type, dates, reason |

## Important Notes

- On-premise and Ellucian Cloud have different OAuth2 endpoints
- Set the `Tenant` field only for Ellucian Cloud deployments
- Banner uses PIDM (Person ID Master) as the internal identifier
- Banner ID (SPRIDEN_ID) is the external-facing identifier
- Term codes typically follow YYYYMM format (e.g., "202310" for Fall 2023)
- CRN (Course Reference Number) uniquely identifies a section within a term

---

[← Back to OAuth2 Providers](./README.md)
