# ClassLink Authentication

*K-12 SSO provider with OIDC and role-based access.*

ClassLink is a popular SSO provider for K-12 schools. It uses OIDC (OpenID Connect) and provides role-based user information.

---

## 1. Create ClassLink Application

1. Go to [ClassLink Developer Portal](https://developer.classlink.com/)
2. Register your application
3. Set your **Redirect URI**: `https://school.app/auth/classlink/callback`
4. Save your **Client ID** and **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    ClassLinkClientID     string `conf:"classlink_client_id"`
    ClassLinkClientSecret string `conf:"classlink_client_secret"`
    ClassLinkRedirectURL  string `conf:"classlink_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create ClassLink OAuth provider
classLinkAuth, err := oauth2.ClassLink(oauth2.ClassLinkConfig{
    ClientID:     appCfg.ClassLinkClientID,
    ClientSecret: appCfg.ClassLinkClientSecret,
    RedirectURL:  appCfg.ClassLinkRedirectURL,
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Route based on role
        if oauth2.IsClassLinkStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsClassLinkTeacher(user) {
            http.Redirect(w, r, "/teacher/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// ClassLink auth routes
r.Get("/auth/classlink/login", classLinkAuth.LoginHandler())
r.Get("/auth/classlink/callback", classLinkAuth.CallbackHandler())
r.Get("/auth/classlink/logout", classLinkAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains ClassLink-specific fields:

| Field | Description |
|-------|-------------|
| `role` | `student`, `teacher`, `administrator`, `parent`, `aide`, or `other` |
| `tenant_id` | ClassLink tenant (district) ID |
| `sourced_id` | OneRoster SourcedId (SIS ID) |
| `login_id` | User's login identifier |
| `building_id` | School/building ID |
| `first_name` | User's first name |
| `last_name` | User's last name |

## Helper Functions

```go
// Check user role
if oauth2.IsClassLinkStudent(user) { ... }
if oauth2.IsClassLinkTeacher(user) { ... }
if oauth2.IsClassLinkAdmin(user) { ... }

// Get role
role := oauth2.GetClassLinkRole(user)  // ClassLinkRole enum
```

## Package Reference

### Types

| Type | Description |
|------|-------------|
| `oauth2.ClassLinkConfig` | ClassLink-specific configuration |
| `oauth2.ClassLinkRole` | Role enum (student, teacher, administrator, parent, aide, other) |

### Functions

| Function | Description |
|----------|-------------|
| `oauth2.ClassLink(cfg, logger)` | Create ClassLink OAuth2 provider |
| `oauth2.IsClassLinkStudent(user)` | Check if user is a student |
| `oauth2.IsClassLinkTeacher(user)` | Check if user is a teacher |
| `oauth2.IsClassLinkAdmin(user)` | Check if user is an admin |
| `oauth2.GetClassLinkRole(user)` | Get role enum |

---

[← Back to OAuth2 Providers](./README.md)
