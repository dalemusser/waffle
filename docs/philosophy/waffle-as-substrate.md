# WAFFLE as Substrate

*WAFFLE is not a stack — it's a foundation you build stacks on.*

---

## The Core Idea

When people ask "what front-end libraries can WAFFLE use?", they're asking the wrong question.

The right question is:

> What UI execution models can reasonably sit on top of a Go, server-centric, layered engineering platform like WAFFLE?

WAFFLE is a **substrate**, not a **stack**. HTMX + Tailwind is one expression of WAFFLE — the reference implementation, the teaching surface, the author's preferred approach — but it is not WAFFLE's identity.

WAFFLE's identity is:
- Layered engineering
- Go-first backend clarity
- UI model optionality
- Systems that age well

---

## What WAFFLE Provides

WAFFLE gives you:

| Capability | Description |
|------------|-------------|
| **Go HTTP server** | The authoritative backend for your application |
| **Clear request/response boundaries** | Explicit HTTP semantics, no magic |
| **First-class HTML rendering** | Go templates with layouts, partials, and composition |
| **First-class API delivery** | JSON endpoints with the same routing and middleware |
| **Long-lived infrastructure** | Health checks, metrics, graceful shutdown |
| **No JS runtime on server** | Go runs your backend, period |
| **Explicit layering** | Auth, config, logging, services — all visible and controllable |
| **Philosophy of durability** | Built for systems that need to last |

---

## What WAFFLE Excludes

By being Go-centric and server-authoritative, WAFFLE immediately excludes:

| Approach | Why It Doesn't Fit |
|----------|-------------------|
| **Node-as-server** (Next.js server, NestJS) | WAFFLE is the server |
| **Full-stack frameworks** (Rails, Laravel, Phoenix) | They replace the server, not build on it |
| **Browser-owns-the-app** platforms | WAFFLE maintains server authority |
| **Server-side JavaScript** | Go is the runtime |

These aren't criticisms of those approaches — they're simply different architectural choices. WAFFLE is for people who want a Go server as their foundation.

---

## What Does Fit

WAFFLE supports multiple UI paradigms. Each is a legitimate way to build on the substrate:

1. **Server-Rendered HTML + HTMX** — The reference implementation
2. **Server-Rendered HTML + Progressive JS** — Vanilla JS, Alpine.js, Stimulus
3. **SPA + WAFFLE as API Server** — React, Vue, Svelte with WAFFLE backend
4. **Hybrid Islands** — Server HTML shell with SPA islands where needed
5. **Web Components** — Standards-based, framework-agnostic
6. **Native Clients** — Mobile apps, desktop apps, Unity games
7. **Live UI Systems** — SSE or WebSocket-driven interfaces

See [UI Paradigms](./ui-paradigms.md) for detailed discussion of each approach.

---

## The Practical Implication

When you choose WAFFLE, you're choosing:

- **A Go server** as your authoritative backend
- **Clear HTTP boundaries** between client and server
- **Freedom to choose** how you build your UI
- **A foundation** that doesn't lock you into one approach

You are *not* choosing:
- A specific JavaScript framework
- A specific CSS approach
- A specific rendering model

Those choices are yours. WAFFLE provides a server worth building on.

---

## The Clean Statement

> WAFFLE is a Go-based web application platform that supports multiple UI paradigms — from server-rendered HTML to modern SPAs — while providing a durable, inspectable backend foundation.

Or more directly:

> WAFFLE does not prescribe how you build your UI. It provides a server worth building on.

---

## See Also

- [UI Paradigms](./ui-paradigms.md) — Detailed breakdown of each valid approach
- [Why HTMX + Tailwind](./why-htmx-tailwind.md) — The author's recommended path
- [Why Go](./why-go.md) — Advantages of Go as a foundation

---

[← Back to Philosophy](./README.md)
