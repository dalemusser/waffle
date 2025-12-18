# Native Clients Flavor

*iOS, Android, desktop, and other native applications with WAFFLE as the backend.*

---

## Overview

This flavor uses WAFFLE purely as an API server and authentication authority, with native applications handling all user interface concerns. This is often forgotten in web framework discussions, but WAFFLE is explicitly good at serving native clients.

---

## Status

**This flavor is supported but documentation is forthcoming.**

WAFFLE's JSON APIs, authentication, and infrastructure work identically for native clients as for web clients.

---

## Why Choose This Flavor

| Reason | Explanation |
|--------|-------------|
| **Mobile apps** | iOS or Android app needs a backend |
| **Desktop apps** | Electron, Tauri, or native desktop applications |
| **Multi-platform** | Same backend serves web, mobile, and desktop |
| **Games** | Unity, Unreal, or other game engines need APIs |

---

## WAFFLE's Role

| Function | Description |
|----------|-------------|
| **API server** | JSON endpoints for all data operations |
| **Authentication** | OAuth2, JWT, API keys — WAFFLE handles auth |
| **File serving** | Asset delivery, file uploads/downloads |
| **Push notifications** | Server-side push notification dispatch |
| **Real-time** | WebSocket connections for live updates |

---

## Client Types

### Mobile

| Platform | Technologies |
|----------|--------------|
| **iOS** | SwiftUI, UIKit, Alamofire |
| **Android** | Kotlin, Jetpack Compose, Retrofit |
| **Cross-platform** | Flutter, React Native, .NET MAUI |

### Desktop

| Platform | Technologies |
|----------|--------------|
| **Electron** | JavaScript/TypeScript |
| **Tauri** | Rust + web frontend |
| **Native** | Swift (macOS), C# (Windows), GTK (Linux) |

### Games

| Engine | Use Case |
|--------|----------|
| **Unity** | WebGL games, mobile games |
| **Unreal** | Desktop/console games |
| **Godot** | Indie games |

---

## Key Considerations

### Authentication

Native clients typically use:
- **OAuth2** — Authorization code flow with PKCE
- **JWT tokens** — Stored securely on device
- **API keys** — For server-to-server or simple auth
- **Biometric** — Device biometrics unlock stored tokens

### API Design

Design APIs that work well for native clients:
- Batch endpoints to reduce round trips
- Pagination for large data sets
- Partial responses (field selection)
- Offline-friendly data structures

### Versioning

Native apps can't be force-updated like web apps:
- API versioning strategy (URL or header)
- Backward compatibility period
- Deprecation communication

---

## Coming Soon

- OAuth2 PKCE flow implementation
- JWT authentication patterns
- API versioning strategies
- Push notification integration
- File upload/download patterns

---

## See Also

- [Philosophy: UI Paradigms](../../philosophy/ui-paradigms.md) — All valid approaches
- [Guides: CORS](../../guides/apis/cors.md) — Cross-origin configuration
- [Guides: OAuth2](../../guides/auth/oauth2.md) — Authentication patterns

---

[← Back to Flavors](../README.md)
