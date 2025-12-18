# Microsoft Authentication

*Azure AD/Entra ID for enterprise and Microsoft 365.*

Microsoft Identity Platform (Azure AD / Microsoft Entra ID) provides OAuth 2.0 and OpenID Connect authentication for Microsoft accounts, Azure AD tenants, and Microsoft 365 organizations.

---

## 1. Register Azure AD Application

1. Go to [Azure Portal](https://portal.azure.com/) → **Azure Active Directory** (or **Microsoft Entra ID**)
2. Navigate to **App registrations** → **New registration**
3. Configure:
   - **Name**: Your application name
   - **Supported account types**: Choose based on your needs
   - **Redirect URI**: `https://yourapp.com/auth/microsoft/callback` (Web)
4. Note your **Application (client) ID** and **Directory (tenant) ID**
5. Go to **Certificates & secrets** → **New client secret**
6. Note your **Client Secret** value

### Account Types

| Option | Description |
|--------|-------------|
| Single tenant | Only your organization |
| Multitenant | Any Azure AD organization |
| Multitenant + personal | Azure AD + personal Microsoft accounts |
| Personal only | Consumer Microsoft accounts only |

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    MicrosoftClientID     string `conf:"microsoft_client_id"`
    MicrosoftClientSecret string `conf:"microsoft_client_secret"`
    MicrosoftRedirectURL  string `conf:"microsoft_redirect_url"`
    MicrosoftTenantID     string `conf:"microsoft_tenant_id"` // "common", "organizations", "consumers", or specific tenant
}
```

## 3. Wire Up in BuildHandler

```go
// Create Microsoft OAuth provider
microsoftAuth, err := oauth2.Microsoft(oauth2.MicrosoftConfig{
    ClientID:     appCfg.MicrosoftClientID,
    ClientSecret: appCfg.MicrosoftClientSecret,
    RedirectURL:  appCfg.MicrosoftRedirectURL,
    TenantID:     appCfg.MicrosoftTenantID,
    Scopes:       []string{"User.Read", "profile", "email", "openid"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// Microsoft auth routes
r.Get("/auth/microsoft/login", microsoftAuth.LoginHandler())
r.Get("/auth/microsoft/callback", microsoftAuth.CallbackHandler())
r.Get("/auth/microsoft/logout", microsoftAuth.LogoutHandler())
```

## Tenant Options

| Tenant Value | Description |
|--------------|-------------|
| `common` | Any Azure AD tenant + personal Microsoft accounts |
| `organizations` | Any Azure AD tenant (no personal accounts) |
| `consumers` | Personal Microsoft accounts only |
| `{tenant-id}` | Specific Azure AD tenant only |

## User Information

The `User.Extra` map contains Microsoft-specific fields:

| Field | Description |
|-------|-------------|
| `oid` | Object ID (unique identifier) |
| `tid` | Tenant ID |
| `preferred_username` | Preferred username/UPN |
| `upn` | User Principal Name |
| `given_name` | First name |
| `family_name` | Last name |
| `job_title` | Job title |
| `department` | Department |
| `office_location` | Office location |
| `mobile_phone` | Mobile phone number |
| `business_phones` | Business phone numbers |

## Helper Functions

```go
// Get user identifiers
objectID := oauth2.GetMicrosoftObjectID(user)
tenantID := oauth2.GetMicrosoftTenantID(user)
upn := oauth2.GetMicrosoftUPN(user)

// Check if from specific tenant
if oauth2.IsMicrosoftTenant(user, "your-tenant-id") { ... }

// Check if personal account
if oauth2.IsMicrosoftPersonalAccount(user) { ... }
```

## API Client

For Microsoft Graph API calls:

```go
func (h *Handler) UserProfile(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    graphClient := oauth2.NewMicrosoftGraphClient(user.AccessToken)

    profile, err := graphClient.GetMe(r.Context())
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
| `GetMe(ctx)` | Get current user profile |
| `GetPhoto(ctx)` | Get user's profile photo |
| `GetManager(ctx)` | Get user's manager |
| `GetDirectReports(ctx)` | Get user's direct reports |
| `GetGroups(ctx)` | Get user's groups |
| `GetCalendar(ctx)` | Get user's calendar |
| `GetMail(ctx)` | Get user's mail |
| `GetDrive(ctx)` | Get user's OneDrive |

## Common Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OpenID Connect sign-in |
| `profile` | Basic profile info |
| `email` | Email address |
| `User.Read` | Read user profile |
| `User.ReadBasic.All` | Read basic profiles of all users |
| `Group.Read.All` | Read all groups |
| `Calendars.Read` | Read user calendars |
| `Mail.Read` | Read user mail |
| `Files.Read` | Read user files (OneDrive) |

## Important Notes

- Azure AD uses tenant-based isolation
- Personal Microsoft accounts use a different user ID format
- Access tokens expire in 1 hour; use refresh tokens
- Admin consent may be required for some scopes
- Microsoft Graph is the primary API for user data
- Token caching is recommended for production

---

[← Back to OAuth2 Providers](./README.md)
