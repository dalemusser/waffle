# WAFFLE Documentation

*WAFFLE is a framework for building deliciously structured Go web services.*

---

## What is WAFFLE?

WAFFLE provides the foundational patterns and optional utilities for building Go web services. Rather than a monolithic framework, it's a set of well-integrated pieces:

- **Core patterns** — Configuration, routing, and service architecture
- **Pantry packages** — Opt-in utilities for auth, databases, caching, email, and more
- **Flavors** — Support for any UI approach (server HTML, SPA, native apps)

WAFFLE is opinionated where it matters (project structure, configuration patterns) and flexible where preferences vary (UI paradigm, database choice).

---

## Quick Start

| I want to... | Go to... |
|--------------|----------|
| Get started fast | [Quickstart Guide](./guides/getting-started/quickstart.md) |
| Understand what WAFFLE is | [Philosophy](./philosophy/README.md) |
| Build my first service | [First Service Tutorial](./guides/getting-started/first-service.md) |
| Choose a UI approach | [Flavors](./flavors/README.md) |

---

## Documentation Structure

```
docs/
├── philosophy/     What WAFFLE is and why
├── core/           Framework foundation
├── guides/         Task-oriented how-tos
├── flavors/        UI paradigm choices
├── pantry/         Optional utility packages
├── reference/      Quick lookup tables
└── to-do/          Future documentation plans
```

### [Philosophy](./philosophy/README.md)

Foundational thinking: WAFFLE as substrate, UI paradigms, why Go, why HTMX.

### [Core](./core/README.md)

The WAFFLE foundation: configuration, routing, architecture diagrams.

### [Guides](./guides/README.md)

How-to documentation: getting started, patterns, databases, authentication.

### [Flavors](./flavors/README.md)

UI paradigm choices: HTMX, vanilla JS, SPA, hybrid islands, native clients.

### [Pantry](./pantry/README.md)

Optional utility packages: authentication, databases, caching, email, and more.

### [Reference](./reference/README.md)

Quick lookup: configuration variables, file reference.

---

## Deep Dive

For a comprehensive index of all documentation with descriptions, see [waffle-docs.md](./waffle-docs.md).

---

## Contributing

See [Documentation Guidelines](./guides/documentation/writing-docs.md) for writing standards.
