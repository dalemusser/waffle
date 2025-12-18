# Microsoft Education Authentication

*Azure AD with EDU tenant and education-specific APIs.*

Microsoft Education provides Azure AD authentication with education-specific APIs and attributes for K-12 and higher education institutions using Microsoft 365 Education.

---

## 1. Register Azure AD Application

1. Go to [Azure Portal](https://portal.azure.com/) with an **Education tenant admin** account
2. Navigate to **Azure Active Directory** → **App registrations** → **New registration**
3. Configure:
   - **Name**: Your application name
   - **Supported account types**: Single tenant (your edu tenant)
   - **Redirect URI**: `https://school.app/auth/microsoft-edu/callback`
4. Note your **Application (client) ID** and **Directory (tenant) ID**
5. Go to **Certificates & secrets** → **New client secret**
6. Under **API permissions**, add Microsoft Graph permissions for education

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    MicrosoftEduClientID     string `conf:"microsoft_edu_client_id"`
    MicrosoftEduClientSecret string `conf:"microsoft_edu_client_secret"`
    MicrosoftEduRedirectURL  string `conf:"microsoft_edu_redirect_url"`
    MicrosoftEduTenantID     string `conf:"microsoft_edu_tenant_id"` // Your edu tenant ID
}
```

## 3. Wire Up in BuildHandler

```go
// Create Microsoft Education OAuth provider
msEduAuth, err := oauth2.MicrosoftEducation(oauth2.MicrosoftEducationConfig{
    ClientID:     appCfg.MicrosoftEduClientID,
    ClientSecret: appCfg.MicrosoftEduClientSecret,
    RedirectURL:  appCfg.MicrosoftEduRedirectURL,
    TenantID:     appCfg.MicrosoftEduTenantID,
    Scopes:       []string{"User.Read", "EduRoster.Read", "EduAssignments.Read"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsMicrosoftEduTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsMicrosoftEduStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Microsoft Education auth routes
r.Get("/auth/microsoft-edu/login", msEduAuth.LoginHandler())
r.Get("/auth/microsoft-edu/callback", msEduAuth.CallbackHandler())
r.Get("/auth/microsoft-edu/logout", msEduAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Microsoft Education-specific fields:

| Field | Description |
|-------|-------------|
| `oid` | Object ID |
| `tid` | Tenant ID |
| `upn` | User Principal Name |
| `primary_role` | Primary education role |
| `external_source` | SIS source |
| `external_id` | SIS user ID |
| `student_id` | Student ID (students) |
| `grade` | Grade level (students) |
| `school_ids` | Comma-separated school IDs |
| `teacher_number` | Teacher number (teachers) |
| `department` | Department |
| `given_name` | First name |
| `family_name` | Last name |

## Helper Functions

```go
// Check education roles
if oauth2.IsMicrosoftEduStudent(user) { ... }
if oauth2.IsMicrosoftEduTeacher(user) { ... }
if oauth2.IsMicrosoftEduFaculty(user) { ... }
if oauth2.IsMicrosoftEduStaff(user) { ... }
if oauth2.IsMicrosoftEduAdmin(user) { ... }

// Get user identifiers
objectID := oauth2.GetMicrosoftEduObjectID(user)
externalID := oauth2.GetMicrosoftEduExternalID(user)  // SIS ID
studentID := oauth2.GetMicrosoftEduStudentID(user)
teacherNumber := oauth2.GetMicrosoftEduTeacherNumber(user)

// Get school info
schoolIDs := oauth2.GetMicrosoftEduSchoolIDs(user)  // []string
grade := oauth2.GetMicrosoftEduGrade(user)
```

## API Client

For Microsoft Graph Education API calls:

```go
func (h *Handler) StudentClasses(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    eduClient := oauth2.NewMicrosoftEduAPIClient(user.AccessToken)

    classes, err := eduClient.GetMyClasses(r.Context())
    if err != nil {
        http.Error(w, "Failed to fetch classes", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(classes)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetMe(ctx)` | Get current education user |
| `GetMyClasses(ctx)` | Get user's classes |
| `GetMySchools(ctx)` | Get user's schools |
| `GetClass(ctx, classID)` | Get class details |
| `GetClassMembers(ctx, classID)` | Get class roster |
| `GetClassTeachers(ctx, classID)` | Get class teachers |
| `GetClassAssignments(ctx, classID)` | Get class assignments |
| `GetSchool(ctx, schoolID)` | Get school details |
| `GetSchoolClasses(ctx, schoolID)` | Get school's classes |
| `GetSchoolUsers(ctx, schoolID)` | Get school's users |

## Education API Scopes

| Scope | Description |
|-------|-------------|
| `EduRoster.Read` | Read education roster |
| `EduRoster.ReadBasic` | Read basic roster info |
| `EduAssignments.Read` | Read assignments |
| `EduAssignments.ReadWrite` | Read/write assignments |
| `EduAdministration.Read` | Read admin settings |

## SIS Integration

Microsoft School Data Sync (SDS) syncs data from SIS systems:

```go
// Access SDS-synced data
externalID := user.Extra["external_id"]     // SIS user ID
externalSource := user.Extra["external_source"]  // SIS name

// The Microsoft Graph Education APIs use SDS-synced data
```

## Important Notes

- Requires Microsoft 365 Education tenant
- SIS data synced via School Data Sync (SDS)
- Education-specific APIs require admin consent
- Grade levels follow local education standards
- Schools and classes are synced from SIS
- Assignments API integrates with Microsoft Teams

---

[← Back to OAuth2 Providers](./README.md)
