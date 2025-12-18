# Server HTML + HTMX Flavor

*Server-rendered HTML with HTMX for dynamic interactivity — the recommended WAFFLE approach.*

---

## Overview

This flavor uses Go templates to render HTML on the server, with HTMX providing dynamic behavior without the complexity of JavaScript frameworks. Tailwind CSS handles styling, and Flowbite provides pre-built component patterns.

This is WAFFLE's **reference implementation** — the most documented, most tested, and author-recommended approach.

---

## Why This Flavor

| Advantage | Explanation |
|-----------|-------------|
| **Simplicity** | Server renders HTML. Browser displays it. No build step. |
| **Maintainability** | Fewer moving parts, easier debugging |
| **Performance** | No client-side framework overhead |
| **Longevity** | HTML and HTTP are stable foundations |
| **Accessibility** | Works without JavaScript, enhanced with it |

For the full reasoning, see [Why HTMX + Tailwind](../../philosophy/why-htmx-tailwind.md).

---

## Documentation

| Document | Description |
|----------|-------------|
| [**Templates**](./templates.md) | Go template rendering, layouts, partials, and view organization |
| [**HTMX Integration**](./htmx.md) | Dynamic frontend interactions without JavaScript frameworks |
| [**Tailwind CSS**](./tailwind.md) | Setting up the standalone CLI, building CSS without Node.js |
| [**Flowbite Components**](./flowbite.md) | Using Flowbite's Tailwind component library with Go templates |

### Related Guides

| Guide | Description |
|-------|-------------|
| [**Static Files (Embedded)**](../../guides/fileserving/static-files.md) | Embedding CSS/JS using `go:embed` |
| [**Static Files (Filesystem)**](../../guides/fileserving/static-files-filesystem.md) | Serving from disk with pre-compression |

---

## Technology Stack

| Technology | Role | Version |
|------------|------|---------|
| [Go Templates](https://pkg.go.dev/html/template) | Server-side HTML rendering | Built-in |
| [HTMX](https://htmx.org/) | Dynamic behavior via HTML attributes | 1.9.x |
| [Tailwind CSS](https://tailwindcss.com/) | Utility-first styling | 3.x (standalone CLI) |
| [Flowbite](https://flowbite.com/) | Pre-built Tailwind components | 2.x |

---

## See Also

- [Philosophy: Why HTMX + Tailwind](../../philosophy/why-htmx-tailwind.md) — The full reasoning
- [Philosophy: UI Paradigms](../../philosophy/ui-paradigms.md) — All valid approaches
- [Guides: First Service](../../guides/getting-started/first-service.md) — Getting started with WAFFLE

---

[← Back to Flavors](../README.md)
