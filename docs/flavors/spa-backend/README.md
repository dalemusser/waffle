# SPA + WAFFLE API Flavor

*React, Vue, or Svelte frontend with WAFFLE as the API backend.*

---

## Overview

This flavor uses WAFFLE as an API server and authentication provider, with a Single Page Application (React, Vue, Svelte, etc.) handling the user interface. The SPA can be served by WAFFLE as static files or deployed separately.

This is a common, production-proven pattern — WAFFLE doesn't care how HTML is produced, it just provides excellent backend infrastructure.

---

## Status

**This flavor is supported but documentation is forthcoming.**

WAFFLE fully supports this approach. The guides for JSON APIs, CORS, and authentication apply directly.

---

## Why Choose This Flavor

| Reason | Explanation |
|--------|-------------|
| **Rich interactivity** | Complex UI interactions are easier in React/Vue |
| **Team expertise** | Your team already knows React/Vue/Svelte |
| **Existing frontend** | Migrating an SPA to a better backend |
| **Offline capability** | SPAs can work offline with service workers |

---

## WAFFLE's Role

| Function | Description |
|----------|-------------|
| **API Server** | JSON endpoints for all data operations |
| **Authentication** | OAuth2, sessions, JWT — WAFFLE handles auth |
| **Static Server** | Serve the built SPA from WAFFLE (optional) |
| **Infrastructure** | Logging, metrics, health checks, graceful shutdown |

---

## What WAFFLE Does NOT Become

- WAFFLE does not become a React framework
- WAFFLE does not run Node.js
- WAFFLE does not care about JSX or component syntax
- WAFFLE does not provide SSR for React (use the HTMX flavor if you want server rendering)

---

## Key Considerations

### CORS

Your SPA will make cross-origin requests. Configure CORS:

```go
r.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins:   []string{"https://myapp.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
}))
```

### Authentication

Options for SPA authentication:

1. **Cookie-based sessions** — WAFFLE sets HttpOnly cookies
2. **JWT tokens** — SPA stores and sends Bearer tokens
3. **OAuth2 flows** — WAFFLE handles the server-side OAuth

### Serving the SPA

Option 1: WAFFLE serves the built SPA
```go
//go:embed spa/dist/*
var spaFiles embed.FS

r.Handle("/*", http.FileServer(http.FS(spaFiles)))
```

Option 2: Deploy SPA separately (CDN, Vercel, etc.)
- WAFFLE is API-only
- Configure CORS for the SPA's origin

---

## Coming Soon

- Complete CORS setup for SPAs
- JWT authentication patterns
- Serving SPA builds from WAFFLE
- Development workflow (SPA dev server + WAFFLE API)

---

## See Also

- [Guides: CORS](../../guides/apis/cors.md) — Cross-origin configuration
- [Philosophy: UI Paradigms](../../philosophy/ui-paradigms.md) — All valid approaches
- [Philosophy: Why Go](../../philosophy/why-go.md) — Go advantages over Node

---

[← Back to Flavors](../README.md)
