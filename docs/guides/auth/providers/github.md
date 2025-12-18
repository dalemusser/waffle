# GitHub Authentication

*Developer accounts and organization access.*

GitHub OAuth provides authentication for GitHub users with access to repositories, organizations, and the GitHub API. Ideal for developer tools and integrations.

---

## 1. Create GitHub OAuth App

1. Go to [GitHub Settings](https://github.com/settings/developers) → **OAuth Apps**
2. Click **New OAuth App**
3. Configure:
   - **Application name**: Your app name
   - **Homepage URL**: Your application URL
   - **Authorization callback URL**: `https://yourapp.com/auth/github/callback`
4. Click **Register application**
5. Note your **Client ID**
6. Click **Generate a new client secret** and note the **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    GitHubClientID     string `conf:"github_client_id"`
    GitHubClientSecret string `conf:"github_client_secret"`
    GitHubRedirectURL  string `conf:"github_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create GitHub OAuth provider
githubAuth, err := oauth2.GitHub(oauth2.GitHubConfig{
    ClientID:     appCfg.GitHubClientID,
    ClientSecret: appCfg.GitHubClientSecret,
    RedirectURL:  appCfg.GitHubRedirectURL,
    Scopes:       []string{"read:user", "user:email"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// GitHub auth routes
r.Get("/auth/github/login", githubAuth.LoginHandler())
r.Get("/auth/github/callback", githubAuth.CallbackHandler())
r.Get("/auth/github/logout", githubAuth.LogoutHandler())
```

## User Information

The `User.Raw` map contains GitHub-specific fields:

| Field | Description |
|-------|-------------|
| `id` | GitHub user ID (int64) |
| `login` | GitHub username |
| `name` | Display name |
| `email` | Public email |
| `avatar_url` | Profile picture URL |
| `html_url` | Profile page URL |
| `bio` | Bio text |
| `company` | Company name |
| `location` | Location |

The `User.Extra` map contains:

| Field | Description |
|-------|-------------|
| `login` | GitHub username |
| `html_url` | Profile page URL |

## Accessing User Data

```go
user := oauth2.UserFromContext(r.Context())

// Standard fields available on all providers
githubID := user.ID              // GitHub user ID as string
email := user.Email              // Primary verified email
name := user.Name                // Display name
picture := user.Picture          // Avatar URL
emailVerified := user.EmailVerified

// GitHub-specific fields from User.Extra
username := user.Extra["login"]
profileURL := user.Extra["html_url"]

// More fields from User.Raw
if login, ok := user.Raw["login"].(string); ok {
    // Use GitHub username
}
if company, ok := user.Raw["company"].(string); ok {
    // Use company
}
if bio, ok := user.Raw["bio"].(string); ok {
    // Use bio
}
```

## Organization Membership

Check organization membership for access control using the GitHub API directly:

```go
githubAuth, err := oauth2.GitHub(oauth2.GitHubConfig{
    // ... other config ...
    Scopes: []string{"read:user", "user:email", "read:org"},
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Use user.AccessToken to make GitHub API calls
        // GET https://api.github.com/user/orgs with Authorization: Bearer <token>
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
```

## Common Scopes

| Scope | Description |
|-------|-------------|
| `(no scope)` | Public read-only access |
| `read:user` | Read user profile |
| `user:email` | Read user email addresses |
| `read:org` | Read org membership |
| `repo` | Full repo access |
| `repo:status` | Commit status access |
| `admin:org` | Full org access |

## GitHub Apps vs OAuth Apps

| Feature | OAuth App | GitHub App |
|---------|-----------|------------|
| User auth | ✓ | ✓ |
| Act as user | ✓ | ✓ (user token) |
| Act as app | ✗ | ✓ (installation token) |
| Webhooks | Limited | Full support |
| Granular permissions | ✗ | ✓ |

## Important Notes

- GitHub usernames can change; use `id` for persistent identity
- Email may be null if user has no public email; use `user:email` scope
- Primary email is marked in the emails list
- Organization membership requires `read:org` scope
- Rate limits apply (5000 requests/hour for authenticated users)
- Consider GitHub Apps for more advanced integrations

---

[← Back to OAuth2 Providers](./README.md)
