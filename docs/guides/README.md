# WAFFLE Guides

*Task-oriented documentation — how to accomplish specific goals with WAFFLE.*

---

## Overview

This section contains practical guides organized by what you're trying to accomplish. Unlike [Core](../core/README.md) (which explains how WAFFLE works) or [Reference](../reference/README.md) (which lists facts), Guides answer the question: **"How do I do X?"**

These guides are paradigm-agnostic — they work regardless of whether you're building with HTMX, React, or serving a mobile API. For UI-specific guidance, see [Flavors](../flavors/README.md).

---

## Getting Started

| Guide | Description |
|-------|-------------|
| [**Quickstart**](./getting-started/quickstart.md) | Get up and running fast |
| [**First Service**](./getting-started/first-service.md) | Build your first WAFFLE application from scratch |
| [**makewaffle CLI**](./getting-started/makewaffle.md) | Scaffold new projects |
| [**Setting PATH**](./getting-started/set-path.md) | Configure your environment |
| [**Terminal Setup (macOS)**](./getting-started/terminal-macos.md) | Terminal basics for macOS users |
| [**Terminal Setup (Windows)**](./getting-started/terminal-windows.md) | Terminal basics for Windows users |
| [**Terminal Setup (Linux)**](./getting-started/terminal-linux.md) | Terminal basics for Linux users |

---

## Patterns

| Guide | Description |
|-------|-------------|
| [**Handlers**](./patterns/handlers.md) | Handler struct patterns |
| [**Routes**](./patterns/routes.md) | Routing patterns |
| [**Middleware**](./patterns/middleware.md) | Middleware composition |
| [**Features**](./patterns/features.md) | Feature folder structure |
| [**AppConfig**](./patterns/appconfig.md) | Configuration patterns |

---

## Database Integration

| Guide | Description |
|-------|-------------|
| [**Overview**](./databases/README.md) | DBDeps patterns and database philosophy |
| [**MongoDB**](./databases/mongo.md) | Connecting to MongoDB |
| [**PostgreSQL**](./databases/postgres.md) | PostgreSQL with `database/sql` |
| [**PostgreSQL (pgxpool)**](./databases/postgres-pgxpool.md) | PostgreSQL with pgx connection pool |
| [**MySQL**](./databases/mysql.md) | MySQL/MariaDB connections |
| [**SQLite**](./databases/sqlite.md) | SQLite for embedded databases |
| [**Redis**](./databases/redis.md) | Redis connections and patterns |
| [**Usage Examples**](./databases/usage-examples.md) | DBDeps in handlers |

---

## Authentication

| Guide | Description |
|-------|-------------|
| [**OAuth2**](./auth/oauth2.md) | OAuth2 integration overview |
| [**Auth Providers**](./auth/providers/README.md) | Provider-specific setup guides |
| [**Core Concepts**](./auth/providers/core-concepts.md) | Authentication concepts and patterns |

### Popular Providers

| Provider | Description |
|----------|-------------|
| [Google](./auth/providers/google.md) | Google OAuth2 |
| [GitHub](./auth/providers/github.md) | GitHub OAuth2 |
| [Microsoft](./auth/providers/microsoft.md) | Microsoft / Azure AD |
| [Apple](./auth/providers/apple.md) | Sign in with Apple |
| [Clever](./auth/providers/clever.md) | Clever for education |
| [ClassLink](./auth/providers/classlink.md) | ClassLink for education |

See [Auth Providers](./auth/providers/README.md) for the complete list including Okta, Discord, LinkedIn, and education-specific providers.

---

## APIs

| Guide | Description |
|-------|-------------|
| [**CORS**](./apis/cors.md) | Cross-origin resource sharing configuration |

---

## Deployment

| Guide | Description |
|-------|-------------|
| [**Windows Service**](./deployment/windows-service.md) | Running WAFFLE as a Windows service |

---

## File Serving

| Guide | Description |
|-------|-------------|
| [**Overview**](./fileserving/README.md) | Static file serving patterns |
| [**Embedded Files**](./fileserving/static-files.md) | Using `go:embed` for single-binary deployment |
| [**Filesystem Files**](./fileserving/static-files-filesystem.md) | Serving from disk with pre-compression support |

---

## Documentation

| Guide | Description |
|-------|-------------|
| [**Writing Docs**](./documentation/writing-docs.md) | Guidelines for writing WAFFLE documentation |

---

## Development

| Guide | Description |
|-------|-------------|
| [**Overview**](./development/README.md) | Architecture and development concepts |
| [**Lifecycle**](./development/lifecycle.md) | Application lifecycle and hooks |
| [**Configuration**](./development/configuration.md) | CoreConfig and AppConfig |
| [**Logging**](./development/logging.md) | Structured logging with zap |
| [**Routing**](./development/routing.md) | Router and middleware |
| [**Health Checks**](./development/health-checks.md) | Health check framework |
| [**Project Structure**](./development/project-structure.md) | Repository and app organization |

---

## Finding What You Need

- **"How do I get started?"** → [Quickstart](./getting-started/quickstart.md)
- **"How do I scaffold a new project?"** → [makewaffle CLI](./getting-started/makewaffle.md)
- **"How do I connect to a database?"** → [Database Guides](./databases/README.md)
- **"How do I add authentication?"** → [OAuth2](./auth/oauth2.md) or [Auth Providers](./auth/providers/README.md)
- **"How do I configure CORS?"** → [CORS](./apis/cors.md)
- **"How do I structure my features?"** → [Features](./patterns/features.md) and [Handlers](./patterns/handlers.md)
- **"How do I add middleware?"** → [Middleware](./patterns/middleware.md)
- **"How do I build a web UI?"** → See [Flavors](../flavors/README.md) for UI-specific guides
- **"What's the full developer reference?"** → [Development Guide](./development/README.md)

---

## See Also

- [Core Documentation](../core/README.md) — How WAFFLE works
- [Flavors](../flavors/README.md) — UI paradigm-specific guides
- [Reference](../reference/README.md) — Quick lookup tables
- [Pantry](../pantry/README.md) — Utility package documentation

---

[← Back to Documentation Index](../waffle-docs.md)
