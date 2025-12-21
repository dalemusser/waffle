# WAFFLE Core

*Documentation for the WAFFLE framework foundation — the substrate everything else builds on.*

---

## Overview

This section covers WAFFLE's core functionality: the lifecycle, configuration, routing, server, and infrastructure that every WAFFLE application uses regardless of UI paradigm.

### Application Lifecycle

Every WAFFLE application follows this lifecycle:

```
LoadConfig → ValidateConfig → ConnectDB → EnsureSchema → Startup → BuildHandler → OnReady → Start Server → Shutdown
```

1. **LoadConfig** — Load CoreConfig and AppConfig from files, environment, and flags
2. **ValidateConfig** — Optional hook to validate configuration before proceeding
3. **ConnectDB** — Establish database connections, return DBDeps
4. **EnsureSchema** — Optional hook to run migrations or verify schema
5. **Startup** — Optional hook for pre-server initialization
6. **BuildHandler** — Build the HTTP router with routes and middleware
7. **OnReady** — Optional hook called after handler is built, before server starts
8. **Start Server** — Begin serving HTTP/HTTPS requests
9. **Shutdown** — Optional hook for graceful cleanup on termination

See [Architecture](./architecture.md) for visual diagrams of this lifecycle.

---

## Documents

| Document | Description |
|----------|-------------|
| [**Configuration**](./configuration.md) | CoreConfig, AppConfig, Viper, environment variables |
| [**App Config Keys**](./app-config-keys.md) | Register app-specific config for files, env vars, and CLI flags |
| [**Routing**](./routing.md) | Chi router, BuildHandler, route patterns |
| [**Operational Endpoints**](./operational-endpoints.md) | Health checks, Prometheus metrics, pprof |
| [**Architecture**](./architecture.md) | Visual diagrams of WAFFLE concepts |

---

## Who Should Read This

Everyone building on WAFFLE should understand the core. Start with:

1. [Configuration](./configuration.md) — Learn how to configure your application
2. [Routing](./routing.md) — See how HTTP requests are handled
3. [Operational Endpoints](./operational-endpoints.md) — Health, metrics, and profiling

---

## See Also

- [Philosophy: WAFFLE as Substrate](../philosophy/waffle-as-substrate.md) — Why WAFFLE is designed this way
- [Getting Started](../guides/getting-started/quickstart.md) — Your first WAFFLE application
- [Guides](../guides/README.md) — Task-oriented how-to documentation

---

[← Back to Documentation Index](../waffle-docs.md)
