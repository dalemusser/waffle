# session

Server-side session management for WAFFLE applications.

## Overview

The `session` package provides:
- **Manager** — Session lifecycle management with cookie handling
- **Session** — Key-value storage with typed getters
- **Store** — Pluggable backends (memory, Redis)
- **Middleware** — Automatic session loading and saving

## Import

```go
import "github.com/dalemusser/waffle/session"
```

---

## Quick Start

```go
// Create store and manager
store := session.NewMemoryStore()
manager := session.NewManager(store, session.DefaultConfig())

// Use middleware
r := chi.NewRouter()
r.Use(session.Middleware(manager))

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    sess := session.FromContext(r.Context())

    // Read values
    visits := sess.GetInt("visits")

    // Write values
    sess.Set("visits", visits+1)
    sess.Set("last_visit", time.Now())

    fmt.Fprintf(w, "Visits: %d", visits+1)
})
```

---

## Manager

Handles session creation, retrieval, and persistence.

### NewManager

**Location:** `session.go`

```go
func NewManager(store Store, cfg Config) *Manager
```

**Config:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| CookieName | string | "session_id" | Session cookie name |
| MaxAge | time.Duration | 24 hours | Session lifetime |
| Path | string | "/" | Cookie path |
| Domain | string | "" | Cookie domain |
| Secure | bool | true | HTTPS only |
| HttpOnly | bool | true | No JavaScript access |
| SameSite | http.SameSite | Lax | CSRF protection |
| IDGenerator | func() (string, error) | crypto random | ID generation |

### Basic Usage

```go
store := session.NewMemoryStore()

manager := session.NewManager(store, session.Config{
    CookieName: "myapp_session",
    MaxAge:     7 * 24 * time.Hour, // 1 week
    Secure:     true,
    HttpOnly:   true,
    SameSite:   http.SameSiteStrictMode,
})
defer manager.Close()
```

### Manager Methods

```go
// Get session (creates new if needed)
sess, err := manager.Get(r)

// Create new session
sess, err := manager.New()

// Save session and set cookie
err := manager.Save(w, r, sess)

// Delete session and clear cookie
err := manager.Destroy(w, r, sess)

// Regenerate session ID (after login)
err := manager.Regenerate(w, r, sess)

// Extend session expiration
err := manager.Refresh(w, r, sess)
```

---

## Session

Key-value storage for session data.

### Reading Values

```go
sess := session.FromContext(r.Context())

// Generic get
value, ok := sess.Get("key")

// Typed getters
name := sess.GetString("name")           // "" if missing
count := sess.GetInt("count")            // 0 if missing
active := sess.GetBool("active")         // false if missing
loginTime := sess.GetTime("login_time")  // zero time if missing
```

### Writing Values

```go
sess.Set("user_id", 123)
sess.Set("username", "alice")
sess.Set("roles", []string{"admin", "user"})
sess.Set("preferences", map[string]any{
    "theme": "dark",
    "lang":  "en",
})
```

### Other Operations

```go
sess.Delete("key")           // Remove a key
sess.Clear()                 // Remove all keys
keys := sess.Keys()          // Get all keys
values := sess.Values()      // Get all data (copy)
sess.ID()                    // Session ID
sess.IsNew()                 // True if just created
sess.Modified()              // True if data changed
sess.ExpiresAt()             // Expiration time
```

---

## Stores

### Memory Store

In-memory storage for development or single-instance deployments.

```go
// Default config
store := session.NewMemoryStore()

// Custom cleanup interval
store := session.NewMemoryStoreWithConfig(session.MemoryStoreConfig{
    CleanupInterval: 5 * time.Minute,
})

// Check session count
count := store.Size()
```

### Redis Store

Redis-backed storage for distributed applications.

```go
// With existing client
store := session.NewRedisStore(redisClient)

// Simple connection
store, err := session.ConnectRedis("localhost:6379", "", 0)

// Full config
store, err := session.NewRedisStoreWithConfig(session.RedisStoreConfig{
    Address:   "localhost:6379",
    Password:  "secret",
    DB:        0,
    KeyPrefix: "myapp:session:",
    PoolSize:  20,
})
```

### Store Interface

Implement for custom backends:

```go
type Store interface {
    Load(ctx context.Context, id string) (*SessionData, error)
    Save(ctx context.Context, data *SessionData) error
    Delete(ctx context.Context, id string) error
    Close() error
}
```

---

## Middleware

### Middleware

**Location:** `middleware.go`

```go
func Middleware(m *Manager) func(http.Handler) http.Handler
```

Automatically loads sessions and saves them if modified.

```go
r := chi.NewRouter()
r.Use(session.Middleware(manager))

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    sess := session.FromContext(r.Context())
    // Session is automatically saved if modified
})
```

### Context Functions

```go
// Get session from context (returns nil if not found)
sess := session.FromContext(r.Context())

// Get session, panic if not found
sess := session.MustFromContext(r.Context())
```

### RequireSession

Returns 401 if no valid session exists.

