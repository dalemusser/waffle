# Google Authentication

*Google accounts and Google Workspace.*

Google OAuth 2.0 provides authentication for consumer Google accounts and Google Workspace (formerly G Suite) organizations. It supports OpenID Connect for identity verification.

---

## 1. Create Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Go to **APIs & Services** → **Credentials**
4. Click **Create Credentials** → **OAuth client ID**
5. Configure the OAuth consent screen if prompted
6. Select **Web application**
7. Add **Authorized redirect URI**: `https://yourapp.com/auth/google/callback`
8. Note your **Client ID** and **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    GoogleClientID     string `conf:"google_client_id"`
    GoogleClientSecret string `conf:"google_client_secret"`
    GoogleRedirectURL  string `conf:"google_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Google OAuth provider
googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
    ClientID:     appCfg.GoogleClientID,
    ClientSecret: appCfg.GoogleClientSecret,
    RedirectURL:  appCfg.GoogleRedirectURL,
    Scopes:       []string{"openid", "profile", "email"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// Google auth routes
r.Get("/auth/google/login", googleAuth.LoginHandler())
r.Get("/auth/google/callback", googleAuth.CallbackHandler())
r.Get("/auth/google/logout", googleAuth.LogoutHandler())
```

## User Information

The `User.Raw` map contains Google-specific fields:

| Field | Description |
|-------|-------------|
| `id` | Google user ID |
| `email` | User's email |
| `verified_email` | Whether email is verified |
| `name` | Full name |
| `given_name` | First name |
| `family_name` | Last name |
| `picture` | Profile picture URL |
| `locale` | User's locale |

## Accessing User Data

```go
user := oauth2.UserFromContext(r.Context())

// Standard fields available on all providers
googleID := user.ID
email := user.Email
name := user.Name
picture := user.Picture
emailVerified := user.EmailVerified

// Google-specific fields from User.Raw
if givenName, ok := user.Raw["given_name"].(string); ok {
    // Use given name
}
if familyName, ok := user.Raw["family_name"].(string); ok {
    // Use family name
}
if locale, ok := user.Raw["locale"].(string); ok {
    // Use locale
}
```

## Domain Restriction (Google Workspace)

For Google Workspace, restrict authentication to your domain:

```go
googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
    // ... other config ...
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Check domain from email
        parts := strings.Split(user.Email, "@")
        if len(parts) != 2 || parts[1] != "company.com" {
            http.Error(w, "Unauthorized domain", http.StatusForbidden)
            return
        }
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
```

## Common Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OpenID Connect sign-in |
| `profile` | Basic profile info |
| `email` | Email address |
| `https://www.googleapis.com/auth/drive.readonly` | Read Google Drive |
| `https://www.googleapis.com/auth/calendar.readonly` | Read Google Calendar |
| `https://www.googleapis.com/auth/gmail.readonly` | Read Gmail |

## OAuth Consent Screen

Configure in Google Cloud Console:

1. **User type**: Internal (Workspace only) or External
2. **App information**: Name, logo, support email
3. **Scopes**: Add required OAuth scopes
4. **Test users**: Add during development

## Important Notes

- Google uses `hd` claim to indicate Workspace domain
- Consumer Google accounts don't have `hd` claim
- `email_verified` should be checked for security
- Google requires OAuth consent screen configuration
- Published apps need verification for sensitive scopes
- Access tokens expire in 1 hour; use refresh tokens

---

[← Back to OAuth2 Providers](./README.md)
