# rabbitmq

RabbitMQ connection utilities for WAFFLE applications.

## Overview

The `rabbitmq` package provides connection helpers for RabbitMQ with timeout-bounded connections and convenience methods for common messaging patterns.

## Import

```go
import "github.com/dalemusser/waffle/mq/rabbitmq"
```

---

## Connect

**Location:** `rabbitmq.go`

```go
func Connect(url string, timeout time.Duration) (*Connection, error)
```

Opens a RabbitMQ connection using the given URL. Applies a timeout to the connection attempt.

**URL formats:**

```
amqp://user:pass@host:port/vhost
amqp://guest:guest@localhost:5672/
amqps://user:pass@host:port/vhost  (TLS)
```

**Example:**

```go
conn, err := rabbitmq.Connect("amqp://guest:guest@localhost:5672/", 10*time.Second)
if err != nil {
    return err
}
defer conn.Close()
```

---

## ConnectWithConfig

**Location:** `rabbitmq.go`

```go
func ConnectWithConfig(url string, config amqp.Config, timeout time.Duration) (*Connection, error)
```

Opens a RabbitMQ connection with custom configuration for TLS, heartbeat, or other advanced settings.

**Example:**

```go
conn, err := rabbitmq.ConnectWithConfig(url, amqp.Config{
    Heartbeat: 10 * time.Second,
    Locale:    "en_US",
}, core.DBConnectTimeout)
```

---

## Channel

**Location:** `rabbitmq.go`

```go
func (c *Connection) Channel() (*Channel, error)
```

Opens a new channel on the connection. Channels are lightweight and can be created/destroyed frequently.

**Example:**

```go
ch, err := conn.Channel()
if err != nil {
    return err
}
defer ch.Close()
```

---

## DeclareQueue

**Location:** `rabbitmq.go`

```go
func (ch *Channel) DeclareQueue(name string) (amqp.Queue, error)
```

Declares a durable queue with defaults suitable for reliable messaging. The queue survives broker restarts.

**Example:**

```go
q, err := ch.DeclareQueue("tasks")
if err != nil {
    return err
}
fmt.Printf("Queue has %d messages\n", q.Messages)
```

---

## DeclareQueueWithOptions

**Location:** `rabbitmq.go`

```go
func (ch *Channel) DeclareQueueWithOptions(name string, opts QueueOptions) (amqp.Queue, error)
```

Declares a queue with custom options.

**Example:**

```go
// Transient queue for temporary use
q, err := ch.DeclareQueueWithOptions("temp-queue", rabbitmq.TransientQueueOptions())

// Queue with dead letter exchange
q, err := ch.DeclareQueueWithOptions("tasks", rabbitmq.QueueOptions{
    Durable: true,
    Args: amqp.Table{
        "x-dead-letter-exchange": "dlx",
        "x-message-ttl":          int32(86400000), // 24 hours
    },
})
```

---

## QueueOptions

**Location:** `rabbitmq.go`

```go
type QueueOptions struct {
    Durable    bool       // Survives broker restart (default: true)
    AutoDelete bool       // Removed when last consumer disconnects (default: false)
    Exclusive  bool       // Only accessible by declaring connection (default: false)
    NoWait     bool       // Don't wait for server confirmation (default: false)
    Args       amqp.Table // Additional arguments (TTL, DLX, etc.)
}
```

---

## DeclareExchange

**Location:** `rabbitmq.go`

```go
func (ch *Channel) DeclareExchange(name, kind string) error
```

Declares a durable exchange. Kind must be one of: `direct`, `fanout`, `topic`, `headers`.

**Example:**

```go
// Direct exchange for point-to-point messaging
err := ch.DeclareExchange("notifications", rabbitmq.ExchangeDirect)

// Fanout exchange for broadcasting
err := ch.DeclareExchange("events", rabbitmq.ExchangeFanout)

// Topic exchange for pattern-based routing
err := ch.DeclareExchange("logs", rabbitmq.ExchangeTopic)
```

---

## BindQueue

**Location:** `rabbitmq.go`

```go
func (ch *Channel) BindQueue(queue, routingKey, exchange string) error
```

Binds a queue to an exchange with a routing key.

**Example:**

```go
// Bind to direct exchange
err := ch.BindQueue("email-tasks", "email", "notifications")

// Bind to topic exchange with pattern
err := ch.BindQueue("error-logs", "*.error", "logs")
err := ch.BindQueue("all-logs", "#", "logs")
```

