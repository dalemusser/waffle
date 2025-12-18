# WAFFLE Documentation Index

*A complete, organized directory of all WAFFLE documentation.*

---

## Quick Navigation

| I want to... | Go to... |
|--------------|----------|
| Understand what WAFFLE is | [Philosophy](./philosophy/README.md) |
| Get started quickly | [Getting Started](./guides/getting-started/quickstart.md) |
| Learn the core framework | [Core Documentation](./core/README.md) |
| Find task-specific guides | [Guides](./guides/README.md) |
| Choose a UI approach | [Flavors](./flavors/README.md) |
| Use optional utilities | [Pantry](./pantry/README.md) |
| Look up specific details | [Reference](./reference/README.md) |

---

## Documentation Structure

### [Philosophy](./philosophy/README.md)

Foundational thinking about what WAFFLE is and why it exists.

- [WAFFLE as Substrate](./philosophy/waffle-as-substrate.md) — WAFFLE is not a stack, it's a foundation
- [UI Paradigms](./philosophy/ui-paradigms.md) — The valid ways to build user interfaces
- [Why HTMX + Tailwind](./philosophy/why-htmx-tailwind.md) — The author's recommended approach
- [Systems That Last](./philosophy/longevity.md) — Philosophy of durability
- [Why Go](./philosophy/why-go.md) — Advantages of Go as a foundation

### [Core](./core/README.md)

The WAFFLE framework foundation — concepts that apply regardless of UI paradigm.

- [Configuration](./core/configuration.md) — CoreConfig, AppConfig, Viper
- [Routing](./core/routing.md) — Chi router, middleware, route patterns
- [Architecture](./core/architecture.md) — Visual diagrams and system design
- [Operational Endpoints](./core/operational-endpoints.md) — Health, metrics, pprof

### [Guides](./guides/README.md)

Task-oriented how-to documentation.

**Getting Started**
- [Quickstart](./guides/getting-started/quickstart.md) — Fastest path to productivity
- [First Service](./guides/getting-started/first-service.md) — Step-by-step tutorial
- [makewaffle CLI](./guides/getting-started/makewaffle.md) — Project scaffolding
- [Setting PATH](./guides/getting-started/set-path.md) — Environment setup

**Patterns**
- [Handlers](./guides/patterns/handlers.md) — Handler structure patterns
- [Routes](./guides/patterns/routes.md) — Routing patterns
- [Middleware](./guides/patterns/middleware.md) — Middleware patterns
- [Features](./guides/patterns/features.md) — Feature organization
- [AppConfig](./guides/patterns/appconfig.md) — Configuration patterns

**Databases**
- [MongoDB](./guides/databases/mongo.md)
- [PostgreSQL](./guides/databases/postgres.md)
- [PostgreSQL with pgxpool](./guides/databases/postgres-pgxpool.md)
- [MySQL](./guides/databases/mysql.md)
- [SQLite](./guides/databases/sqlite.md)
- [Redis](./guides/databases/redis.md)

**Authentication**
- [OAuth2](./guides/auth/oauth2.md) — Google, GitHub, etc.
- [Auth Providers](./guides/auth/providers/README.md) — Provider-specific guides

**APIs**
- [CORS](./guides/apis/cors.md) — Cross-origin configuration

**Deployment**
- [Windows Service](./guides/deployment/windows-service.md)

**File Serving**
- [Static Files (Embedded)](./guides/fileserving/static-files.md) — Using `go:embed`
- [Static Files (Filesystem)](./guides/fileserving/static-files-filesystem.md) — Pre-compressed file support

**Development**
- [Overview](./guides/development/README.md) — Architecture and development concepts
- [Lifecycle](./guides/development/lifecycle.md) — Application lifecycle and hooks
- [Configuration](./guides/development/configuration.md) — CoreConfig and AppConfig
- [Logging](./guides/development/logging.md) — Structured logging with zap
- [Routing](./guides/development/routing.md) — Router and middleware
- [Health Checks](./guides/development/health-checks.md) — Health check framework
- [Project Structure](./guides/development/project-structure.md) — Repository and app organization

### [Flavors](./flavors/README.md)

UI paradigm choices — different ways to serve your WAFFLE.

- [Server HTML + HTMX](./flavors/server-html-htmx/README.md) — **Recommended** reference implementation
- [Server HTML + Vanilla JS](./flavors/server-html-vanilla/README.md) — Progressive enhancement without HTMX
- [SPA + WAFFLE API](./flavors/spa-backend/README.md) — React, Vue, or Svelte frontend
- [Hybrid Islands](./flavors/hybrid-islands/README.md) — Server HTML shell + SPA components
- [Native Clients](./flavors/native-clients/README.md) — iOS, Android, desktop apps

### [Pantry](./pantry/README.md)

Optional, paradigm-agnostic utilities — browse by category or use the [alphabetical index](./pantry/alphabetical-index.md).

### [Reference](./reference/README.md)

Quick lookup tables and API documentation.

- [Configuration Variables](./reference/config-vars.md) — All config options
- [File Reference](./reference/file-reference.md) — Generated file documentation

---

## Document Map

```
docs/
├── philosophy/           # Foundational thinking
│   ├── waffle-as-substrate.md
│   ├── ui-paradigms.md
│   ├── why-htmx-tailwind.md
│   ├── longevity.md
│   └── why-go.md
│
├── core/                 # Framework foundation
│   ├── configuration.md
│   ├── routing.md
│   ├── architecture.md
│   └── operational-endpoints.md
│
├── guides/               # Task-oriented how-tos
│   ├── getting-started/
│   ├── patterns/
│   ├── databases/
│   ├── auth/
│   ├── apis/
│   ├── deployment/
│   ├── fileserving/
│   ├── development/
│   └── documentation/
│
├── flavors/              # UI paradigm choices
│   ├── server-html-htmx/
│   ├── server-html-vanilla/
│   ├── spa-backend/
│   ├── hybrid-islands/
│   └── native-clients/
│
├── pantry/               # Optional utilities index
│   ├── README.md
│   └── alphabetical-index.md
│
├── reference/            # Quick lookup
│   ├── config-vars.md
│   └── file-reference.md
│
└── to-do/                # Future work tracking
    ├── docs-to-do.md
    └── customized-makewaffle.md
```

---

## How to Use This Documentation

| Audience | Start Here |
|----------|------------|
| New to WAFFLE | [First Service Tutorial](./guides/getting-started/first-service.md) |
| Experienced developers | [Quickstart](./guides/getting-started/quickstart.md) |
| Choosing UI approach | [UI Paradigms](./philosophy/ui-paradigms.md) → [Flavors](./flavors/README.md) |
| Integrating databases | [Databases Guide](./guides/databases/README.md) |
| Understanding design | [Philosophy](./philosophy/README.md) |

---

## See Also

- [Documentation Guidelines](./guides/documentation/writing-docs.md) — How to write WAFFLE docs
- [Documentation To-Do](./to-do/docs-to-do.md) — Planned documentation