```go
r.Group(func(r chi.Router) {
    r.Use(session.RequireSession(manager))
    r.Get("/dashboard", dashboardHandler)
})
```

### RequireKey

Returns 401 if session doesn't have a specific key.

```go
r.Group(func(r chi.Router) {
    r.Use(session.RequireKey("user_id"))
    r.Get("/profile", profileHandler)
})
```

---

## Flash Messages

One-time messages that are deleted after reading.

```go
// Set flash message
session.Flash(sess, "success", "Your changes have been saved!")
session.Flash(sess, "error", "Something went wrong")

// Get and remove flash message
if msg, ok := session.GetFlash(sess, "success"); ok {
    fmt.Println(msg)
}

// String helper
msg := session.GetFlashString(sess, "error")
```

---

## WAFFLE Integration

### Application Setup

```go
func main() {
    logger, _ := zap.NewProduction()

    // Choose store based on environment
    var store session.Store
    if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
        var err error
        store, err = session.ConnectRedis(redisURL, "", 0)
        if err != nil {
            log.Fatal(err)
        }
    } else {
        store = session.NewMemoryStore()
    }

    manager := session.NewManager(store, session.Config{
        CookieName: "myapp_session",
        MaxAge:     24 * time.Hour,
        Secure:     os.Getenv("ENV") == "production",
    })

    app := waffle.New(waffle.Config{
        Logger: logger,
        Shutdown: func(ctx context.Context) error {
            return manager.Close()
        },
    })

    // Make manager available
    app.Set("sessions", manager)

    setupRoutes(app.Router(), manager)
    app.Run()
}

func setupRoutes(r chi.Router, manager *session.Manager) {
    // Apply session middleware
    r.Use(session.Middleware(manager))

    r.Get("/", homeHandler)
    r.Post("/login", loginHandler(manager))
    r.Post("/logout", logoutHandler(manager))

    r.Group(func(r chi.Router) {
        r.Use(session.RequireKey("user_id"))
        r.Get("/dashboard", dashboardHandler)
    })
}
```

### Login Handler

```go
func loginHandler(manager *session.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Validate credentials...
        user, err := authenticateUser(r)
        if err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        sess := session.FromContext(r.Context())

        // Regenerate session ID to prevent fixation attacks
        if err := manager.Regenerate(w, r, sess); err != nil {
            http.Error(w, "session error", http.StatusInternalServerError)
            return
        }

        // Store user data
        sess.Set("user_id", user.ID)
        sess.Set("username", user.Username)
        sess.Set("login_time", time.Now())

        session.Flash(sess, "success", "Welcome back!")

        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
    }
}
```

### Logout Handler

```go
func logoutHandler(manager *session.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sess := session.FromContext(r.Context())

        if err := manager.Destroy(w, r, sess); err != nil {
            http.Error(w, "logout failed", http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}
```

### Protected Handler

```go
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    sess := session.FromContext(r.Context())

    userID := sess.GetInt("user_id")
    username := sess.GetString("username")

    // Check for flash messages
    if msg := session.GetFlashString(sess, "success"); msg != "" {
        // Display success message
    }

    // Render dashboard...
}
```

### Checking Authentication

```go
func isAuthenticated(r *http.Request) bool {
    sess := session.FromContext(r.Context())
    if sess == nil {
        return false
    }
    _, ok := sess.Get("user_id")
    return ok
}

func getCurrentUserID(r *http.Request) (int, bool) {
    sess := session.FromContext(r.Context())
    if sess == nil {
        return 0, false
    }
    id := sess.GetInt("user_id")
    return id, id != 0
}
```

---

## Security Best Practices

### Session Fixation Prevention

Always regenerate the session ID after authentication:

```go
// After successful login
manager.Regenerate(w, r, sess)
sess.Set("user_id", user.ID)
```

### Secure Cookie Settings

```go
manager := session.NewManager(store, session.Config{
    Secure:   true,                      // HTTPS only
    HttpOnly: true,                      // No JS access
    SameSite: http.SameSiteStrictMode,   // CSRF protection
})
```

### Session Timeout

Set appropriate session lifetimes:

```go
// Short sessions for sensitive apps
cfg := session.Config{
    MaxAge: 15 * time.Minute,
}

// Refresh on activity
func activityMiddleware(manager *session.Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            sess := session.FromContext(r.Context())
            if sess != nil && !sess.IsNew() {
                manager.Refresh(w, r, sess)
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Sensitive Data

Don't store sensitive data in sessions:

```go
// Good: Store user ID, fetch data as needed
sess.Set("user_id", user.ID)

// Bad: Don't store passwords, tokens, or PII
// sess.Set("password", user.Password)  // Never do this
```

---

## Errors

```go
session.ErrNotFound       // Session doesn't exist
session.ErrExpired        // Session has expired
session.ErrInvalidSession // Session data is invalid
```

---

## See Also

- [auth/oauth2](../auth/oauth2/oauth2.md) — OAuth2 authentication
- [cache](../cache/cache.md) — Response caching
- [middleware](../middleware/middleware.md) — HTTP middleware
