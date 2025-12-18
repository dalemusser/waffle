# LinkedIn Authentication

*Professional network with profile access.*

LinkedIn OAuth 2.0 provides authentication for LinkedIn users with access to professional profile data. Useful for professional networking apps, recruitment tools, and B2B applications.

---

## 1. Create LinkedIn Application

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/)
2. Click **Create app**
3. Fill in application details:
   - **App name**: Your application name
   - **LinkedIn Page**: Associate with a company page
   - **Privacy policy URL**: Your privacy policy
   - **App logo**: Upload logo
4. Under **Auth** tab, add **Authorized redirect URL**: `https://yourapp.com/auth/linkedin/callback`
5. Note your **Client ID** and **Client Secret**
6. Under **Products**, request access to **Sign In with LinkedIn using OpenID Connect**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    LinkedInClientID     string `conf:"linkedin_client_id"`
    LinkedInClientSecret string `conf:"linkedin_client_secret"`
    LinkedInRedirectURL  string `conf:"linkedin_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create LinkedIn OAuth provider
linkedinAuth, err := oauth2.LinkedIn(oauth2.LinkedInConfig{
    ClientID:     appCfg.LinkedInClientID,
    ClientSecret: appCfg.LinkedInClientSecret,
    RedirectURL:  appCfg.LinkedInRedirectURL,
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

// LinkedIn auth routes
r.Get("/auth/linkedin/login", linkedinAuth.LoginHandler())
r.Get("/auth/linkedin/callback", linkedinAuth.CallbackHandler())
r.Get("/auth/linkedin/logout", linkedinAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains LinkedIn-specific fields:

| Field | Description |
|-------|-------------|
| `sub` | LinkedIn member ID |
| `given_name` | First name |
| `family_name` | Last name |
| `picture` | Profile picture URL |
| `locale` | User's locale |
| `email_verified` | Whether email is verified |

## Helper Functions

```go
// Get user identifiers
linkedinID := oauth2.GetLinkedInUserID(user)
picture := oauth2.GetLinkedInPicture(user)

// Check email verification
if oauth2.IsLinkedInEmailVerified(user) { ... }
```

## API Client

For LinkedIn API calls:

```go
func (h *Handler) UserProfile(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    linkedinClient := oauth2.NewLinkedInAPIClient(user.AccessToken)

    profile, err := linkedinClient.GetProfile(r.Context())
    if err != nil {
        http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(profile)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetProfile(ctx)` | Get user profile (OpenID Connect) |
| `GetUserInfo(ctx)` | Get user info from userinfo endpoint |

## Common Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OpenID Connect authentication |
| `profile` | Name and profile picture |
| `email` | Email address |
| `w_member_social` | Post on behalf of user (Marketing API) |

## LinkedIn API Versions

LinkedIn has multiple API versions:

| API | Access | Description |
|-----|--------|-------------|
| Sign In with LinkedIn (OIDC) | Default | Basic auth and profile |
| Marketing API | Requires approval | Company pages, ads |
| Consumer Solutions | Limited | Advanced profile data |

## Important Notes

- LinkedIn deprecated v1 APIs; use OpenID Connect
- Profile data is limited to basic info (name, email, picture)
- Additional data requires Marketing API approval
- Access tokens expire in 60 days
- LinkedIn requires app verification for production
- Company page association is required for app creation

---

[← Back to OAuth2 Providers](./README.md)
