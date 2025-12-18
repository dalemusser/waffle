# Hybrid Islands Flavor

*Server-rendered HTML shell with SPA components mounted into specific areas.*

---

## Overview

This flavor combines server-rendered HTML (via Go templates) with SPA components (React, Vue, Svelte) that mount into specific DOM elements — "islands" of interactivity in a sea of static HTML.

This is the pattern many mature teams settle into after trying full SPA.

---

## Status

**This flavor is supported but documentation is forthcoming.**

WAFFLE provides the foundation. The island mounting is standard SPA practice.

---

## Why Choose This Flavor

| Reason | Explanation |
|--------|-------------|
| **Complexity only where needed** | Most pages stay simple, SPA power only where justified |
| **Progressive enhancement** | Start server-rendered, add interactivity incrementally |
| **Specific complex widgets** | Rich editors, real-time collaboration, complex drag-drop |
| **Migration path** | Evolving from server-rendered toward more dynamic |

---

## WAFFLE's Role

| Function | Description |
|----------|-------------|
| **Page composition** | Go templates render the overall page structure |
| **Authentication** | Session and auth handled server-side |
| **Initial render** | HTML arrives fully formed, islands hydrate |
| **API endpoints** | JSON APIs for island data needs |

---

## The Pattern

```html
<!-- Server-rendered template -->
<html>
<head>...</head>
<body>
    <nav><!-- Server-rendered navigation --></nav>

    <main>
        <h1>Dashboard</h1>

        <!-- Island: React mounts here -->
        <div id="analytics-chart"
             data-endpoint="/api/analytics"
             data-period="30d">
        </div>

        <!-- Regular server HTML -->
        <section>
            <h2>Recent Activity</h2>
            <!-- HTMX or static HTML -->
        </section>

        <!-- Another island -->
        <div id="collaborative-editor"
             data-doc-id="{{.DocID}}">
        </div>
    </main>
</body>
</html>
```

---

## Coming Soon

- Island mounting patterns
- Data passing from server to islands
- Authentication context sharing
- Build tooling for islands
- When to use HTMX vs island

---

## See Also

- [Philosophy: UI Paradigms](../../philosophy/ui-paradigms.md) — All valid approaches
- [Server HTML + HTMX](../server-html-htmx/README.md) — For simpler interactivity
- [SPA Backend](../spa-backend/README.md) — For full SPA approach

---

[← Back to Flavors](../README.md)
