# jwt

JSON Web Token authentication for WAFFLE applications.

## Overview

The `jwt` package provides:
- **Signer** — Create and verify JWTs using HMAC-SHA256
- **Claims** — Standard and custom claim types with builders
- **Middleware** — HTTP middleware for JWT authentication
- **TokenService** — Access/refresh token management

## Import

```go
import "github.com/dalemusser/waffle/auth/jwt"
```

---

## Quick Start

```go
// Create signer
signer, _ := jwt.NewHS256String("your-secret-key")

// Create token
claims := jwt.NewUserBuilder().
    UserID("123").
    Username("alice").
    Roles("admin", "user").
    ExpiresIn(time.Hour).
    Build()

token, _ := signer.Sign(claims)

// Verify token
var parsed jwt.UserClaims
err := signer.Verify(token, &parsed)

// Use middleware
r := chi.NewRouter()
r.Use(jwt.Middleware(jwt.Config{
    Signer: signer,
    ClaimsFactory: func() any { return &jwt.UserClaims{} },
}))
```

---

## Signer

Creates and verifies JWTs.

### Creating a Signer

**Location:** `jwt.go`

```go
// From bytes
signer, err := jwt.NewHS256([]byte("secret"))

// From string
signer, err := jwt.NewHS256String("secret")
```

### Signing Tokens

```go
claims := &jwt.Claims{
    Subject:   "user-123",
    Issuer:    "myapp",
    ExpiresAt: jwt.NewTime(time.Now().Add(time.Hour)),
}

token, err := signer.Sign(claims)
// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Verifying Tokens

```go
var claims jwt.Claims
err := signer.Verify(token, &claims)

if errors.Is(err, jwt.ErrTokenExpired) {
    // Token has expired
}
if errors.Is(err, jwt.ErrInvalidSignature) {
    // Signature doesn't match
}
```

### Parsing Without Verification

For debugging or when signature is verified elsewhere:

```go
var claims jwt.Claims
err := jwt.Parse(token, &claims)
```

---

## Claims

### Standard Claims

**Location:** `jwt.go`

```go
type Claims struct {
    Issuer    string   `json:"iss,omitempty"`
    Subject   string   `json:"sub,omitempty"`
    Audience  Audience `json:"aud,omitempty"`
    ExpiresAt *Time    `json:"exp,omitempty"`
    NotBefore *Time    `json:"nbf,omitempty"`
    IssuedAt  *Time    `json:"iat,omitempty"`
    ID        string   `json:"jti,omitempty"`
}
```

### User Claims

**Location:** `claims.go`

Common pattern for user authentication:

```go
type UserClaims struct {
    Claims
    UserID   string   `json:"uid,omitempty"`
    Username string   `json:"username,omitempty"`
    Email    string   `json:"email,omitempty"`
    Roles    []string `json:"roles,omitempty"`
}

// Role checking
claims.HasRole("admin")
claims.HasAnyRole("admin", "moderator")
claims.HasAllRoles("user", "verified")
```

### Custom Claims

Embed standard claims or use generics:

```go
// Embedding
type MyClaims struct {
    jwt.Claims
    TenantID string `json:"tenant_id"`
    Plan     string `json:"plan"`
}

// With generics
type CustomData struct {
    TenantID string
    Plan     string
}
claims := jwt.CustomClaims[CustomData]{
    Claims: jwt.Claims{Subject: "user-123"},
    Custom: CustomData{TenantID: "t1", Plan: "pro"},
}
```

### Builders

Fluent API for creating claims:

```go
// Standard claims
claims := jwt.NewBuilder().
    Issuer("myapp").
    Subject("user-123").
    Audience("api", "web").
    ExpiresIn(time.Hour).
    ID("unique-id").
    Build()

// User claims
userClaims := jwt.NewUserBuilder().
    Issuer("myapp").
    UserID("123").
    Username("alice").
    Email("alice@example.com").
    Roles("admin", "user").
    ExpiresIn(24 * time.Hour).
    Build()
```

---

## Middleware

### Basic Middleware

**Location:** `middleware.go`

```go
r := chi.NewRouter()
r.Use(jwt.Middleware(jwt.Config{
    Signer: signer,
}))

