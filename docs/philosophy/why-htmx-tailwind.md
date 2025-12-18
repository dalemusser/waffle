# Why HTMX + Tailwind

*The author's recommended approach for building on WAFFLE.*

---

## A Personal Recommendation

This document represents the author's opinion. WAFFLE supports multiple UI paradigms — see [UI Paradigms](./ui-paradigms.md) for the full list. But if you're asking "what should I use?", here's the answer:

**Use server-rendered HTML with HTMX and Tailwind CSS.**

---

## Why This Combination

### HTMX: Hypermedia as the Engine of Application State

HTMX returns web development to its roots while providing modern interactivity.

| Advantage | Explanation |
|-----------|-------------|
| **Server authority** | The server produces HTML. The browser displays it. Clear separation. |
| **Minimal JavaScript** | No build step. No framework updates. No node_modules. |
| **Progressive enhancement** | Works without JS, better with JS. |
| **Debuggable** | Network tab shows exactly what happened. View source works. |
| **Longevity** | HTML is stable. HTMX is simple. This will work in 10 years. |

### Tailwind CSS: Utility-First Styling

Tailwind provides a constrained design system through utility classes.

| Advantage | Explanation |
|-----------|-------------|
| **Co-located styling** | Styles live with the HTML that uses them. |
| **Design constraints** | The utility classes enforce consistent spacing, colors, typography. |
| **No CSS files to manage** | Styles are in the template, not a separate file that drifts. |
| **Predictable** | `class="p-4 text-lg"` always means the same thing. |
| **Purging** | Production CSS contains only what you use. |

### Together: A Complete Approach

HTMX + Tailwind + Go templates gives you:

- Server-rendered HTML (Go templates)
- Dynamic behavior without SPA complexity (HTMX)
- Beautiful, consistent styling (Tailwind)
- No JavaScript build process
- No runtime framework on the client
- Single binary deployment (Go + embedded assets)

---

## What I Can Do Without React

Everything.

Seriously. Every interactive pattern I need in production applications:

| Pattern | HTMX Approach |
|---------|---------------|
| Dynamic forms | `hx-post`, `hx-target` |
| Live search | `hx-get` with `hx-trigger="keyup changed delay:300ms"` |
| Infinite scroll | `hx-get` with `hx-trigger="revealed"` |
| Modal dialogs | `hx-get` returning modal HTML, targeting a modal container |
| Tab interfaces | `hx-get` swapping content |
| Dependent dropdowns | `hx-get` on change, targeting the dependent field |
| Real-time updates | `hx-trigger="every 5s"` for polling |
| Form validation | Server validation returning error states |
| Optimistic UI | `hx-swap-oob` for updating multiple elements |

The patterns that seem to require SPAs usually don't. They require thinking differently about where logic lives.

---

## The Debugging Advantage

When something goes wrong in an HTMX application:

1. Open browser dev tools
2. Look at Network tab
3. See the request
4. See the HTML response
5. Understand exactly what happened

When something goes wrong in a React application:

1. Which component?
2. Which state?
3. Which effect?
4. Is it the store?
5. Is it the API layer?
6. Is it the rendering?
7. `console.log` everywhere
8. Still confused

Simplicity has compounding returns over the lifetime of a project.

---

## The Team Advantage

HTMX + Tailwind is accessible to:

- Backend developers (it's just HTML)
- Designers (they can read and modify templates)
- New team members (the learning curve is gentle)
- Future maintainers (the code explains itself)

React requires:
- JavaScript expertise
- Framework knowledge
- Build tool knowledge
- State management philosophy
- Component thinking
- Hook rules
- Endless ecosystem choices

Every additional requirement is a hiring filter, an onboarding cost, and a maintenance burden.

---

## When This Approach Is Wrong

Be honest about the cases where HTMX + Tailwind might not be the best choice:

| Situation | Consider Instead |
|-----------|------------------|
| **Rich collaborative editing** (Google Docs-style) | SPA with operational transforms |
| **Complex drag-and-drop** | SPA with dedicated drag library |
| **Offline-first requirements** | SPA with service workers |
| **Heavy client-side computation** | SPA or WebAssembly |
| **Team already expert in React** | Maybe just use React |

For most business applications, admin dashboards, content sites, and internal tools — HTMX + Tailwind handles everything needed.

---

## The Longevity Argument

Technology choices should be evaluated on a 10-year horizon, not a 10-month horizon.

| Component | 10-Year Outlook |
|-----------|-----------------|
| **HTML** | Will exist. Will work. |
| **HTTP** | Will exist. Will work. |
| **Go** | Stable, backward compatible, well-maintained. |
| **HTMX** | Small, simple, few dependencies. Easy to maintain or fork. |
| **Tailwind** | CSS utilities. Even if Tailwind disappears, utility classes work. |

Compare to:

| Component | 10-Year Outlook |
|-----------|-----------------|
| **React** | Which version? Which state manager? Which router? |
| **Node ecosystem** | Will your dependencies still build? |
| **Build tools** | Webpack? Vite? The next thing? |
| **JavaScript frameworks** | How many have come and gone in the last 10 years? |

WAFFLE is designed for systems that need to last. HTMX + Tailwind shares that philosophy.

---

## Getting Started

If you're convinced (or curious), see:

- [Flavors: Server-HTML-HTMX](../flavors/server-html-htmx/README.md) — Implementation guide
- [Templates and Views](../flavors/server-html-htmx/templates.md) — Go template patterns
- [HTMX Integration](../flavors/server-html-htmx/htmx.md) — HTMX patterns for WAFFLE
- [Tailwind Setup](../flavors/server-html-htmx/tailwind.md) — Using Tailwind without Node.js

---

## See Also

- [UI Paradigms](./ui-paradigms.md) — All valid approaches
- [WAFFLE as Substrate](./waffle-as-substrate.md) — WAFFLE's foundational philosophy
- [Systems That Last](./longevity.md) — Durability in software

---

[← Back to Philosophy](./README.md)
