# Apple Authentication

*Sign in with Apple for iOS and web.*

Sign in with Apple provides OAuth 2.0/OpenID Connect authentication for Apple ID users. Required for iOS apps that offer third-party sign-in, and available for web applications.

---

## 1. Configure Apple Developer Account

1. Go to [Apple Developer Portal](https://developer.apple.com/)
2. Navigate to **Certificates, Identifiers & Profiles**
3. Under **Identifiers**, create:
   - **App ID** with Sign in with Apple capability
   - **Services ID** for web authentication

### Create Services ID:
1. Click **+** → **Services IDs**
2. Enter description and identifier
3. Enable **Sign in with Apple**
4. Configure:
   - **Domains**: `yourapp.com`
   - **Return URLs**: `https://yourapp.com/auth/apple/callback`
5. Note your **Services ID** (Client ID)

### Create Private Key:
1. Go to **Keys** → **+**
2. Enable **Sign in with Apple**
3. Configure with your App ID
4. Download the `.p8` key file
5. Note your **Key ID** and **Team ID**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    AppleClientID    string `conf:"apple_client_id"`    // Services ID
    AppleTeamID      string `conf:"apple_team_id"`
    AppleKeyID       string `conf:"apple_key_id"`
    ApplePrivateKey  string `conf:"apple_private_key"`  // Contents of .p8 file
    AppleRedirectURL string `conf:"apple_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Apple OAuth provider
appleAuth, err := oauth2.Apple(oauth2.AppleConfig{
    ClientID:     appCfg.AppleClientID,
    TeamID:       appCfg.AppleTeamID,
    KeyID:        appCfg.AppleKeyID,
    PrivateKey:   appCfg.ApplePrivateKey,
    RedirectURL:  appCfg.AppleRedirectURL,
    Scopes:       []string{"name", "email"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// Apple auth routes
r.Get("/auth/apple/login", appleAuth.LoginHandler())
r.Post("/auth/apple/callback", appleAuth.CallbackHandler()) // Apple uses POST
r.Get("/auth/apple/logout", appleAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Apple-specific fields:

| Field | Description |
|-------|-------------|
| `sub` | Apple user ID (stable identifier) |
| `email` | Email address (if shared) |
| `email_verified` | Whether email is verified |
| `is_private_email` | Whether using private relay |
| `real_user_status` | User validation status |
| `given_name` | First name (first auth only) |
| `family_name` | Last name (first auth only) |

**Important**: Apple only provides name on the **first authentication**. Store it immediately!

## Helper Functions

```go
// Get user identifiers
appleID := oauth2.GetAppleUserID(user)

// Check email status
if oauth2.IsAppleEmailVerified(user) { ... }
if oauth2.IsApplePrivateEmail(user) { ... }

// Get real user status
// 0 = unsupported, 1 = unknown, 2 = likely real
status := oauth2.GetAppleRealUserStatus(user)
if oauth2.IsAppleLikelyRealUser(user) { ... }
```

## First Authentication Handling

```go
appleAuth, err := oauth2.Apple(oauth2.AppleConfig{
    // ... other config ...
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Check if this is first authentication (name available)
        if givenName, ok := user.Extra["given_name"]; ok && givenName != "" {
            // Store the name - it won't be provided again!
            storeUserName(user.ID, givenName, user.Extra["family_name"])
        }

        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
```

## Private Email Relay

Apple's private email relay:
- Format: `abc123@privaterelay.appleid.com`
- Forwards to user's real email
- Must configure your domain for outbound email

Configure relay:
1. Apple Developer Portal → **Services** → **Sign in with Apple for Email Communication**
2. Register email domains and addresses
3. Add SPF and DKIM records

## Common Scopes

| Scope | Description |
|-------|-------------|
| `name` | User's name (first auth only) |
| `email` | User's email address |

## Platform Differences

| Platform | Callback Method | Notes |
|----------|----------------|-------|
| Web | POST | Uses form post |
| iOS | SDK | Uses native AuthenticationServices |
| Android | Web | Uses web flow |

## Important Notes

- Apple requires HTTPS for all endpoints
- Client secret is generated dynamically from private key (JWT)
- Name is only provided on **first** authorization
- Apple user IDs are unique per developer team
- Private relay email requires email domain configuration
- App Store apps **must** offer Sign in with Apple if they offer other social logins
- Store the private key securely (never commit to version control)

---

[← Back to OAuth2 Providers](./README.md)
