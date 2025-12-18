# Authentication in WAFFLE

*How to add authentication to your WAFFLE application.*

This guide shows how to integrate authentication using WAFFLE's built-in `auth/oauth2` package.

---

## Overview

WAFFLE provides authentication support through the `auth/oauth2` package, which includes:

- **Multiple protocols** — OAuth2, OIDC, SAML 2.0, OAuth 1.0a, LTI 1.3
- **Session management** — In-memory (development) and interfaces for Redis/MongoDB
- **25+ providers** — Social, enterprise, and education identity providers

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

## Documentation

All authentication documentation is in the [**providers/**](./providers/README.md) directory:

| Document | Contents |
|----------|----------|
| [**Core Concepts**](./providers/core-concepts.md) | Session stores, middleware, common patterns |
| [**Protocols**](./providers/protocols.md) | Understanding OAuth2, OIDC, SAML, OAuth 1.0a, LTI 1.3 |
| [**Provider Index**](./providers/README.md) | All 25+ providers organized by category |

---

## Quick Links by Category

### Social Providers
[Google](./providers/google.md) ·
[GitHub](./providers/github.md) ·
[Discord](./providers/discord.md) ·
[LinkedIn](./providers/linkedin.md) ·
[Apple](./providers/apple.md)

### Enterprise & Federated Identity
[Microsoft](./providers/microsoft.md) ·
[Okta](./providers/okta.md) ·
[Shibboleth/SAML](./providers/shibboleth.md)

### K-12 SSO
[Clever](./providers/clever.md) ·
[ClassLink](./providers/classlink.md) ·
[GG4L](./providers/gg4l.md)

### Student Information Systems
[PowerSchool](./providers/powerschool.md) ·
[Infinite Campus](./providers/infinite-campus.md) ·
[Skyward](./providers/skyward.md) ·
[Banner](./providers/banner.md) ·
[Workday](./providers/workday.md)

### Learning Management Systems
[Canvas](./providers/canvas.md) ·
[Blackboard](./providers/blackboard.md) ·
[Moodle](./providers/moodle.md) ·
[Schoology](./providers/schoology.md) ·
[Google Classroom](./providers/google-classroom.md)

### Data Standards & Interoperability
[Ed-Fi](./providers/edfi.md) ·
[LTI 1.3](./providers/lti.md)

### Microsoft Education
[Microsoft Education](./providers/microsoft-education.md)

---

## Package Import

```go
import "github.com/dalemusser/waffle/pantry/auth/oauth2"
```

---

## Core Types

| Type | Description |
|------|-------------|
| `oauth2.User` | Authenticated user information |
| `oauth2.Session` | User session with expiration |
| `oauth2.Provider` | Authentication provider with handlers |
| `oauth2.SessionStore` | Interface for session storage |
| `oauth2.StateStore` | Interface for state storage |

---

## Provider Methods

All providers share these methods:

| Method | Description |
|--------|-------------|
| `LoginHandler()` | Handler that redirects to provider |
| `CallbackHandler()` | Handler that processes callback |
| `LogoutHandler()` | Handler that ends user session |
| `RequireAuth(loginURL)` | Middleware for HTML routes |
| `RequireAuthJSON()` | Middleware for API routes |
| `GetSession(r)` | Get session from request |

---

## See Also

- [Routes & Middleware Guide](../../core/routing.md) — Middleware patterns
- [DBDeps Redis Example](../databases/redis.md) — Redis integration for sessions
- [Handler Structure Examples](../patterns/handlers.md) — Feature handler patterns
