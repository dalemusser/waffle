# GG4L (Global Grid for Learning) Authentication

*K-12 SSO and rostering platform with OneRoster support.*

GG4L (Global Grid for Learning) is an education data platform that provides SSO (Single Sign-On) and rostering services for K-12 schools. It enables secure data exchange between schools and educational applications using industry standards like OneRoster and SIF.

GG4L Connect provides:
- Single Sign-On for students, teachers, parents, and staff
- OneRoster-compliant rostering data
- SIF (Schools Interoperability Framework) integration
- Secure data transport between SIS and learning applications

---

## 1. Register as a GG4L Vendor

1. Register as a vendor at [GG4L Vendor Portal](https://www.gg4l.com/vendors)
2. Create an application in the GG4L Developer Portal
3. Set your **Redirect URI**: `https://school.app/auth/gg4l/callback`
4. Obtain your **Client ID** and **Client Secret**
5. Configure the scopes your application needs

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    GG4LClientID     string `conf:"gg4l_client_id"`
    GG4LClientSecret string `conf:"gg4l_client_secret"`
    GG4LRedirectURL  string `conf:"gg4l_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create GG4L OAuth provider
gg4lAuth, err := oauth2.GG4L(oauth2.GG4LConfig{
    ClientID:     appCfg.GG4LClientID,
    ClientSecret: appCfg.GG4LClientSecret,
    RedirectURL:  appCfg.GG4LRedirectURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Route based on user type
        if oauth2.IsGG4LStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsGG4LTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsGG4LParent(user) {
            http.Redirect(w, r, "/parent/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// GG4L auth routes
r.Get("/auth/gg4l/login", gg4lAuth.LoginHandler())
r.Get("/auth/gg4l/callback", gg4lAuth.CallbackHandler())
r.Get("/auth/gg4l/logout", gg4lAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains GG4L-specific fields:

| Field | Description |
|-------|-------------|
| `gg4l_id` | GG4L user ID |
| `source_id` | Source system ID (from SIS) |
| `user_type` | `student`, `teacher`, `admin`, `parent`, or `staff` |
| `district_id` | District identifier |
| `district_name` | District name |
| `school_ids` | Comma-separated list of school IDs |
| `school_names` | Comma-separated list of school names |
| `first_name` | User's first name |
| `last_name` | User's last name |
| `middle_name` | User's middle name |
| `grade_level` | Grade level (students only) |
| `graduation_year` | Expected graduation year (students only) |
| `student_ids` | Comma-separated student IDs (parents only) |
| `org_unit_ids` | Comma-separated organizational unit IDs |

## Helper Functions

```go
// Check user type
if oauth2.IsGG4LStudent(user) { ... }
if oauth2.IsGG4LTeacher(user) { ... }
if oauth2.IsGG4LAdmin(user) { ... }
if oauth2.IsGG4LParent(user) { ... }
if oauth2.IsGG4LStaff(user) { ... }

// Get user type as enum
userType := oauth2.GetGG4LUserType(user)  // GG4LUserType enum

// Get user identifiers
gg4lID := oauth2.GetGG4LID(user)
sourceID := oauth2.GetGG4LSourceID(user)  // SIS ID
districtID := oauth2.GetGG4LDistrictID(user)
districtName := oauth2.GetGG4LDistrictName(user)

// Get school info
schoolIDs := oauth2.GetGG4LSchoolIDs(user)      // []string
schoolNames := oauth2.GetGG4LSchoolNames(user)  // []string

// Student-specific
gradeLevel := oauth2.GetGG4LGradeLevel(user)

// Parent-specific
studentIDs := oauth2.GetGG4LStudentIDs(user)    // []string of linked students

// Organizational units
orgUnitIDs := oauth2.GetGG4LOrgUnitIDs(user)    // []string
```

## API Client

For post-authentication API calls (rostering data), use `GG4LAPIClient`:

```go
func (h *Handler) TeacherClasses(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Create API client
    gg4lClient := oauth2.NewGG4LAPIClient(user.AccessToken)

    // Get teacher's classes
    classes, err := gg4lClient.GetTeacherClasses(r.Context(), oauth2.GetGG4LID(user))
    if err != nil {
        http.Error(w, "Failed to fetch classes", http.StatusInternalServerError)
        return
    }

    // Get enrollments for each class
    for _, class := range classes {
        enrollments, _ := gg4lClient.GetClassEnrollments(r.Context(), class.ID)
        // Display class roster...
    }
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetSchools(ctx)` | Get schools user has access to |
| `GetSchool(ctx, schoolID)` | Get specific school info |
| `GetDistrict(ctx, districtID)` | Get district info |
| `GetStudents(ctx, schoolID)` | Get students for a school |
| `GetTeachers(ctx, schoolID)` | Get teachers for a school |
| `GetClasses(ctx, schoolID)` | Get classes for a school |
| `GetClassEnrollments(ctx, classID)` | Get enrollments for a class |
| `GetStudentClasses(ctx, studentID)` | Get classes for a student |
| `GetTeacherClasses(ctx, teacherID)` | Get classes for a teacher |
| `GetOrgUnits(ctx, districtID)` | Get organizational units |
| `GetParentStudents(ctx, parentID)` | Get students linked to a parent |

## Parent Access Pattern

Parents can view data for their linked students:

```go
func (h *Handler) ParentDashboard(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    if !oauth2.IsGG4LParent(user) {
        http.Error(w, "Parent access required", http.StatusForbidden)
        return
    }

    gg4lClient := oauth2.NewGG4LAPIClient(user.AccessToken)

    // Get linked students
    students, err := gg4lClient.GetParentStudents(r.Context(), oauth2.GetGG4LID(user))
    if err != nil {
        http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
        return
    }

    // For each student, get their classes
    for _, student := range students {
        classes, _ := gg4lClient.GetStudentClasses(r.Context(), student.ID)
        // Display student info and classes...
    }
}
```

## Important Notes

- GG4L provides OneRoster-compliant rostering data
- Source IDs (`source_id`) match the original SIS identifiers
- Schools are retrieved based on user's district membership
- Users can belong to multiple schools
- Organizational units provide hierarchical structure (departments, grades, etc.)
- Parent accounts are linked to specific students via `student_ids`

---

[← Back to OAuth2 Providers](./README.md)
