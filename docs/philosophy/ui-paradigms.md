# UI Paradigms

*The valid ways to build user interfaces on top of WAFFLE.*

---

## Overview

WAFFLE supports multiple UI paradigms. Below are distinct **development models**, not libraries. Each is a legitimate WAFFLE "mode" with its own strengths and appropriate use cases.

---

## 1. Server-Rendered HTML + HTMX

*The canonical WAFFLE paradigm*

### WAFFLE's Role
- HTML renderer
- Service orchestrator
- State manager

### Why It Fits Perfectly
- WAFFLE already excels at HTML rendering
- Minimal JavaScript complexity
- Long-term maintainability
- Pedagogically strong — easy to understand and debug

### Who Chooses This
- Educators building teaching tools
- Teams building admin dashboards
- Internal tools and business applications
- Systems meant to last decades

### Key Technologies
- Go templates
- HTMX for dynamic behavior
- Tailwind CSS for styling
- Flowbite for component patterns

This is the **reference implementation** — the approach with the most documentation, the most examples, and the author's recommendation. But it is not the only valid choice.

---

## 2. Server-Rendered HTML + Progressive JS

*HTML-first without HTMX*

### WAFFLE's Role
- Same as above
- JS augments behavior, not structure

### UI Tools
- Vanilla JavaScript
- Alpine.js
- Stimulus
- Flowbite JS components

### Why Someone Chooses This
- They prefer HTML-first but don't want HTMX
- They want explicit JavaScript files they control
- They want more direct control over event handling
- Team familiarity with vanilla JS patterns

### The Key Insight

HTMX is optional. The server-rendered philosophy survives without it. You can build excellent applications with Go templates and vanilla JavaScript.

---

## 3. SPA Frontend + WAFFLE as API Server

*React, Vue, or Svelte with a WAFFLE backend*

### This Is Important

React + WAFFLE is:
- ✅ Doable
- ✅ Reasonable
- ✅ Something people will want

### WAFFLE's Role
- Authentication server
- API provider (JSON endpoints)
- Static file server (serves the built SPA)
- Long-lived infrastructure

### Frontend
- React / Vue / Svelte / Solid
- Built and deployed separately
- Or statically served by WAFFLE

### Why This Fits
- WAFFLE doesn't care how HTML is produced
- Go APIs + SPA UIs are extremely common
- WAFFLE's strength is backend clarity, not UI enforcement

### What WAFFLE Does NOT Become
- It does not become a React framework
- It does not run Node
- It does not care about JSX or component syntax

### Analogous To
- Django + React
- Rails (API mode) + React
- Except cleaner and Go-native

---

## 4. Hybrid: Server HTML Shell + SPA Islands

*The best of both worlds*

### The Pattern
- WAFFLE renders the page layout and navigation
- React/Vue mounts into specific DOM elements ("islands")
- SPA logic lives only where it's justified

### WAFFLE's Role
- Page composition
- Authentication and authorization
- Initial HTML render
- API endpoints for island data

### Why This Is Attractive
- Keeps most of the application simple (server-rendered)
- Uses SPA power only where complexity is justified
- Avoids full SPA complexity for simple pages
- Progressive enhancement path

### Who Chooses This
- Teams with complex interactive widgets in otherwise simple apps
- Applications evolving from server-rendered to more dynamic
- Projects that need specific SPA features (rich editors, real-time collaboration)

This is the model many mature teams settle into after trying full SPA.

---

## 5. Web Components

*Standards-based, framework-agnostic*

### UI Model
- Custom Elements
- Shadow DOM
- Vanilla JavaScript

### WAFFLE's Role
- HTML host
- API provider

### Why This Fits WAFFLE Philosophically
- Standards-based (no framework churn)
- Framework-agnostic (works with any approach)
- Long lifespan (browser-native)
- No runtime coupling

### The Reality
This is a slow-burn choice. Web Components haven't achieved mainstream dominance, but they're very aligned with WAFFLE's longevity goals for teams willing to invest.

---

## 6. Native Clients

*Mobile, desktop, and game clients*

### Clients
- iOS (SwiftUI, UIKit)
- Android (Kotlin, Jetpack Compose)
- Desktop apps (Electron, Tauri, native)
- Unity WebGL games

### WAFFLE's Role
- API server
- Authentication authority
- Data and services layer
- File/asset server

### Why This Matters
Often forgotten in web framework discussions, but WAFFLE is explicitly good at this. A well-designed WAFFLE backend serves web, mobile, and desktop clients equally well.

---

## 7. Live UI Systems

*Real-time, server-driven interfaces*

### Examples
- Phoenix LiveView–style patterns
- Server-Sent Events (SSE) driven UIs
- WebSocket-based real-time updates

### WAFFLE's Role
- Stateful server
- Event broadcaster
- Connection manager

### The Reality
This is possible with WAFFLE, but it's not WAFFLE's sweet spot. It's an advanced mode requiring careful state management. Consider whether simpler polling or HTMX's built-in polling handles your use case first.

---

## What Does NOT Fit WAFFLE

Being explicit about what doesn't fit helps clarify what does:

| Approach | Why It Doesn't Fit |
|----------|-------------------|
| **Rails / Laravel / Phoenix** | They *are* the server — they compete with WAFFLE |
| **Next.js as server** | Node-first, replaces Go |
| **NestJS / Express apps** | Node-first, competes with WAFFLE |
| **Frameworks assuming server-side JS** | WAFFLE is Go |
| **Platforms that hide HTTP boundaries** | WAFFLE values explicit HTTP semantics |

These aren't bad approaches — they're simply different foundations. If you want one of these, you don't want WAFFLE.

---

## Choosing Your Paradigm

| If You Want... | Consider... |
|----------------|-------------|
| Simplicity and maintainability | Server-Rendered HTML + HTMX |
| Full control over JavaScript | Server-Rendered HTML + Progressive JS |
| Rich interactive experiences | SPA + WAFFLE API |
| Mix of simple and complex pages | Hybrid Islands |
| Maximum longevity | Web Components or HTMX |
| Mobile/desktop apps | Native Clients + WAFFLE API |
| Real-time features | Start with HTMX polling, escalate to SSE/WebSocket if needed |

---

## The Author's Recommendation

If you don't have existing requirements dictating your UI approach, the author recommends **Server-Rendered HTML + HTMX + Tailwind**.

See [Why HTMX + Tailwind](./why-htmx-tailwind.md) for the reasoning.

But this recommendation is a preference, not a prescription. WAFFLE supports you regardless of which valid paradigm you choose.

---

## See Also

- [WAFFLE as Substrate](./waffle-as-substrate.md) — The foundational concept
- [Why HTMX + Tailwind](./why-htmx-tailwind.md) — The recommended approach
- [Flavors Documentation](../flavors/README.md) — Implementation guides for each paradigm

---

[← Back to Philosophy](./README.md)
