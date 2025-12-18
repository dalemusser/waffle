# Skyward Authentication

*K-12 SIS with student/staff/family roles.*

Skyward is a widely-used Student Information System (SIS) serving thousands of school districts across the United States. It provides comprehensive student data management including grades, attendance, scheduling, fees, and family/student/staff portals.

Skyward offers two main product lines:
- **Skyward SMS 2.0**: Legacy system still used by many districts
- **Skyward Qmlativ**: Modern cloud-based platform

---

## 1. Request Skyward API Access

1. Contact your **Skyward district administrator** or **Skyward support**
2. Request API access for your application
3. Provide your **Redirect URI**: `https://school.app/auth/skyward/callback`
4. Receive your **Client ID** and **Client Secret**
5. Note your district's Skyward URL (e.g., `https://skyward.iscorp.com/districtname`)

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    SkywardClientID     string `conf:"skyward_client_id"`
    SkywardClientSecret string `conf:"skyward_client_secret"`
    SkywardRedirectURL  string `conf:"skyward_redirect_url"`
    SkywardDistrictURL  string `conf:"skyward_district_url"` // e.g., "https://skyward.iscorp.com/districtname"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Skyward OAuth provider
skywardAuth, err := oauth2.Skyward(oauth2.SkywardConfig{
    ClientID:     appCfg.SkywardClientID,
    ClientSecret: appCfg.SkywardClientSecret,
    RedirectURL:  appCfg.SkywardRedirectURL,
    DistrictURL:  appCfg.SkywardDistrictURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsSkywardStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsSkywardStaff(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsSkywardFamily(user) {
            http.Redirect(w, r, "/family/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Skyward auth routes
r.Get("/auth/skyward/login", skywardAuth.LoginHandler())
r.Get("/auth/skyward/callback", skywardAuth.CallbackHandler())
r.Get("/auth/skyward/logout", skywardAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Skyward-specific fields:

| Field | Description |
|-------|-------------|
| `skyward_id` | Skyward user ID |
| `name_id` | Name ID (unique identifier for the person) |
| `user_type` | `student`, `staff`, or `family` |
| `district_id` | District identifier |
| `entity_id` | Entity (school/building) ID |
| `entity_name` | Entity (school/building) name |
| `first_name` | User's first name |
| `last_name` | User's last name |
| `student_id` | Student ID number (students only) |
| `grade_level` | Grade level (students only) |
| `employee_id` | Employee ID (staff only) |
| `student_name_ids` | Comma-separated student name IDs (family only) |

## Helper Functions

```go
// Check user type
if oauth2.IsSkywardStudent(user) { ... }
if oauth2.IsSkywardStaff(user) { ... }
if oauth2.IsSkywardFamily(user) { ... }

// Get user type as enum
userType := oauth2.GetSkywardUserType(user)

// Get user identifiers
nameID := oauth2.GetSkywardNameID(user)
entityID := oauth2.GetSkywardEntityID(user)
districtID := oauth2.GetSkywardDistrictID(user)

// Student-specific helpers
studentID := oauth2.GetSkywardStudentID(user)
gradeLevel := oauth2.GetSkywardGradeLevel(user)

// Staff-specific helpers
employeeID := oauth2.GetSkywardEmployeeID(user)

// Family-specific helpers
studentNameIDs := oauth2.GetSkywardStudentNameIDs(user)  // []string of linked students
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) StudentGrades(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    skywardClient := oauth2.NewSkywardAPIClient(user.AccessToken, h.Config.SkywardDistrictURL)
    nameID, _ := strconv.ParseInt(oauth2.GetSkywardNameID(user), 10, 64)

    grades, err := skywardClient.GetStudentGrades(r.Context(), nameID)
    if err != nil {
        http.Error(w, "Failed to fetch grades", http.StatusInternalServerError)
        return
    }

    summary, err := skywardClient.GetStudentGradeSummary(r.Context(), nameID)
    // ...
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetStudentSchedule(ctx, nameID)` | Get student's class schedule |
| `GetStudentGrades(ctx, nameID)` | Get detailed assignment grades |
| `GetStudentGradeSummary(ctx, nameID)` | Get overall course grade summary |
| `GetStudentAttendance(ctx, nameID)` | Get attendance records |
| `GetStudentFees(ctx, nameID)` | Get fees and balances |
| `GetFamilyStudents(ctx, familyNameID)` | Get students linked to family account |
| `GetStaffSections(ctx, nameID)` | Get sections taught by staff member |
| `GetSectionRoster(ctx, sectionID)` | Get students in a section |
| `GetEntity(ctx, entityID)` | Get school/building information |
| `GetGradingPeriods(ctx, entityID)` | Get grading periods for entity |

## Important Notes

- Each district has their own Skyward instance with a unique URL
- URLs typically follow patterns like `https://skyward.iscorp.com/districtname`
- API access requires district administrator approval
- Skyward uses "entities" to refer to schools/buildings within a district
- Family accounts can access data for their linked students only
- Staff can access rosters and student data for their assigned sections
- Skyward tracks fees and balances that can be retrieved via the API

---

[← Back to OAuth2 Providers](./README.md)
