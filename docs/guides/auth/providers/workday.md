# Workday Student Authentication

*Cloud-based student information system for higher education.*

Workday Student is a cloud-based student information system and ERP used by colleges and universities. It provides comprehensive student lifecycle management including admissions, student records, academic advising, financial aid, and billing.

---

## 1. Create Workday OAuth Application

1. Access your Workday tenant administration
2. Navigate to **Integration → API Clients**
3. Create a new OAuth2 API client
4. Configure the **Redirect URI**: `https://university.app/auth/workday/callback`
5. Note your **Client ID** and **Client Secret**
6. Configure appropriate scopes and security groups

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    WorkdayClientID     string `conf:"workday_client_id"`
    WorkdayClientSecret string `conf:"workday_client_secret"`
    WorkdayRedirectURL  string `conf:"workday_redirect_url"`
    WorkdayTenantURL    string `conf:"workday_tenant_url"`   // e.g., "https://wd5-impl-services1.workday.com"
    WorkdayTenantName   string `conf:"workday_tenant_name"`  // e.g., "university_student"
}
```

## 3. Wire Up in BuildHandler

```go
// Create Workday OAuth provider
workdayAuth, err := oauth2.Workday(oauth2.WorkdayConfig{
    ClientID:     appCfg.WorkdayClientID,
    ClientSecret: appCfg.WorkdayClientSecret,
    RedirectURL:  appCfg.WorkdayRedirectURL,
    TenantURL:    appCfg.WorkdayTenantURL,
    TenantName:   appCfg.WorkdayTenantName,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsWorkdayStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsWorkdayFaculty(user) {
            http.Redirect(w, r, "/faculty/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Workday auth routes
r.Get("/auth/workday/login", workdayAuth.LoginHandler())
r.Get("/auth/workday/callback", workdayAuth.CallbackHandler())
r.Get("/auth/workday/logout", workdayAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Workday-specific fields:

| Field | Description |
|-------|-------------|
| `user_type` | `student`, `prospect`, `applicant`, `faculty`, `staff`, `advisor`, `instructor`, `alumni` |
| `workday_id` | Workday Worker ID or Student ID |
| `wid` | Workday internal ID (WID) |
| `student_id` | Student ID (students only) |
| `employee_id` | Employee ID (faculty/staff only) |
| `academic_level` | Academic level (undergraduate, graduate, etc.) |
| `academic_period` | Current academic period |
| `program` | Academic program (students only) |
| `department` | Department (faculty/staff only) |
| `primary_position` | Primary job position (faculty/staff only) |
| `first_name` | User's first name |
| `last_name` | User's last name |
| `preferred_name` | User's preferred/chosen name |

## Helper Functions

```go
// Check user type
if oauth2.IsWorkdayStudent(user) { ... }
if oauth2.IsWorkdayProspect(user) { ... }
if oauth2.IsWorkdayApplicant(user) { ... }
if oauth2.IsWorkdayFaculty(user) { ... }
if oauth2.IsWorkdayStaff(user) { ... }
if oauth2.IsWorkdayAdvisor(user) { ... }
if oauth2.IsWorkdayInstructor(user) { ... }
if oauth2.IsWorkdayAlumni(user) { ... }
if oauth2.IsWorkdayEmployee(user) { ... }  // faculty, staff, or instructor

// Get user type and identifiers
userType := oauth2.GetWorkdayUserType(user)
workdayID := oauth2.GetWorkdayID(user)
wid := oauth2.GetWorkdayWID(user)
studentID := oauth2.GetWorkdayStudentID(user)
employeeID := oauth2.GetWorkdayEmployeeID(user)
roles := oauth2.GetWorkdayRoles(user)

// Student-specific
academicLevel := oauth2.GetWorkdayAcademicLevel(user)
program := oauth2.GetWorkdayProgram(user)

// Employee-specific
department := oauth2.GetWorkdayDepartment(user)
position := oauth2.GetWorkdayPosition(user)
```

## API Client

For post-authentication API calls:

```go
func (h *Handler) StudentSchedule(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Create Workday API client
    wdClient := oauth2.NewWorkdayAPIClient(
        user.AccessToken,
        h.Config.WorkdayTenantURL,
        h.Config.WorkdayTenantName,
    )

    // Get student's courses
    studentID := oauth2.GetWorkdayStudentID(user)
    courses, err := wdClient.GetStudentCourses(r.Context(), studentID)
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
| `GetStudentCourses(ctx, studentID)` | Get student's course enrollments |
| `GetStudentGrades(ctx, studentID, period)` | Get student's grades |
| `GetStudentRecord(ctx, studentID)` | Get student's academic record |
| `GetInstructorSections(ctx, instructorID)` | Get sections taught by instructor |
| `GetSectionRoster(ctx, sectionWID)` | Get class roster |
| `GetAcademicPeriods(ctx)` | Get available academic periods |
| `GetCurrentAcademicPeriod(ctx)` | Get current academic period |
| `GetAdvisees(ctx, advisorID)` | Get advisees for an advisor |

## Data Types

| Type | Description |
|------|-------------|
| `WorkdayCourse` | Course enrollment with subject, credits, grade, instructor |
| `WorkdayGrade` | Grade record with course, grade, grade points |
| `WorkdayStudentRecord` | Academic record with GPA, credits, program, major |
| `WorkdaySection` | Class section with enrollment, meeting patterns |
| `WorkdayRoster` | Roster entry with student info, grade, status |
| `WorkdayAcademicPeriod` | Academic period with dates and type |
| `WorkdayAdvisee` | Advisee record with student info, advisor type |

## Important Notes

- Workday is cloud-only - there are no on-premise deployments
- WID (Workday ID) is the internal identifier for all objects
- Workday uses "Workers" for employees and separate student records
- Users can have multiple roles (e.g., student + employee)
- Preferred name support is built-in (for chosen names)
- Academic periods can be semesters, quarters, or terms
- API access requires appropriate security group membership

---

[← Back to OAuth2 Providers](./README.md)
