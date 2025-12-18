# Server HTML + Vanilla JS Flavor

*Server-rendered HTML with vanilla JavaScript or lightweight libraries like Alpine.js.*

---

## Overview

This flavor uses Go templates to render HTML on the server, with vanilla JavaScript (or lightweight alternatives like Alpine.js) for client-side interactivity. It shares the server-rendered philosophy with the HTMX flavor but gives you explicit control over JavaScript.

---

## Status

**This flavor is supported but documentation is forthcoming.**

WAFFLE fully supports this approach — the core features (templates, routing, static file serving) work the same as with HTMX. What's different is the client-side interaction pattern.

---

## Why Choose This Flavor

| Reason | Explanation |
|--------|-------------|
| **Explicit JS control** | You write the fetch calls, you control the behavior |
| **Team familiarity** | Your team knows vanilla JS, not HTMX |
| **Gradual adoption** | Easier to adopt incrementally in existing projects |
| **Specific requirements** | Need JS behavior that doesn't fit HTMX patterns |

---

## Key Technologies

| Technology | Role |
|------------|------|
| Go templates | Server-side HTML rendering |
| Vanilla JavaScript | Client-side interactivity |
| Alpine.js (optional) | Declarative behavior without a build step |
| Stimulus (optional) | Modest JS framework from Basecamp |

---

## What Works Today

Everything from WAFFLE core and pantry:

- Template rendering
- Static file serving
- Authentication
- Database integration
- Deployment

The difference from the HTMX flavor is just the client-side approach.

---

## Coming Soon

- Patterns for vanilla JS with WAFFLE
- Alpine.js integration examples
- Progressive enhancement patterns
- Form handling without HTMX

---

## See Also

- [Server HTML + HTMX](../server-html-htmx/README.md) — The recommended approach
- [Philosophy: UI Paradigms](../../philosophy/ui-paradigms.md) — All valid approaches
- [Core: Routing](../../core/routing.md) — Handling requests

---

[← Back to Flavors](../README.md)
