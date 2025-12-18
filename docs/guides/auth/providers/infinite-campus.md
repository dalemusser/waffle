# Infinite Campus Authentication

*K-12 SIS with student/staff/parent roles.*

Infinite Campus is one of the most widely used Student Information Systems (SIS) in K-12 education in the United States. It provides comprehensive student data management including grades, attendance, scheduling, and more.

**Important**: Each school district has their own Infinite Campus instance, so you must configure the district URL for your specific district.

---

## 1. Create Infinite Campus OAuth Application

1. Log in to Infinite Campus as a **System Administrator**
2. Go to **System Administration → Portal → Portal Options → OAuth Applications**
3. Click **Add** and configure your application
4. Set your **Redirect URI**: `https://school.app/auth/infinitecampus/callback`
5. Note your **Client ID** and **Client Secret**
6. Configure the appropriate scopes for your application

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    InfiniteCampusClientID     string `conf:"infinitecampus_client_id"`
    InfiniteCampusClientSecret string `conf:"infinitecampus_client_secret"`
    InfiniteCampusRedirectURL  string `conf:"infinitecampus_redirect_url"`
    InfiniteCampusDistrictURL  string `conf:"infinitecampus_district_url"` // e.g., "https://campus.district.k12.state.us"
    InfiniteCampusAppName      string `conf:"infinitecampus_app_name"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Infinite Campus OAuth provider
icAuth, err := oauth2.InfiniteCampus(oauth2.InfiniteCampusConfig{
    ClientID:     appCfg.InfiniteCampusClientID,
    ClientSecret: appCfg.InfiniteCampusClientSecret,
    RedirectURL:  appCfg.InfiniteCampusRedirectURL,
    DistrictURL:  appCfg.InfiniteCampusDistrictURL,
    AppName:      appCfg.InfiniteCampusAppName,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        switch oauth2.GetInfiniteCampusUserType(user) {
        case oauth2.InfiniteCampusUserTypeStudent:
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        case oauth2.InfiniteCampusUserTypeStaff:
            http.Redirect(w, r, "/staff/dashboard", http.StatusTemporaryRedirect)
        case oauth2.InfiniteCampusUserTypeParent:
            http.Redirect(w, r, "/parent/dashboard", http.StatusTemporaryRedirect)
        default:
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Infinite Campus auth routes
r.Get("/auth/infinitecampus/login", icAuth.LoginHandler())
r.Get("/auth/infinitecampus/callback", icAuth.CallbackHandler())
r.Get("/auth/infinitecampus/logout", icAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Infinite Campus-specific fields:

| Field | Description |
|-------|-------------|
| `ic_person_id` | Infinite Campus person ID |
| `ic_user_id` | Infinite Campus user ID |
| `user_type` | `student`, `staff`, or `parent` |
| `district_id` | District identifier |
| `calendar_id` | Current calendar ID |
| `school_id` | Current school ID |
| `school_name` | Current school name |
| `first_name` | User's first name |
| `last_name` | User's last name |
| `student_number` | Student number (students only) |
| `grade_level` | Grade level (students only) |
| `staff_number` | Staff number (staff only) |
| `student_person_ids` | Comma-separated student IDs (parents only) |

## Helper Functions

```go
// Check user type
if oauth2.IsInfiniteCampusStudent(user) { ... }
if oauth2.IsInfiniteCampusStaff(user) { ... }
if oauth2.IsInfiniteCampusParent(user) { ... }

// Get user type as enum
userType := oauth2.GetInfiniteCampusUserType(user)

// Get IDs
personID := oauth2.GetInfiniteCampusPersonID(user)
schoolID := oauth2.GetInfiniteCampusSchoolID(user)
districtID := oauth2.GetInfiniteCampusDistrictID(user)

// Student-specific
studentNumber := oauth2.GetInfiniteCampusStudentNumber(user)
gradeLevel := oauth2.GetInfiniteCampusGradeLevel(user)

// Parent-specific - get linked student IDs
studentIDs := oauth2.GetInfiniteCampusStudentPersonIDs(user)
```

## API Client

For additional API calls after authentication:

```go
func getStudentScheduleHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    icClient := oauth2.NewInfiniteCampusAPIClient(
        user.AccessToken,
        "https://campus.district.k12.state.us",
    )

    personID, _ := strconv.ParseInt(oauth2.GetInfiniteCampusPersonID(user), 10, 64)

    schedule, err := icClient.GetStudentSchedule(r.Context(), personID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(schedule)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetStudentSchedule(ctx, personID)` | Get student's class schedule |
| `GetStudentGrades(ctx, personID)` | Get student's grades |
| `GetStudentAttendance(ctx, personID)` | Get attendance records |
| `GetParentStudents(ctx, parentPersonID)` | Get linked students for parent |
| `GetStaffSections(ctx, personID)` | Get sections for staff member |
| `GetSectionRoster(ctx, sectionID)` | Get class roster |
| `GetSchool(ctx, schoolID)` | Get school information |

## Important Notes

- Each district has their own Infinite Campus instance with a unique URL
- URLs typically follow patterns like `https://campus.district.k12.state.us` or `https://districtname.infinitecampus.org`
- API access requires district administrator approval
- Parents can access data for their linked students only
- Staff can access rosters and student data for their assigned sections

---

[← Back to OAuth2 Providers](./README.md)
