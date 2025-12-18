# Ed-Fi Authentication

*K-12 data standard with ODS API access.*

Ed-Fi is a widely-adopted data standard and technology framework for K-12 education. The Ed-Fi Operational Data Store (ODS) provides a standardized API for accessing student, staff, and school data across different SIS systems.

---

## 1. Register Ed-Fi API Client

1. Contact your **Ed-Fi ODS administrator**
2. Request API client credentials for your application
3. Provide your **Redirect URI**: `https://school.app/auth/edfi/callback`
4. Receive your **Client ID** (Key) and **Client Secret**
5. Note your Ed-Fi ODS URL

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    EdFiClientID     string `conf:"edfi_client_id"`
    EdFiClientSecret string `conf:"edfi_client_secret"`
    EdFiRedirectURL  string `conf:"edfi_redirect_url"`
    EdFiBaseURL      string `conf:"edfi_base_url"` // e.g., "https://api.ed-fi.org/v5.3/api"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Ed-Fi OAuth provider
edfiAuth, err := oauth2.EdFi(oauth2.EdFiConfig{
    ClientID:     appCfg.EdFiClientID,
    ClientSecret: appCfg.EdFiClientSecret,
    RedirectURL:  appCfg.EdFiRedirectURL,
    BaseURL:      appCfg.EdFiBaseURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        switch oauth2.GetEdFiUserType(user) {
        case oauth2.EdFiUserTypeStudent:
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        case oauth2.EdFiUserTypeTeacher:
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        case oauth2.EdFiUserTypeStaff:
            http.Redirect(w, r, "/staff/dashboard", http.StatusTemporaryRedirect)
        case oauth2.EdFiUserTypeParent:
            http.Redirect(w, r, "/parent/dashboard", http.StatusTemporaryRedirect)
        default:
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Ed-Fi auth routes
r.Get("/auth/edfi/login", edfiAuth.LoginHandler())
r.Get("/auth/edfi/callback", edfiAuth.CallbackHandler())
r.Get("/auth/edfi/logout", edfiAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Ed-Fi-specific fields:

| Field | Description |
|-------|-------------|
| `unique_id` | Ed-Fi unique ID |
| `user_type` | `student`, `teacher`, `staff`, `admin`, or `parent` |
| `staff_unique_id` | Staff unique ID (staff/teachers) |
| `student_unique_id` | Student unique ID (students) |
| `parent_unique_id` | Parent unique ID (parents) |
| `first_name` | First name |
| `last_name` | Last name |
| `school_ids` | Comma-separated school IDs |
| `lea_id` | Local Education Agency ID |
| `grades` | Grade levels (students) |

## Helper Functions

```go
// Check user type
if oauth2.IsEdFiStudent(user) { ... }
if oauth2.IsEdFiTeacher(user) { ... }
if oauth2.IsEdFiStaff(user) { ... }
if oauth2.IsEdFiAdmin(user) { ... }
if oauth2.IsEdFiParent(user) { ... }

// Get user type as enum
userType := oauth2.GetEdFiUserType(user)

// Get user identifiers
uniqueID := oauth2.GetEdFiUniqueId(user)
schoolIDs := oauth2.GetEdFiSchoolIds(user)  // []string
leaID := oauth2.GetEdFiLEAId(user)
```

## API Client

For post-authentication API calls to Ed-Fi ODS:

```go
func (h *Handler) StudentGrades(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    edfiClient := oauth2.NewEdFiAPIClient(user.AccessToken, h.Config.EdFiBaseURL)

    studentID := oauth2.GetEdFiUniqueId(user)
    grades, err := edfiClient.GetStudentGrades(r.Context(), studentID)
    if err != nil {
        http.Error(w, "Failed to fetch grades", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(grades)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetStudent(ctx, uniqueID)` | Get student record |
| `GetStudentSchoolAssociations(ctx, studentID)` | Get student's schools |
| `GetStudentGrades(ctx, studentID)` | Get student grades |
| `GetStudentAttendance(ctx, studentID)` | Get attendance records |
| `GetStaff(ctx, uniqueID)` | Get staff record |
| `GetStaffSchoolAssociations(ctx, staffID)` | Get staff's schools |
| `GetSchool(ctx, schoolID)` | Get school information |
| `GetSections(ctx, schoolID)` | Get class sections |
| `GetSectionStudents(ctx, sectionID)` | Get section roster |
| `GetParent(ctx, uniqueID)` | Get parent record |
| `GetParentStudents(ctx, parentID)` | Get parent's students |
| `GetCalendar(ctx, schoolID)` | Get school calendar |
| `GetGradingPeriods(ctx, schoolID)` | Get grading periods |
| `GetLocalEducationAgency(ctx, leaID)` | Get LEA information |

## Ed-Fi Data Model

Ed-Fi uses a standardized data model:

| Entity | Description |
|--------|-------------|
| Student | Student demographic and enrollment data |
| Staff | Staff/teacher information |
| Parent | Parent/guardian information |
| School | School/building information |
| LocalEducationAgency | District/LEA information |
| Section | Class sections |
| Grade | Student grades |
| Attendance | Attendance events |
| Calendar | School calendar |

## Important Notes

- Ed-Fi ODS API versions vary; confirm your district's version
- Each district/state may have different Ed-Fi implementations
- API access requires proper authorization from the LEA
- Ed-Fi uses unique IDs that are consistent across systems
- The ODS contains operational (current) data, not historical

---

[← Back to OAuth2 Providers](./README.md)