---

## Publish

**Location:** `rabbitmq.go`

```go
func (ch *Channel) Publish(ctx context.Context, exchange, routingKey string, body []byte) error
```

Publishes a persistent message to an exchange.

**Example:**

```go
err := ch.Publish(ctx, "tasks", "email", []byte("send welcome email"))
```

---

## PublishJSON

**Location:** `rabbitmq.go`

```go
func (ch *Channel) PublishJSON(ctx context.Context, exchange, routingKey string, body []byte) error
```

Publishes a persistent JSON message with appropriate content type.

**Example:**

```go
task := map[string]string{"to": "user@example.com", "template": "welcome"}
body, _ := json.Marshal(task)
err := ch.PublishJSON(ctx, "tasks", "email", body)
```

---

## PublishWithOptions

**Location:** `rabbitmq.go`

```go
func (ch *Channel) PublishWithOptions(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error
```

Publishes a message with full control over publishing options.

**Example:**

```go
err := ch.PublishWithOptions(ctx, "tasks", "email", amqp.Publishing{
    DeliveryMode:  amqp.Persistent,
    ContentType:   "application/json",
    Body:          body,
    Expiration:    "3600000", // 1 hour TTL
    CorrelationId: requestID,
    Headers: amqp.Table{
        "retry-count": int32(0),
    },
})
```

---

## Consume

**Location:** `rabbitmq.go`

```go
func (ch *Channel) Consume(queue, consumer string) (<-chan amqp.Delivery, error)
```

Starts consuming messages with auto-acknowledgment. Returns a channel of deliveries.

**Example:**

```go
msgs, err := ch.Consume("tasks", "worker-1")
if err != nil {
    return err
}

for msg := range msgs {
    processTask(msg.Body)
}
```

---

## ConsumeWithOptions

**Location:** `rabbitmq.go`

```go
func (ch *Channel) ConsumeWithOptions(queue, consumer string, opts ConsumeOptions) (<-chan amqp.Delivery, error)
```

Starts consuming with custom options. Use `ReliableConsumeOptions()` for manual acknowledgment.

**Example:**

```go
// Manual acknowledgment for reliable processing
msgs, err := ch.ConsumeWithOptions("tasks", "worker-1", rabbitmq.ReliableConsumeOptions())
if err != nil {
    return err
}

for msg := range msgs {
    if err := processTask(msg.Body); err != nil {
        msg.Nack(false, true) // Requeue on failure
        continue
    }
    msg.Ack(false)
}
```

---

## ConsumeOptions

**Location:** `rabbitmq.go`

```go
type ConsumeOptions struct {
    AutoAck   bool       // Auto-acknowledge messages (default: true)
    Exclusive bool       // Only this consumer can access queue (default: false)
    NoLocal   bool       // Don't receive own messages (default: false)
    NoWait    bool       // Don't wait for server confirmation (default: false)
    Args      amqp.Table // Additional arguments
}
```

---

## SetQos

**Location:** `rabbitmq.go`

```go
func (ch *Channel) SetQos(prefetchCount int) error
```

Sets the prefetch count (how many unacknowledged messages the server sends). Essential for manual-ack consumers to prevent overwhelming workers.

**Example:**

```go
// Process up to 10 messages at a time
err := ch.SetQos(10)
```

---

## HealthCheck

**Location:** `rabbitmq.go`

```go
func HealthCheck(conn *Connection) func(ctx context.Context) error
```

Returns a health check function compatible with the health package.

**Example:**

```go
health.Mount(r, map[string]health.Check{
    "rabbitmq": rabbitmq.HealthCheck(conn),
}, logger)
```

---

## Exchange Constants

**Location:** `rabbitmq.go`

```go
const (
    ExchangeDirect  = "direct"  // Routes to queues by exact routing key match
    ExchangeFanout  = "fanout"  // Broadcasts to all bound queues
    ExchangeTopic   = "topic"   // Routes by routing key pattern (* = one word, # = zero or more)
    ExchangeHeaders = "headers" // Routes by message headers
)
```

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
type DBDeps struct {
    Pool     *pgxpool.Pool
    RabbitMQ *rabbitmq.Connection
}

