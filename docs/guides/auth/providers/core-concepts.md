# Authentication Core Concepts

*Session stores, middleware, and common patterns for WAFFLE authentication.*

This document covers foundational concepts that apply to all authentication providers (OAuth2, OIDC, SAML, LTI).

---

## Overview

WAFFLE provides OAuth2 support through the `auth/oauth2` package. The authentication flow works the same way for all providers:

```
User clicks "Login with Provider"
        ↓
LoginHandler redirects to Provider
        ↓
User authenticates with Provider
        ↓
Provider redirects to CallbackHandler
        ↓
CallbackHandler exchanges code for tokens
        ↓
Fetches user info from Provider
        ↓
Creates session, sets cookie
        ↓
User is authenticated
```

---

## Quick Start: Google Authentication

Here's a complete example showing the core pattern used by all providers:

### 1. Create OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Go to **APIs & Services → Credentials**
4. Click **Create Credentials → OAuth client ID**
5. Select **Web application**
6. Add your redirect URI: `https://yourapp.com/auth/google/callback`
7. Save your **Client ID** and **Client Secret**

### 2. Add Configuration to AppConfig

```go
// internal/app/bootstrap/appconfig.go
package bootstrap

type AppConfig struct {
    Greeting string `conf:"greeting" conf-default:"Hello from WAFFLE!"`

    // OAuth2 settings
    GoogleClientID     string `conf:"google_client_id"`
    GoogleClientSecret string `conf:"google_client_secret"`
    GoogleRedirectURL  string `conf:"google_redirect_url"`
    BaseURL            string `conf:"base_url" conf-default:"http://localhost:8080"`
}
```

### 3. Set Environment Variables

```bash
export WAFFLE_GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export WAFFLE_GOOGLE_CLIENT_SECRET="your-client-secret"
export WAFFLE_GOOGLE_REDIRECT_URL="http://localhost:8080/auth/google/callback"
```

### 4. Wire Up in BuildHandler

```go
// internal/app/bootstrap/routes.go
package bootstrap

import (
    "net/http"

    "github.com/dalemusser/waffle/pantry/auth/oauth2"
    "github.com/dalemusser/waffle/config"
    "github.com/dalemusser/waffle/router"
    "github.com/go-chi/chi/v5"
    "go.uber.org/zap"
)

// Shared stores (in production, use Redis or MongoDB)
var (
    sessionStore = oauth2.NewMemorySessionStore()
    stateStore   = oauth2.NewMemoryStateStore()
)

func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Create Google OAuth provider
    googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
        ClientID:     appCfg.GoogleClientID,
        ClientSecret: appCfg.GoogleClientSecret,
        RedirectURL:  appCfg.GoogleRedirectURL,
        SessionStore: sessionStore,
        StateStore:   stateStore,
        OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
            // Redirect to dashboard after successful login
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        },
    }, logger)
    if err != nil {
        return nil, err
    }

    // Public routes
    r.Get("/", homeHandler)

    // Auth routes
    r.Get("/auth/google/login", googleAuth.LoginHandler())
    r.Get("/auth/google/callback", googleAuth.CallbackHandler())
    r.Get("/auth/google/logout", googleAuth.LogoutHandler())

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(googleAuth.RequireAuth("/auth/google/login"))
        r.Get("/dashboard", dashboardHandler)
        r.Get("/profile", profileHandler)
    })

    return r, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(`
        <h1>Welcome</h1>
        <a href="/auth/google/login">Login with Google</a>
    `))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(`
        <h1>Dashboard</h1>
        <p>Welcome, ` + user.Name + `!</p>
        <p>Email: ` + user.Email + `</p>
        <a href="/auth/google/logout">Logout</a>
    `))
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(`
        <h1>Profile</h1>
        <img src="` + user.Picture + `" width="100">
        <p>Name: ` + user.Name + `</p>
        <p>Email: ` + user.Email + `</p>
    `))
}
```

---

## Multiple Providers

