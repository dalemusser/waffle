# WAFFLE Pantry

*Pre-built utilities for common application needs — use what you need, ignore the rest.*

---

## What is the Pantry?

The Pantry is WAFFLE's collection of optional utility packages. Unlike core WAFFLE (app, config, router), Pantry packages are opt-in — import only what your application needs.

Each package:
- Works independently (minimal inter-package dependencies)
- Follows WAFFLE's explicit, debuggable philosophy
- Includes documentation in the package directory
- Integrates cleanly with WAFFLE's lifecycle hooks

---

## Quick Navigation

- **[Alphabetical Index](./alphabetical-index.md)** — Find packages by name
- **[Cross-cutting Guides](#cross-cutting-guides)** — How to choose between options

---

## Packages by Category

### Authentication & Sessions

Secure your application with OAuth2, API keys, JWT tokens, or session management.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **auth** | Authentication umbrella (OAuth2 + API key) | [auth.md](../../pantry/auth/auth.md) |
| **auth/jwt** | JWT token creation and validation | [jwt.md](../../pantry/auth/jwt/jwt.md) |
| **auth/oauth2** | OAuth2 with 20+ providers (Google, GitHub, education, enterprise) | [auth.md](../../pantry/auth/auth.md#oauth2) |
| **auth/apikey** | Static API key middleware | [auth.md](../../pantry/auth/auth.md#api-key) |
| **session** | Server-side session management | [session.md](../../pantry/session/session.md) |

---

### Databases & Data Stores

Connect to databases with timeout-bounded connections and health check integration.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **db** | Database connection overview | [db.md](../../pantry/db/db.md) |
| **db/postgres** | PostgreSQL with pgx connection pooling | [postgres.md](../../pantry/db/postgres/postgres.md) |
| **db/mysql** | MySQL/MariaDB connections | [mysql.md](../../pantry/db/mysql/mysql.md) |
| **db/sqlite** | Embedded SQLite with WAL mode | [sqlite.md](../../pantry/db/sqlite/sqlite.md) |
| **db/mongo** | MongoDB/DocumentDB with replica set support | [mongo.md](../../pantry/db/mongo/mongo.md) |
| **db/redis** | Redis with cluster/sentinel support | [redis.md](../../pantry/db/redis/redis.md) |
| **db/oracle** | Oracle database integration guide | [oracle.md](../../pantry/db/oracle/oracle.md) |
| **mongo** | MongoDB query utilities and helpers | [mongo.md](../../pantry/mongo/mongo.md) |

---

### Caching & Performance

Improve response times and reduce database load.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **cache** | In-memory and Redis caching | [cache.md](../../pantry/cache/cache.md) |
| **ratelimit** | Request rate limiting | [ratelimit.md](../../pantry/ratelimit/ratelimit.md) |

---

### Communication & Notifications

Send emails, push notifications, and real-time updates.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **email** | SMTP email sending | [email.md](../../pantry/email/email.md) |
| **notify** | Web Push notifications (VAPID) | [notify.md](../../pantry/notify/notify.md) |
| **apns** | Apple Push Notification Service | [apns.md](../../pantry/apns/apns.md) |
| **fcm** | Firebase Cloud Messaging | [fcm.md](../../pantry/fcm/fcm.md) |
| **webhook** | Outgoing webhook delivery | [webhook.md](../../pantry/webhook/webhook.md) |
| **sse** | Server-Sent Events | [sse.md](../../pantry/sse/sse.md) |
| **websocket** | WebSocket connections | [websocket.md](../../pantry/websocket/websocket.md) |

---

### Background Processing

Run tasks asynchronously and process message queues.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **jobs** | Background job processing | [jobs.md](../../pantry/jobs/jobs.md) |
| **mq** | Message queue overview | [mq.md](../../pantry/mq/mq.md) |
| **mq/rabbitmq** | RabbitMQ integration | [rabbitmq.md](../../pantry/mq/rabbitmq/rabbitmq.md) |
| **mq/sqs** | AWS SQS integration | [sqs.md](../../pantry/mq/sqs/sqs.md) |
| **retry** | Retry logic with backoff | [retry.md](../../pantry/retry/retry.md) |

---

### Input & Validation

Validate and process user input.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **validate** | Input validation utilities | [validate.md](../../pantry/validate/validate.md) |
| **pagination** | Keyset and offset pagination | [pagination.md](../../pantry/pagination/pagination.md) |

---

### Output & Export

Generate documents and export data.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **pdf** | PDF generation | [pdf.md](../../pantry/pdf/pdf.md) |
| **export** | Data export utilities (CSV, Excel, etc.) | [export.md](../../pantry/export/export.md) |
| **templates** | HTML template utilities | [templates.md](../../pantry/templates/templates.md) |

---

### Security & Cryptography

Encrypt data, hash passwords, and secure your application.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **crypto** | Encryption, hashing, password utilities | [crypto.md](../../pantry/crypto/crypto.md) |

---

### Observability & Debugging

Monitor, audit, and debug your application.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **audit** | Audit logging | [audit.md](../../pantry/audit/audit.md) |
| **health** | Health check utilities | [health.md](../../pantry/health/health.md) |
| **pprof** | Go profiling utilities | [pprof.md](../../pantry/pprof/pprof.md) |
| **requestid** | Request ID generation and propagation | [requestid.md](../../pantry/requestid/requestid.md) |

---

### Internationalization & Localization

Support multiple languages and regions.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **i18n** | Internationalization support | [i18n.md](../../pantry/i18n/i18n.md) |
| **geo** | Geolocation overview | [geo.md](../../pantry/geo/geo.md) |
| **geo/ip** | IP-based geolocation | [ip.md](../../pantry/geo/ip/ip.md) |
| **geo/tz** | Timezone utilities | [tz.md](../../pantry/geo/tz/tz.md) |

---

### HTTP & Web Utilities

Helpers for HTTP handling and web development.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **fileserver** | Static file serving (embedded or filesystem) | [fileserver.md](../../pantry/fileserver/fileserver.md) |
| **httpnav** | HTTP navigation helpers | [httpnav.md](../../pantry/httpnav/httpnav.md) |
| **query** | Query parameter extraction with trimming and limits | [query.md](../../pantry/query/query.md) |
| **urlutil** | URL parsing and manipulation | [urlutil.md](../../pantry/urlutil/urlutil.md) |

---

### Storage & Files

Store and retrieve files locally or in the cloud.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **storage** | File storage abstraction (S3, local, etc.) | [storage.md](../../pantry/storage/storage.md) |
| **search** | Search indexing utilities | [search.md](../../pantry/search/search.md) |

---

### Utilities

General-purpose helpers.

| Package | Description | Documentation |
|---------|-------------|---------------|
| **errors** | Error handling utilities | [errors.md](../../pantry/errors/errors.md) |
| **text** | Text processing utilities | [text.md](../../pantry/text/text.md) |
| **timeout** | Context timeout utilities | [timeout.md](../../pantry/timeout/timeout.md) |
| **version** | Version parsing and comparison | [version.md](../../pantry/version/version.md) |
| **testing** | Test helpers and utilities | [testing.md](../../pantry/testing/testing.md) |
| **feature** | Feature flags | [feature.md](../../pantry/feature/feature.md) |

---

## Import Pattern

All Pantry packages use the same import pattern:

```go
import (
    "github.com/dalemusser/waffle/pantry/email"
    "github.com/dalemusser/waffle/pantry/validate"
    "github.com/dalemusser/waffle/pantry/db/postgres"
    "github.com/dalemusser/waffle/pantry/auth/jwt"
)
```

---

## Cross-cutting Guides

*Coming soon — guides for choosing between similar packages.*

- Choosing an authentication strategy (OAuth2 vs JWT vs API keys vs sessions)
- Choosing a database (PostgreSQL vs MySQL vs SQLite vs MongoDB)
- Notification strategies (Email vs Push vs WebSocket vs SSE)
- Caching patterns (In-memory vs Redis)

---

## See Also

- [Alphabetical Index](./alphabetical-index.md) — Quick package lookup by name
- [Core Documentation](../core/README.md) — WAFFLE's foundation
- [Guides](../guides/README.md) — Task-oriented how-to docs
- [Reference](../reference/README.md) — API reference documentation

---

[← Back to Documentation Index](../waffle-docs.md)
