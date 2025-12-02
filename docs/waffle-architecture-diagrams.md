

# WAFFLE Architecture Diagrams  
*A unified, linkable collection of diagrams illustrating core WAFFLE concepts.*

This file exists as the **central diagram index** for all WAFFLE documentation.  
Each diagram has a stable heading so other docs can link directly to it using GitHub’s auto‑generated anchors.

Diagrams are provided in two formats:

- **Mermaid diagrams** (rendered visually on GitHub)  
- **ASCII equivalents** (for plaintext readers, AI systems, and terminals)

Use these diagrams for:

- linking from other docs (e.g., Quickstart, Developer Docs, How‑To)
- teaching materials
- architectural overviews
- wafflestudio.ai documentation pages

---

# 🟦 WAFFLE Lifecycle  
*(How WAFFLE boots, connects, and serves your app)*

## Mermaid
```mermaid
flowchart TD
    A[LoadConfig] --> B[ConnectDB<br/>produce DBDeps]
    B --> C[EnsureSchema<br/>optional]
    C --> D[BuildHandler<br/>router + middleware]
    D --> E[Start HTTP/HTTPS Server]
```

## ASCII
```
LoadConfig
    ↓
ConnectDB (produces DBDeps)
    ↓
EnsureSchema (optional)
    ↓
BuildHandler (construct routes + middleware)
    ↓
Start HTTP/HTTPS Server
```

---

# 🟩 Configuration Flow  
*(How config values move into CoreConfig and AppConfig)*

## Mermaid
```mermaid
flowchart LR
    A[config.toml / yaml / json] --> D[Viper Loader]
    B[Environment Variables<br/>WAFFLE_*] --> D
    C[CLI Flags] --> D
    D --> E[CoreConfig]
    D --> F[AppConfig]
```

## ASCII
```
config file ─┐
env vars ────┼──→ Viper Loader → CoreConfig
CLI flags ───┘                  → AppConfig
```

---

# 🟧 Handler / Routes / BuildHandler Relationship  
*(How WAFFLE wires HTTP behavior together)*

## Mermaid
```mermaid
flowchart LR
    A["AppConfig + DBDeps + Logger"] --> B["Feature Handler"]
    B --> C["Feature Routes(h)"]
    C --> D["BuildHandler mounts subrouters"]
    D --> E["chi.Router"]
```

## ASCII
```
AppConfig + DBDeps + Logger
            ↓
        Handler
            ↓
     Routes(h *Handler)
            ↓
     Mounted in BuildHandler
            ↓
         chi.Router
```

---

# 🟨 Feature Folder Structure  
*(Recommended WAFFLE feature-based organization)*

## Mermaid
```mermaid
flowchart TD
    A["features/xyz"] --> B["handler.go"]
    A --> C["routes.go"]
    A --> D["templates/ (optional)"]
    A --> E["service.go (optional)"]
```

## ASCII
```
internal/app/features/xyz/
    handler.go
    routes.go
    templates/       (optional)
    service.go       (optional)
```

---

# 🟪 Request Flow Through WAFFLE  
*(What happens when an HTTP request arrives)*

## Mermaid
```mermaid
flowchart TD
    A[Incoming Request] --> B[chi Router]
    B --> C[Middleware Stack]
    C --> D[Feature Router]
    D --> E[Handler Method]
    E --> F[Response Written]
```

## ASCII
```
Incoming Request
      ↓
   chi Router
      ↓
Middleware Stack
      ↓
Feature Subrouter
      ↓
Handler Method
      ↓
Response
```

---

# 🟥 TLS / HTTPS / ACME Flow  
*(How WAFFLE manages HTTPS and Let’s Encrypt)*

## Mermaid
```mermaid
flowchart TD
    A["use_https = true"] --> B{"use_lets_encrypt?"}
    B -- "yes" --> C["ACME http-01 or dns-01"]
    C --> D["Certificate cache"]
    B -- "no" --> E["Manual cert_file and key_file"]
    D --> F["HTTPS server on https_port"]
    E --> F
```

## ASCII
```
use_https = true
        ↓
   use_lets_encrypt?
       ↙        ↘
   yes           no
   ↓             ↓
ACME client   Manual cert_file/key_file
   ↓             ↓
Certificate Cache
        ↓
   HTTPS Server
```

---

# 🟫 WAFFLE Toolkit Integration  
*(Where optional helpers plug into the architecture)*

## Mermaid
```mermaid
flowchart TD
    A[Handler + Routes] --> B[BuildHandler]
    B --> C[chi Router]
    C --> D[Middleware Chain]
    D --> E[Toolkit Modules<br/>CORS, WindowsService]
    E --> F[HTTP Server]
```

## ASCII
```
Handlers & Routes
       ↓
   BuildHandler
       ↓
    chi Router
       ↓
 Middleware Chain
       ↓
 Toolkit (CORS, Windows services, etc.)
       ↓
   HTTP Server
```

---

# 🟦 WAFFLE Project Structure (Scaffolded)

## Mermaid
```mermaid
flowchart TD
    A[cmd/appname<br/>main.go] --> B[internal/app/bootstrap]
    B --> C[appconfig.go]
    B --> D[dbdeps.go]
    B --> E[hooks.go]
    A --> F[internal/app/features/...]
    A --> G[internal/app/policy/...]
    A --> H[internal/app/store/...]
    A --> I[internal/domain/models/...]
```

## ASCII
```
cmd/appname/main.go
internal/
  app/
    bootstrap/
      appconfig.go
      dbdeps.go
      hooks.go
    features/
    policy/
    store/
  domain/
    models/
```

---

# 🧇 Linking to Diagrams from Other Docs

You may deep‑link to any diagram heading using GitHub’s anchor format, e.g.:

```markdown
See the [WAFFLE Lifecycle](../waffle-architecture-diagrams.md#waffle-lifecycle) diagram.
```

Each heading in this file is intentionally stable and anchor-safe.

---

# ⭐ Summary

This document gathers **all WAFFLE architecture diagrams in one place** and provides:

- Mermaid diagrams for GitHub rendering  
- ASCII diagrams for plaintext readers  
- Stable anchors for deep-linking  
- A shared visual vocabulary for the framework  

Use these diagrams to enhance clarity across the entire WAFFLE documentation set.