# Development Guide

*Understanding WAFFLE's architecture and building applications with it.*

---

## Overview

WAFFLE (Web Application Framework for Flavorful Layered Engineering) is a Go framework that standardizes the infrastructure layer of web services:

- Application lifecycle management
- Structured logging
- Configuration
- Metrics
- HTTP/HTTPS server with Let's Encrypt
- Middleware
- Graceful shutdown
- Health checks

WAFFLE is **not** a monolithic server. It provides the foundation so your application can focus on domain logic.

---

## Architecture

A WAFFLE service is structured into layers:

```
┌──────────────────────────┐
│     Application Code     │
└──────────────┬───────────┘
               │
┌──────────────┴───────────┐
│        WAFFLE Core       │
└──────────────┬───────────┘
               │
┌──────────────┴────────────┐
│       WAFFLE Pantry       │
└───────────────────────────┘
```

- **Core** — The cross-platform framework for building WAFFLE apps
- **Pantry** — Optional, reusable helpers (databases, email, validation, etc.)
- **App** — Your domain logic

---

## Documentation

| Document | Description |
|----------|-------------|
| [**Lifecycle**](./lifecycle.md) | Application lifecycle, hooks, server startup, and graceful shutdown |
| [**Configuration**](./configuration.md) | CoreConfig, AppConfig, and configuration loading |
| [**Logging**](./logging.md) | Structured logging with zap |
| [**Routing**](./routing.md) | Chi router, middleware stack, and handler patterns |
| [**Health Checks**](./health-checks.md) | Health check framework and load balancer support |
| [**Project Structure**](./project-structure.md) | Repository layout and application organization |

---

## Quick Reference

### Starting a New Project

```bash
# Install the CLI
go install github.com/dalemusser/waffle/cmd/makewaffle@latest

# Create a new project
makewaffle new myservice --module github.com/you/myservice

# Run it
cd myservice && go mod tidy && go run ./cmd/myservice
```

### Core Imports

```go
"github.com/dalemusser/waffle/app"
"github.com/dalemusser/waffle/config"
"github.com/dalemusser/waffle/router"
"github.com/dalemusser/waffle/middleware"
```

### Pantry Imports

```go
"github.com/dalemusser/waffle/pantry/db/postgres"
"github.com/dalemusser/waffle/pantry/auth/jwt"
"github.com/dalemusser/waffle/pantry/session"
```

---

## Philosophy

WAFFLE believes in:

- Go clean
- Go fast
- Go layered
- Go composable
- Go build
- Go ship
- Go again

WAFFLE removes the boilerplate so your applications stay flavorful, focused, and maintainable.

---

## See Also

- [Getting Started](../getting-started/README.md) — Quickstart and first service tutorials
- [Patterns](../patterns/README.md) — Handler, routing, and middleware patterns
- [Core Documentation](../../core/README.md) — Framework foundation
- [Pantry](../../pantry/README.md) — Optional utility packages

---

[← Back to Guides](../README.md)
