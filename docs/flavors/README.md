# WAFFLE Flavors

*Different ways to serve your WAFFLE — choose the UI approach that fits your needs.*

---

## What Are Flavors?

WAFFLE is a substrate, not a stack. It provides a solid foundation — lifecycle management, configuration, routing, database connections, authentication — and lets you choose how to build your user interface on top.

We call these choices **flavors** because they're variations of the same delicious foundation. Each flavor is a valid way to serve your WAFFLE, optimized for different situations and preferences.

For the philosophical foundation of this design, see [WAFFLE as Substrate](../philosophy/waffle-as-substrate.md) and [UI Paradigms](../philosophy/ui-paradigms.md).

---

## Available Flavors

### Fully Documented

| Flavor | Description | Best For |
|--------|-------------|----------|
| [**Server HTML + HTMX**](./server-html-htmx/README.md) | Server-rendered HTML with HTMX for interactivity | Admin dashboards, internal tools, educational apps, systems built to last |

### Documented (Placeholder)

These flavors are fully supported but detailed documentation is forthcoming:

| Flavor | Description | Best For |
|--------|-------------|----------|
| [**Server HTML + Vanilla JS**](./server-html-vanilla/README.md) | Server-rendered HTML with vanilla JavaScript | Teams preferring explicit JS control |
| [**SPA + WAFFLE API**](./spa-backend/README.md) | React/Vue/Svelte frontend, WAFFLE as API server | Rich interactive applications, existing SPA expertise |
| [**Hybrid Islands**](./hybrid-islands/README.md) | Server HTML shell with SPA components where needed | Mix of simple and complex pages |
| [**Native Clients**](./native-clients/README.md) | Mobile apps, desktop apps, Unity games | Non-web clients using WAFFLE APIs |

---

## Choosing a Flavor

### Start Here

If you don't have existing requirements dictating your approach:

**→ Use [Server HTML + HTMX](./server-html-htmx/README.md)**

This is the author's recommended approach. See [Why HTMX + Tailwind](../philosophy/why-htmx-tailwind.md) for the reasoning.

### Decision Guide

| If You... | Consider... |
|-----------|-------------|
| Want simplicity and maintainability | Server HTML + HTMX |
| Have an existing React/Vue team | SPA + WAFFLE API |
| Need rich interactive widgets in a simple app | Hybrid Islands |
| Are building mobile/desktop apps | Native Clients |
| Want explicit control over JavaScript | Server HTML + Vanilla JS |

---

## What All Flavors Share

Regardless of which flavor you choose, you get the full WAFFLE foundation:

- **WAFFLE Core** — Lifecycle, configuration, routing, middleware
- **WAFFLE Pantry** — Email, caching, database utilities, and more
- **Authentication** — OAuth2, API keys, session management
- **Deployment** — HTTPS, Let's Encrypt, Windows services
- **Observability** — Logging, metrics, health checks

The flavor choice affects how you build your UI, not how you build your backend. Your WAFFLE stays the same — only the toppings change.

---

## See Also

- [Philosophy: UI Paradigms](../philosophy/ui-paradigms.md) — Detailed discussion of each approach
- [Philosophy: Why HTMX + Tailwind](../philosophy/why-htmx-tailwind.md) — The recommended path
- [Core Documentation](../core/README.md) — WAFFLE foundation (all flavors)
- [Guides](../guides/README.md) — Task-oriented docs (paradigm-agnostic)

---

[← Back to Documentation Index](../waffle-docs.md)
