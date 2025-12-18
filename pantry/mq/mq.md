# mq

Message queue utilities for WAFFLE applications.

## Overview

The `mq` package provides connection helpers for message queues with timeout-bounded connections and convenience methods for common messaging patterns.

## Packages

| Package | Service | Description |
|---------|---------|-------------|
| [rabbitmq](rabbitmq/rabbitmq.md) | RabbitMQ | AMQP messaging with exchanges, queues, and routing |
| [sqs](sqs/sqs.md) | Amazon SQS | AWS managed queue service with FIFO support |

## Import

```go
import "github.com/dalemusser/waffle/mq/rabbitmq"
import "github.com/dalemusser/waffle/mq/sqs"
```

## Common Patterns

Both message queue packages support common messaging patterns:

- **Work queues** — Distribute tasks among workers
- **Dead letter queues** — Handle failed messages
- **Reliable delivery** — Acknowledgment-based processing

## Choosing a Message Queue

| Use Case | Recommended |
|----------|-------------|
| Self-hosted, full AMQP features | RabbitMQ |
| AWS infrastructure, managed service | SQS |
| Complex routing (topics, headers) | RabbitMQ |
| Simple queue with DLQ | Either |
| Strict ordering required | SQS FIFO or RabbitMQ |

## See Also

- [db](../db/db.md) — Database connection utilities
- [app](../app/app.md) — Application lifecycle hooks
- [config](../config/config.md) — Core configuration including `DBConnectTimeout`