func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.PostgresURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("postgres: %w", err)
    }

    rmq, err := rabbitmq.Connect(appCfg.RabbitMQURL, core.DBConnectTimeout)
    if err != nil {
        pool.Close()
        return DBDeps{}, fmt.Errorf("rabbitmq: %w", err)
    }

    logger.Info("connected to PostgreSQL and RabbitMQ")

    return DBDeps{
        Pool:     pool,
        RabbitMQ: rmq,
    }, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    var errs []error

    if db.RabbitMQ != nil {
        if err := db.RabbitMQ.Close(); err != nil {
            errs = append(errs, fmt.Errorf("rabbitmq close: %w", err))
        }
        logger.Info("disconnected from RabbitMQ")
    }

    if db.Pool != nil {
        db.Pool.Close()
        logger.Info("disconnected from PostgreSQL")
    }

    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    RabbitMQURL string `conf:"rabbitmq_url"`
}
```

```bash
# Environment variables
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

---

## Common Patterns

### Work Queue (Task Distribution)

```go
// Producer
ch, _ := conn.Channel()
ch.DeclareQueue("tasks")

for _, task := range tasks {
    ch.Publish(ctx, "", "tasks", task)
}

// Consumer (run multiple instances)
ch, _ := conn.Channel()
ch.SetQos(1) // Fair dispatch
msgs, _ := ch.ConsumeWithOptions("tasks", "", rabbitmq.ReliableConsumeOptions())

for msg := range msgs {
    processTask(msg.Body)
    msg.Ack(false)
}
```

### Pub/Sub (Broadcasting)

```go
// Publisher
ch, _ := conn.Channel()
ch.DeclareExchange("events", rabbitmq.ExchangeFanout)

ch.Publish(ctx, "events", "", []byte("user.created"))

// Subscriber (each gets all messages)
ch, _ := conn.Channel()
q, _ := ch.DeclareQueueWithOptions("", rabbitmq.QueueOptions{
    Exclusive: true, // Anonymous queue
})
ch.BindQueue(q.Name, "", "events")

msgs, _ := ch.Consume(q.Name, "")
for msg := range msgs {
    handleEvent(msg.Body)
}
```

### Topic Routing

```go
// Publisher
ch, _ := conn.Channel()
ch.DeclareExchange("logs", rabbitmq.ExchangeTopic)

ch.Publish(ctx, "logs", "app.error", []byte("database connection failed"))
ch.Publish(ctx, "logs", "app.info", []byte("user logged in"))

// Error subscriber (receives *.error)
ch.BindQueue("errors", "*.error", "logs")

// All logs subscriber (receives #)
ch.BindQueue("all-logs", "#", "logs")
```

### Dead Letter Queue

```go
// Declare DLX and DLQ
ch.DeclareExchange("dlx", rabbitmq.ExchangeDirect)
ch.DeclareQueueWithOptions("dlq", rabbitmq.QueueOptions{Durable: true})
ch.BindQueue("dlq", "failed", "dlx")

// Main queue with DLX
ch.DeclareQueueWithOptions("tasks", rabbitmq.QueueOptions{
    Durable: true,
    Args: amqp.Table{
        "x-dead-letter-exchange":    "dlx",
        "x-dead-letter-routing-key": "failed",
    },
})

// Consumer rejects bad messages
for msg := range msgs {
    if err := processTask(msg.Body); err != nil {
        msg.Nack(false, false) // Don't requeue, goes to DLQ
        continue
    }
    msg.Ack(false)
}
```

---

## Best Practices

**Connection Management:**
- **Reuse connections** — Create one connection per application, not per request
- **Use multiple channels** — Channels are lightweight; create per-goroutine if needed
- **Handle disconnects** — Monitor `conn.NotifyClose()` for reconnection logic

**Reliable Messaging:**
- **Use manual acknowledgment** — `ReliableConsumeOptions()` for important tasks
- **Set prefetch count** — Prevents overwhelming workers with `SetQos()`
- **Use persistent messages** — Default for `Publish()` and `PublishJSON()`
- **Durable queues and exchanges** — Default for `DeclareQueue()` and `DeclareExchange()`

**Error Handling:**
- **Dead letter exchanges** — Route failed messages for later inspection
- **Message TTL** — Prevent queue buildup with `x-message-ttl`
- **Retry with backoff** — Use message headers to track retry count

---

## See Also

- [mq](../mq.md) — Message queue package index
- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
