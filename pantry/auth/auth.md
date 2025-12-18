# auth

Authentication support for WAFFLE applications, including OAuth2 providers and API key middleware.

## Overview

The `auth` package provides two authentication approaches:

- **OAuth2** (`auth/oauth2`) — Full OAuth2 flow with support for 20+ providers (Google, GitHub, Microsoft, Apple, education providers, etc.)
- **API Key** (`auth/apikey`) — Simple static key authentication for APIs and admin endpoints

Both approaches integrate cleanly with Chi middleware and support session/cookie management.

## Import

```go
import (
    "github.com/dalemusser/waffle/auth/oauth2"
    "github.com/dalemusser/waffle/auth/apikey"
)
```

---

## OAuth2

The OAuth2 package provides a complete authentication flow: login redirect, callback handling, session management, and middleware for protecting routes.

### Quick Start

```go
func buildHandler(core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) (http.Handler, error) {
    r := chi.NewRouter()

    // Create stores (use Redis in production for multi-instance)
    sessionStore := oauth2.NewMemorySessionStore()
    stateStore := oauth2.NewMemoryStateStore()

    // Configure Google OAuth2
    googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
        ClientID:     appCfg.GoogleClientID,
        ClientSecret: appCfg.GoogleClientSecret,
        RedirectURL:  "https://myapp.com/auth/google/callback",
        SessionStore: sessionStore,
        StateStore:   stateStore,
    }, logger)
    if err != nil {
        return nil, err
    }

    // Auth routes
    r.Get("/auth/google/login", googleAuth.LoginHandler())
    r.Get("/auth/google/callback", googleAuth.CallbackHandler())
    r.Get("/auth/google/logout", googleAuth.LogoutHandler())

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(googleAuth.RequireAuth("/auth/google/login"))
        r.Get("/dashboard", dashboardHandler)
    })

    return r, nil
}
```

### Supported Providers

**Location:** `oauth2/*.go`

| Provider | Function | File |
|----------|----------|------|
| Google | `oauth2.Google()` | `google.go` |
| GitHub | `oauth2.GitHub()` | `github.go` |
| Microsoft | `oauth2.Microsoft()` | `microsoft.go` |
| Apple | `oauth2.Apple()` | `apple.go` |
| Discord | `oauth2.Discord()` | `discord.go` |
| LinkedIn | `oauth2.LinkedIn()` | `linkedin.go` |
| Okta | `oauth2.Okta()` | `okta.go` |
| **Education** | | |
| Clever | `oauth2.Clever()` | `clever.go` |
| ClassLink | `oauth2.ClassLink()` | `classlink.go` |
| Google Classroom | `oauth2.GoogleClassroom()` | `google_classroom.go` |
| Canvas | `oauth2.Canvas()` | `canvas.go` |
| Schoology | `oauth2.Schoology()` | `schoology.go` |
| Blackboard | `oauth2.Blackboard()` | `blackboard.go` |
| Moodle | `oauth2.Moodle()` | `moodle.go` |
| PowerSchool | `oauth2.PowerSchool()` | `powerschool.go` |
| Infinite Campus | `oauth2.InfiniteCampus()` | `infinitecampus.go` |
| Skyward | `oauth2.Skyward()` | `skyward.go` |
| Ed-Fi | `oauth2.EdFi()` | `edfi.go` |
| GG4L | `oauth2.GG4L()` | `gg4l.go` |
| Shibboleth | `oauth2.Shibboleth()` | `shibboleth.go` |
| LTI | `oauth2.LTI()` | `lti.go` |
| **Enterprise** | | |
| Banner | `oauth2.Banner()` | `banner.go` |
| Workday | `oauth2.Workday()` | `workday.go` |

### Core Types

**Location:** `oauth2/oauth2.go`

#### User

```go
type User struct {
    ID            string            // Provider-specific user ID
    Email         string            // User's email address
    EmailVerified bool              // Whether email is verified
    Name          string            // Display name
    Picture       string            // Profile picture URL
    Provider      string            // Provider name ("google", "github", etc.)
    Raw           map[string]any    // Raw claims from provider
    AccessToken   string            // OAuth access token (not serialized)
    RefreshToken  string            // OAuth refresh token (not serialized)
    TokenExpiry   time.Time         // When access token expires
    Extra         map[string]string // App-specific metadata
}
```

#### Session

```go
type Session struct {
    ID        string    // Unique session ID
    User      User      // Authenticated user info
    CreatedAt time.Time // When session was created
    ExpiresAt time.Time // When session expires
}

func (s *Session) IsExpired() bool
```

#### Config

```go
type Config struct {
    ProviderName    string
    OAuth2Config    *oauth2.Config
    FetchUserInfo   UserInfoFetcher
    SessionStore    SessionStore
    StateStore      StateStore
    SessionDuration time.Duration  // Default: 24h
    StateDuration   time.Duration  // Default: 10m
    CookieName      string         // Default: "waffle_session"
    CookiePath      string         // Default: "/"
    CookieSecure    bool           // Default: true
    CookieSameSite  http.SameSite  // Default: Lax
    OnSuccess       func(w, r, user)
    OnError         func(w, r, err)
    Logger          *zap.Logger
}
```

### Provider Methods

**Location:** `oauth2/oauth2.go`

#### NewProvider

```go
func NewProvider(cfg *Config) (*Provider, error)
```