r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
    claims := jwt.ClaimsFromContext(r.Context())
    fmt.Fprintf(w, "Hello, %s", claims.Subject)
})
```

### Config Options

```go
jwt.Middleware(jwt.Config{
    // Required: signer for verification
    Signer: signer,

    // Where to find the token
    // Options: "header:Authorization", "query:token", "cookie:jwt"
    TokenLookup: "header:Authorization",

    // Auth scheme (for header lookup)
    AuthScheme: "Bearer",

    // Custom error handling
    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{
            "error": err.Error(),
        })
    },

    // Skip authentication for certain requests
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health"
    },

    // Custom claims type
    ClaimsFactory: func() any { return &jwt.UserClaims{} },
})
```

### Typed Middleware

```go
// Middleware that parses into specific claims type
r.Use(jwt.MiddlewareWithClaims[jwt.UserClaims](signer))
```

### Optional Authentication

Parse JWT if present, but don't require it:

```go
r.Use(jwt.Optional(jwt.Config{
    Signer: signer,
}))

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    claims := jwt.ClaimsFromContext(r.Context())
    if claims != nil {
        // Authenticated user
    } else {
        // Anonymous user
    }
})
```

### Getting Claims from Context

```go
// Generic (returns any)
claims := jwt.FromContext(r.Context())

// Standard claims
claims := jwt.ClaimsFromContext(r.Context())

// User claims
claims := jwt.UserClaimsFromContext(r.Context())

// Typed generic
claims := jwt.GetClaims[MyClaims](r.Context())
```

### Role-Based Access

```go
r.Group(func(r chi.Router) {
    r.Use(jwt.Middleware(cfg))
    r.Use(jwt.RequireRole("admin"))
    r.Get("/admin", adminHandler)
})

r.Group(func(r chi.Router) {
    r.Use(jwt.Middleware(cfg))
    r.Use(jwt.RequireAnyRole("admin", "moderator"))
    r.Delete("/posts/{id}", deletePostHandler)
})

r.Group(func(r chi.Router) {
    r.Use(jwt.Middleware(cfg))
    r.Use(jwt.RequireAllRoles("user", "verified"))
    r.Post("/comments", createCommentHandler)
})
```

---

## Token Service

Manages access/refresh token pairs with rotation.

### Setup

**Location:** `refresh.go`

```go
signer, _ := jwt.NewHS256String("secret")
store := jwt.NewMemoryRefreshStore()

service := jwt.NewTokenService(jwt.TokenServiceConfig{
    Signer:     signer,
    Store:      store,
    AccessTTL:  15 * time.Minute,
    RefreshTTL: 7 * 24 * time.Hour,
    Issuer:     "myapp",
    Audience:   []string{"api"},
})
```

### Creating Token Pairs

```go
// On login
pair, err := service.CreateTokenPair(ctx, "user-123", map[string]any{
    "roles": []string{"admin"},
})

// Response:
// {
//   "access_token": "eyJ...",
//   "refresh_token": "a1b2c3...",
//   "expires_in": 900,
//   "token_type": "Bearer"
// }
```

### Refreshing Tokens

```go
// Client sends refresh token
newPair, err := service.RefreshTokens(ctx, refreshToken)
if err != nil {
    // Invalid or expired refresh token
}
// Old refresh token is invalidated (rotation)
```

### Revoking Tokens

```go
// Revoke single refresh token (logout from one device)
service.RevokeRefreshToken(ctx, refreshToken)