You can support multiple providers simultaneously:

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := router.New(coreCfg, logger)

    // Shared session store (both providers use the same sessions)
    sessionStore := oauth2.NewMemorySessionStore()
    stateStore := oauth2.NewMemoryStateStore()

    // Google provider
    googleAuth, _ := oauth2.Google(oauth2.GoogleConfig{
        ClientID:     appCfg.GoogleClientID,
        ClientSecret: appCfg.GoogleClientSecret,
        RedirectURL:  appCfg.BaseURL + "/auth/google/callback",
        SessionStore: sessionStore,
        StateStore:   stateStore,
        CookieName:   "session", // Use same cookie name
    }, logger)

    // GitHub provider
    githubAuth, _ := oauth2.GitHub(oauth2.GitHubConfig{
        ClientID:     appCfg.GitHubClientID,
        ClientSecret: appCfg.GitHubClientSecret,
        RedirectURL:  appCfg.BaseURL + "/auth/github/callback",
        SessionStore: sessionStore,
        StateStore:   stateStore,
        CookieName:   "session", // Use same cookie name
    }, logger)

    // Login page with multiple options
    r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(`
            <h1>Login</h1>
            <a href="/auth/google/login">Login with Google</a><br>
            <a href="/auth/github/login">Login with GitHub</a>
        `))
    })

    // Auth routes
    r.Route("/auth/google", func(r chi.Router) {
        r.Get("/login", googleAuth.LoginHandler())
        r.Get("/callback", googleAuth.CallbackHandler())
        r.Get("/logout", googleAuth.LogoutHandler())
    })

    r.Route("/auth/github", func(r chi.Router) {
        r.Get("/login", githubAuth.LoginHandler())
        r.Get("/callback", githubAuth.CallbackHandler())
        r.Get("/logout", githubAuth.LogoutHandler())
    })

    // Protected routes (either provider works)
    r.Group(func(r chi.Router) {
        r.Use(googleAuth.RequireAuth("/login")) // Redirect to login page
        r.Get("/dashboard", dashboardHandler)
    })

    return r, nil
}
```

---

## JSON API Authentication

For APIs, use `RequireAuthJSON` instead of `RequireAuth`:

```go
// API routes that return JSON errors
r.Route("/api", func(r chi.Router) {
    r.Use(googleAuth.RequireAuthJSON())
    r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
        user := oauth2.UserFromContext(r.Context())
        json.NewEncoder(w).Encode(user)
    })
})
```

Unauthenticated requests receive:

```json
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error": "unauthorized"}
```

---

## Production Session Stores

For production deployments with multiple instances, use Redis or MongoDB instead of the in-memory store.

### Redis Session Store Example

```go
// internal/app/system/redisstore.go
package system

import (
    "context"
    "encoding/json"
    "time"

    "github.com/dalemusser/waffle/pantry/auth/oauth2"
    "github.com/redis/go-redis/v9"
)

type RedisSessionStore struct {
    client *redis.Client
    prefix string
}

func NewRedisSessionStore(client *redis.Client, prefix string) *RedisSessionStore {
    return &RedisSessionStore{client: client, prefix: prefix}
}

func (s *RedisSessionStore) Save(ctx context.Context, session *oauth2.Session) error {
    data, err := json.Marshal(session)
    if err != nil {
        return err
    }
    ttl := time.Until(session.ExpiresAt)
    return s.client.Set(ctx, s.prefix+session.ID, data, ttl).Err()
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*oauth2.Session, error) {
    data, err := s.client.Get(ctx, s.prefix+sessionID).Bytes()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    var session oauth2.Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }
    return &session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
    return s.client.Del(ctx, s.prefix+sessionID).Err()
}

// RedisStateStore for OAuth2 state
type RedisStateStore struct {
    client *redis.Client
    prefix string
}

func NewRedisStateStore(client *redis.Client, prefix string) *RedisStateStore {
    return &RedisStateStore{client: client, prefix: prefix}
}

func (s *RedisStateStore) Save(ctx context.Context, state string, expiresAt time.Time) error {
    ttl := time.Until(expiresAt)
    return s.client.Set(ctx, s.prefix+state, "1", ttl).Err()
}