Creates a generic OAuth2 provider. Use the provider-specific constructors (e.g., `oauth2.Google()`) for convenience.

#### LoginHandler

```go
func (p *Provider) LoginHandler() http.HandlerFunc
```

Returns a handler that initiates the OAuth2 flow by redirecting to the provider.

#### CallbackHandler

```go
func (p *Provider) CallbackHandler() http.HandlerFunc
```

Returns a handler that processes the OAuth2 callback, exchanges the code for tokens, fetches user info, creates a session, and sets the session cookie.

#### LogoutHandler

```go
func (p *Provider) LogoutHandler() http.HandlerFunc
```

Returns a handler that ends the user session and clears the cookie.

#### RequireAuth

```go
func (p *Provider) RequireAuth(loginURL string) func(http.Handler) http.Handler
```

Returns middleware that requires authentication. Redirects to `loginURL` if not authenticated.

#### RequireAuthJSON

```go
func (p *Provider) RequireAuthJSON() func(http.Handler) http.Handler
```

Returns middleware that requires authentication. Returns 401 JSON response if not authenticated (for APIs).

#### GetSession

```go
func (p *Provider) GetSession(r *http.Request) (*Session, error)
```

Retrieves the current session from the request cookie.

### Context Helpers

**Location:** `oauth2/oauth2.go`

```go
func ContextWithUser(ctx context.Context, user *User) context.Context
func UserFromContext(ctx context.Context) *User
```

Store and retrieve the authenticated user from request context.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())
    if user != nil {
        fmt.Fprintf(w, "Hello, %s!", user.Name)
    }
}
```

### Session Stores

**Location:** `oauth2/store.go`

#### SessionStore Interface

```go
type SessionStore interface {
    Save(ctx context.Context, session *Session) error
    Get(ctx context.Context, sessionID string) (*Session, error)
    Delete(ctx context.Context, sessionID string) error
}
```

#### StateStore Interface

```go
type StateStore interface {
    Save(ctx context.Context, state string, expiresAt time.Time) error
    Validate(ctx context.Context, state string) (bool, error)
}
```

#### Memory Stores (Development)

```go
sessionStore := oauth2.NewMemorySessionStore()
stateStore := oauth2.NewMemoryStateStore()

// Optional: start cleanup goroutines
stopSessionCleanup := sessionStore.StartCleanupTask(5 * time.Minute)
stopStateCleanup := stateStore.StartCleanupTask(1 * time.Minute)
defer stopSessionCleanup()
defer stopStateCleanup()
```

For production with multiple instances, implement `SessionStore` and `StateStore` using Redis, MongoDB, or another distributed store.

---

## API Key

The API key package provides simple static key authentication for protecting APIs and admin endpoints.

### Quick Start

**Location:** `apikey/apikey.go`

```go
import "github.com/dalemusser/waffle/auth/apikey"

func buildHandler(...) (http.Handler, error) {
    r := chi.NewRouter()

    // Protect admin routes with API key
    r.Group(func(r chi.Router) {
        r.Use(apikey.Require(appCfg.AdminAPIKey, apikey.Options{
            Realm: "admin",
        }, logger))
        r.Mount("/admin", adminRoutes())
    })

    return r, nil
}
```

### Require

```go
func Require(expected string, opts Options, logger *zap.Logger) func(http.Handler) http.Handler
```

Returns middleware that requires a valid API key. Key lookup order:

1. `Authorization: Bearer <token>`
2. `X-API-Key` header
3. `api_key` query parameter
4. Cookie (if `CookieName` is set)

### Options

```go
type Options struct {
    // Realm is used in WWW-Authenticate header
    Realm string

    // CookieName enables cookie-based auth for browser flows
    // (useful for /debug/pprof/* endpoints)
    CookieName string
}
```

### Example: Protecting pprof

```go
// Allow API key via cookie for browser access to pprof
r.Group(func(r chi.Router) {
    r.Use(apikey.Require(appCfg.DebugKey, apikey.Options{
        Realm:      "debug",
        CookieName: "debug_auth",
    }, logger))
    r.Mount("/debug", middleware.Profiler())
})
```

First access with `?api_key=...` sets a cookie, enabling subsequent browser navigation without the query parameter.

---

## Multiple Providers

Support multiple OAuth2 providers in the same application:

```go
googleAuth, _ := oauth2.Google(oauth2.GoogleConfig{...}, logger)
githubAuth, _ := oauth2.GitHub(oauth2.GitHubConfig{...}, logger)

// Each provider gets its own routes
r.Get("/auth/google/login", googleAuth.LoginHandler())
r.Get("/auth/google/callback", googleAuth.CallbackHandler())
r.Get("/auth/github/login", githubAuth.LoginHandler())
r.Get("/auth/github/callback", githubAuth.CallbackHandler())

// Protected routes can use either provider
r.Group(func(r chi.Router) {
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Try Google first, then GitHub
            if session, _ := googleAuth.GetSession(r); session != nil && !session.IsExpired() {
                ctx := oauth2.ContextWithUser(r.Context(), &session.User)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
            if session, _ := githubAuth.GetSession(r); session != nil && !session.IsExpired() {
                ctx := oauth2.ContextWithUser(r.Context(), &session.User)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
            http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
        })
    })
    r.Get("/dashboard", dashboardHandler)
})
```

## See Also

- [middleware](../middleware/middleware.md) — CORS and other middleware
- [app](../app/app.md) — Application lifecycle