// Revoke all refresh tokens (logout from all devices)
service.RevokeAllTokens(ctx, "user-123")
```

### Custom Claims with Token Service

```go
service := jwt.NewTokenService(jwt.TokenServiceConfig{
    Signer: signer,
    Store:  store,
    ClaimsFunc: func(subject string, data any) any {
        userData := data.(map[string]any)
        return &jwt.UserClaims{
            Claims: jwt.Claims{
                Subject:   subject,
                IssuedAt:  jwt.NewTime(time.Now()),
                ExpiresAt: jwt.NewTime(time.Now().Add(15 * time.Minute)),
            },
            UserID:   subject,
            Username: userData["username"].(string),
            Roles:    userData["roles"].([]string),
        }
    },
})
```

---

## WAFFLE Integration

### Complete Auth Setup

```go
func main() {
    logger, _ := zap.NewProduction()

    // JWT setup
    secret := os.Getenv("JWT_SECRET")
    signer, _ := jwt.NewHS256String(secret)

    // Token service
    store := jwt.NewMemoryRefreshStore() // Use Redis in production
    tokenService := jwt.NewTokenService(jwt.TokenServiceConfig{
        Signer:     signer,
        Store:      store,
        AccessTTL:  15 * time.Minute,
        RefreshTTL: 7 * 24 * time.Hour,
    })

    app := waffle.New(waffle.Config{Logger: logger})

    setupRoutes(app.Router(), signer, tokenService)
    app.Run()
}

func setupRoutes(r chi.Router, signer *jwt.Signer, tokenService *jwt.TokenService) {
    // Public routes
    r.Post("/auth/login", loginHandler(tokenService))
    r.Post("/auth/refresh", refreshHandler(tokenService))

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(jwt.Middleware(jwt.Config{
            Signer:        signer,
            ClaimsFactory: func() any { return &jwt.UserClaims{} },
        }))

        r.Get("/me", meHandler)
        r.Post("/auth/logout", logoutHandler(tokenService))

        // Admin only
        r.Group(func(r chi.Router) {
            r.Use(jwt.RequireRole("admin"))
            r.Get("/admin/users", listUsersHandler)
        })
    })
}
```

### Login Handler

```go
func loginHandler(tokenService *jwt.TokenService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Username string `json:"username"`
            Password string `json:"password"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        // Validate credentials
        user, err := authenticateUser(req.Username, req.Password)
        if err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        // Create tokens
        pair, err := tokenService.CreateTokenPair(r.Context(), user.ID, map[string]any{
            "username": user.Username,
            "roles":    user.Roles,
        })
        if err != nil {
            http.Error(w, "token error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(pair)
    }
}
```

### Refresh Handler

```go
func refreshHandler(tokenService *jwt.TokenService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            RefreshToken string `json:"refresh_token"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        pair, err := tokenService.RefreshTokens(r.Context(), req.RefreshToken)
        if err != nil {
            http.Error(w, "invalid refresh token", http.StatusUnauthorized)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(pair)
    }
}
```

### Logout Handler

```go
func logoutHandler(tokenService *jwt.TokenService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            RefreshToken string `json:"refresh_token"`
            AllDevices   bool   `json:"all_devices"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        claims := jwt.UserClaimsFromContext(r.Context())

        if req.AllDevices {
            tokenService.RevokeAllTokens(r.Context(), claims.Subject)
        } else if req.RefreshToken != "" {
            tokenService.RevokeRefreshToken(r.Context(), req.RefreshToken)
        }

        w.WriteHeader(http.StatusNoContent)
    }
}
```

---

## Errors

```go
jwt.ErrInvalidToken      // Malformed token
jwt.ErrInvalidSignature  // Signature verification failed
jwt.ErrTokenExpired      // Token has expired
jwt.ErrTokenNotYetValid  // Token's nbf is in the future
jwt.ErrMissingSecret     // No secret provided to signer
jwt.ErrUnsupportedAlg    // Algorithm mismatch
jwt.ErrInvalidClaims     // Claims parsing failed
```

---

## Security Considerations

1. **Secret Key**: Use a strong, random secret (32+ bytes). Store securely.

2. **Token Lifetime**: Keep access tokens short-lived (15-60 minutes).

3. **Refresh Token Rotation**: Always rotate refresh tokens on use.

4. **HTTPS Only**: Always transmit tokens over HTTPS.

5. **Token Storage**: Store refresh tokens securely (httpOnly cookies or secure storage).

6. **Revocation**: Implement token revocation for logout and security incidents.

---

## See Also

- [session](../../session/session.md) — Server-side sessions
- [auth/oauth2](../oauth2/oauth2.md) — OAuth2 authentication
- [middleware](../../middleware/middleware.md) — HTTP middleware