func (s *RedisStateStore) Validate(ctx context.Context, state string) (bool, error) {
    result, err := s.client.GetDel(ctx, s.prefix+state).Result()
    if err == redis.Nil {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return result == "1", nil
}
```

### Using Redis Stores in BuildHandler

```go
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
    // Use Redis stores from DBDeps
    sessionStore := system.NewRedisSessionStore(deps.RedisClient, "session:")
    stateStore := system.NewRedisStateStore(deps.RedisClient, "oauth_state:")

    googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
        ClientID:     appCfg.GoogleClientID,
        ClientSecret: appCfg.GoogleClientSecret,
        RedirectURL:  appCfg.GoogleRedirectURL,
        SessionStore: sessionStore,
        StateStore:   stateStore,
    }, logger)
    // ...
}
```

---

## User Information

The `oauth2.User` struct contains:

| Field | Description |
|-------|-------------|
| `ID` | Provider-specific user ID |
| `Email` | User's email address |
| `EmailVerified` | Whether email is verified by provider |
| `Name` | Display name |
| `Picture` | Profile picture URL |
| `Provider` | OAuth provider name ("google", "github") |
| `Raw` | Raw claims from the provider |
| `Extra` | Provider-specific extras (e.g., GitHub login) |

Access the user in handlers:

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    if user == nil {
        http.Error(w, "not authenticated", http.StatusUnauthorized)
        return
    }

    // Use user info
    fmt.Fprintf(w, "Hello, %s!", user.Name)

    // Access raw claims
    if locale, ok := user.Raw["locale"].(string); ok {
        fmt.Fprintf(w, "Locale: %s", locale)
    }

    // GitHub-specific: get username
    if user.Provider == "github" {
        fmt.Fprintf(w, "GitHub: @%s", user.Extra["login"])
    }
}
```

---

## Custom Success/Error Handlers

```go
googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
    // ... other config

    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        // Check if user exists in your database
        dbUser, err := deps.UserStore.FindByEmail(r.Context(), user.Email)
        if err != nil {
            // Create new user
            dbUser, _ = deps.UserStore.Create(r.Context(), user.Email, user.Name)
        }

        // Store user ID in session or redirect with user info
        http.Redirect(w, r, "/dashboard?welcome=true", http.StatusTemporaryRedirect)
    },

    OnError: func(w http.ResponseWriter, r *http.Request, err error) {
        logger.Error("OAuth2 authentication failed", zap.Error(err))
        http.Redirect(w, r, "/login?error=auth_failed", http.StatusTemporaryRedirect)
    },
}, logger)
```

---

## Security Considerations

1. **Always use HTTPS in production** — Set `CookieSecure: true`
2. **Validate email verification** — Check `user.EmailVerified` before trusting email
3. **Use distributed session stores** — Don't use `MemorySessionStore` with multiple instances
4. **Rotate client secrets** — Periodically rotate OAuth2 client secrets
5. **Limit scopes** — Only request the scopes you need

---

## Provider Methods

All OAuth2 providers share these methods:

| Method | Description |
|--------|-------------|
| `LoginHandler()` | Handler that redirects to OAuth provider |
| `CallbackHandler()` | Handler that processes OAuth callback |
| `LogoutHandler()` | Handler that ends user session |
| `RequireAuth(loginURL)` | Middleware for HTML routes |
| `RequireAuthJSON()` | Middleware for API routes |
| `GetSession(r)` | Get session from request |

---

## Core Types

| Type | Description |
|------|-------------|
| `oauth2.User` | Authenticated user information |
| `oauth2.Session` | User session with expiration |
| `oauth2.Provider` | OAuth2 provider with handlers |
| `oauth2.SessionStore` | Interface for session storage |
| `oauth2.StateStore` | Interface for state storage |

---

## Core Functions

| Function | Description |
|----------|-------------|
| `oauth2.NewMemorySessionStore()` | Create in-memory session store |
| `oauth2.NewMemoryStateStore()` | Create in-memory state store |
| `oauth2.UserFromContext(ctx)` | Get user from request context |

---

## See Also

- [Authentication Providers Index](./README.md) — All available providers
- [Routes & Middleware Guide](../../../core/routing.md) — Middleware patterns
- [DBDeps Redis Example](../../databases/redis.md) — Redis integration for sessions
