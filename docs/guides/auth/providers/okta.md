# Okta Authentication

*Enterprise identity management with SSO.*

Okta is a leading enterprise identity and access management platform. It provides OAuth 2.0/OIDC authentication with robust security features, user management, and SSO capabilities.

---

## 1. Create Okta Application

1. Log in to your [Okta Admin Console](https://admin.okta.com/)
2. Go to **Applications** → **Create App Integration**
3. Select **OIDC - OpenID Connect** and **Web Application**
4. Configure:
   - **App name**: Your application name
   - **Sign-in redirect URI**: `https://yourapp.com/auth/okta/callback`
   - **Sign-out redirect URI**: `https://yourapp.com/auth/okta/logout/callback`
   - **Controlled access**: Select who can access
5. Note your **Client ID** and **Client Secret**
6. Note your **Okta domain** (e.g., `dev-12345.okta.com`)

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    OktaClientID     string `conf:"okta_client_id"`
    OktaClientSecret string `conf:"okta_client_secret"`
    OktaRedirectURL  string `conf:"okta_redirect_url"`
    OktaDomain       string `conf:"okta_domain"` // e.g., "dev-12345.okta.com"
    OktaAuthServerID string `conf:"okta_auth_server_id"` // Optional: custom auth server
}
```

## 3. Wire Up in BuildHandler

```go
// Create Okta OAuth provider
oktaAuth, err := oauth2.Okta(oauth2.OktaConfig{
    ClientID:     appCfg.OktaClientID,
    ClientSecret: appCfg.OktaClientSecret,
    RedirectURL:  appCfg.OktaRedirectURL,
    Domain:       appCfg.OktaDomain,
    AuthServerID: appCfg.OktaAuthServerID, // Optional: "default" or custom
    Scopes:       []string{"openid", "profile", "email", "groups"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.HasOktaGroup(user, "Administrators") {
            http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// Okta auth routes
r.Get("/auth/okta/login", oktaAuth.LoginHandler())
r.Get("/auth/okta/callback", oktaAuth.CallbackHandler())
r.Get("/auth/okta/logout", oktaAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Okta-specific fields:

| Field | Description |
|-------|-------------|
| `sub` | Okta user ID |
| `preferred_username` | Username/login |
| `given_name` | First name |
| `family_name` | Last name |
| `zoneinfo` | Timezone |
| `locale` | Locale |
| `groups` | Comma-separated group names |
| `department` | Department |
| `title` | Job title |
| `employee_number` | Employee number |
| `manager` | Manager |
| `organization` | Organization |

## Helper Functions

```go
// Get user identifiers
oktaID := oauth2.GetOktaUserID(user)
username := oauth2.GetOktaUsername(user)
employeeNumber := oauth2.GetOktaEmployeeNumber(user)

// Get groups
groups := oauth2.GetOktaGroups(user)  // []string

// Check group membership
if oauth2.HasOktaGroup(user, "Administrators") { ... }
if oauth2.HasOktaGroup(user, "Developers") { ... }

// Check for any of multiple groups
if oauth2.HasAnyOktaGroup(user, []string{"Admins", "Managers"}) { ... }
```

## API Client

For Okta Management API calls:

```go
func (h *Handler) UserGroups(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    // Note: Management API uses a separate API token, not user tokens
    oktaClient := oauth2.NewOktaAPIClient(h.Config.OktaDomain, h.Config.OktaAPIToken)

    userID := oauth2.GetOktaUserID(user)
    groups, err := oktaClient.GetUserGroups(r.Context(), userID)
    if err != nil {
        http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(groups)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetUser(ctx, userID)` | Get user details |
| `GetUserGroups(ctx, userID)` | Get user's groups |
| `GetUserAppLinks(ctx, userID)` | Get user's app links |
| `GetGroup(ctx, groupID)` | Get group details |
| `GetGroupMembers(ctx, groupID)` | Get group members |
| `ListUsers(ctx, query)` | Search users |
| `ListGroups(ctx, query)` | Search groups |

## Authorization Server Options

| Option | Description |
|--------|-------------|
| Org Authorization Server | Default, uses `https://{domain}/oauth2` |
| Custom Authorization Server | Use `https://{domain}/oauth2/{authServerID}` |
| `default` | The "default" custom auth server |

## Common Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OpenID Connect authentication |
| `profile` | User profile info |
| `email` | Email address |
| `groups` | User's group memberships |
| `offline_access` | Refresh token support |

## MFA Configuration

Okta can enforce MFA. Handle in your app:

```go
// Okta handles MFA before redirecting back to your app
// No additional code needed - MFA is transparent to your app
// Configure MFA policies in the Okta Admin Console
```

## Important Notes

- Okta domain format: `{org}.okta.com` or `{org}.oktapreview.com`
- Custom authorization servers support custom claims
- Group claims require the `groups` scope
- API tokens (for Management API) are separate from OAuth tokens
- Okta provides robust audit logging
- Session management can be configured per application

---

[← Back to OAuth2 Providers](./README.md)
