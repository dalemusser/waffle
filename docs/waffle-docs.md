# WAFFLE Documentation Index  
*A complete, organized directory of all WAFFLE guides, references, and examples.*

This page serves as the **master list of all documentation** for the WAFFLE framework.  
Use it as your starting point when exploring or contributing to WAFFLE docs.

---

# 📘 Core Guides

### **1. First-Time & Getting Started**
- **[How to Write Your First WAFFLE Service](./first-waffle-service-howto.md)**  
  Step-by-step beginner tutorial that walks you from zero to a running service.

- **[WAFFLE Quickstart Guide](./quickstart-guide.md)**  
  A concise, practical overview for developers who want the fastest path to productivity.

### **2. Core Framework Concepts**
- **[WAFFLE Framework – Developer Documentation](./developer-documentation.md)**  
  Deep dive into WAFFLE’s architecture, lifecycle, configuration, logging, server behavior, and middleware.

- **[WAFFLE Routes & Middleware Guide](./routes-and-middleware-guide.md)**  
  A conceptual + practical guide to routing patterns, subrouters, middleware, and feature composition.

---

# 🧩 Example Library (Recipes)

Focused, practical “how to do exactly this” documents.  
All live under `docs/examples/`.

### **Database Integration**
- **[MongoDB in DBDeps](./examples/dbdeps-mongo.md)**
- **[Postgres in DBDeps (`*sql.DB`)](./examples/dbdeps-postgres.md)**
- **[Postgres with pgxpool (`pgxpool.Pool`)](./examples/dbdeps-postgres-pgxpool.md)**
- **[Redis in DBDeps](./examples/dbdeps-redis.md)**

### **Configuration Patterns**
- **[Examples of AppConfig Patterns](./examples/examples-of-appconfig-patterns.md)**
- **[CORS Examples](./examples/cors-examples.md)**
- **[DBDeps Usage Examples](./examples/dbdeps-usage-examples.md)**
- **[Feature Structure Examples](./examples/feature-structure-examples.md)**
- **[Handler Structure Examples](./examples/handler-structure-examples.md)**
- **[Middleware Examples](./examples/middleware-examples.md)**
- **[Route Examples](./examples/routes-examples.md)**
- **[Windows Service Examples](./examples/windows-service-examples.md)**

### **(Reserved for future examples — see to‑do list)**
- Migrations (`golang-migrate`)
- Multiple databases in DBDeps
- Mocking DBDeps for testing
- Shared route prefixes
- API versioning (`/v1`, `/v2`)
- JSON APIs
- Template rendering
- WebSockets
- High‑security admin panels

---

# 🗂️ Document Map Overview

```
docs/
│
├── developer-documentation.md
├── first-waffle-service-howto.md
├── quickstart-guide.md
├── routes-and-middleware-guide.md
│
├── examples/
│   ├── dbdeps-mongo.md
│   ├── dbdeps-postgres.md
│   ├── dbdeps-postgres-pgxpool.md
│   ├── dbdeps-redis.md
│   ├── dbdeps-usage-examples.md
│   ├── feature-structure-examples.md
│   ├── handler-structure-examples.md
│   ├── middleware-examples.md
│   ├── routes-examples.md
│   ├── windows-service-examples.md
│   └── examples-of-appconfig-patterns.md
│
└── to-do/
    └── to-do.md
```

---

# 🧭 How to Use This Index

- New WAFFLE developers → start with **“How to Write Your First WAFFLE Service”**  
- Returning users → jump to **Quickstart** or **Routes & Middleware Guide**  
- Need to integrate a database? → check **Database Integration** recipes  
- Looking for patterns? → see **Configuration Patterns** and upcoming recipes
