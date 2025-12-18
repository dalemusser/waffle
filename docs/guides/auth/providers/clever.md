# Clever Authentication

*K-12 SSO provider with student/teacher/admin roles and SIS integration.*

Clever is the leading SSO provider for K-12 schools. It provides user information including role (student, teacher, admin) and SIS (Student Information System) IDs.

---

## 1. Create Clever Application

1. Go to [Clever Developer Portal](https://dev.clever.com/)
2. Create a new application
3. Set your **Redirect URI**: `https://school.app/auth/clever/callback`
4. Save your **Client ID** and **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    CleverClientID     string `conf:"clever_client_id"`
    CleverClientSecret string `conf:"clever_client_secret"`
    CleverRedirectURL  string `conf:"clever_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Clever OAuth provider
cleverAuth, err := oauth2.Clever(oauth2.CleverConfig{
    ClientID:     appCfg.CleverClientID,
    ClientSecret: appCfg.CleverClientSecret,
    RedirectURL:  appCfg.CleverRedirectURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Route based on user type
        if oauth2.IsCleverStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsCleverTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Clever auth routes
r.Get("/auth/clever/login", cleverAuth.LoginHandler())
r.Get("/auth/clever/callback", cleverAuth.CallbackHandler())
r.Get("/auth/clever/logout", cleverAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Clever-specific fields:

| Field | Description |
|-------|-------------|
| `user_type` | `student`, `teacher`, `district_admin`, `school_admin`, or `contact` |
| `district_id` | Clever district ID |
| `sis_id` | Student Information System ID |
| `school_ids` | Comma-separated list of school IDs |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user role
if oauth2.IsCleverStudent(user) { ... }
if oauth2.IsCleverTeacher(user) { ... }
if oauth2.IsCleverAdmin(user) { ... }  // district or school admin

// Get user type
userType := oauth2.GetCleverUserType(user)  // CleverUserType enum
```

## Package Reference

### Types

| Type | Description |
|------|-------------|
| `oauth2.CleverConfig` | Clever-specific configuration |
| `oauth2.CleverUserType` | User type enum (student, teacher, district_admin, school_admin, contact) |

### Functions

| Function | Description |
|----------|-------------|
| `oauth2.Clever(cfg, logger)` | Create Clever OAuth2 provider |
| `oauth2.IsCleverStudent(user)` | Check if user is a student |
| `oauth2.IsCleverTeacher(user)` | Check if user is a teacher |
| `oauth2.IsCleverAdmin(user)` | Check if user is an admin |
| `oauth2.GetCleverUserType(user)` | Get user type enum |

---

[← Back to OAuth2 Providers](./README.md)
