# sqs

Amazon SQS connection utilities for WAFFLE applications.

## Overview

The `sqs` package provides connection helpers for Amazon Simple Queue Service (SQS) with support for standard queues, FIFO queues, and local development with LocalStack.

## Import

```go
import "github.com/dalemusser/waffle/mq/sqs"
```

---

## Connect

**Location:** `sqs.go`

```go
func Connect(ctx context.Context, region string, timeout time.Duration) (*Client, error)
```

Creates an SQS client using the default AWS credential chain (environment variables, shared credentials file, IAM role, etc.).

**Example:**

```go
client, err := sqs.Connect(ctx, "us-east-1", 10*time.Second)
if err != nil {
    return err
}
```

---

## ConnectWithCredentials

**Location:** `sqs.go`

```go
func ConnectWithCredentials(ctx context.Context, region, accessKey, secretKey string, timeout time.Duration) (*Client, error)
```

Creates an SQS client with explicit credentials. Use for testing or when you can't use the default credential chain.

**Example:**

```go
client, err := sqs.ConnectWithCredentials(ctx, "us-east-1", accessKey, secretKey, 10*time.Second)
```

---

## ConnectWithEndpoint

**Location:** `sqs.go`

```go
func ConnectWithEndpoint(ctx context.Context, region, endpoint string, timeout time.Duration) (*Client, error)
```

Creates an SQS client with a custom endpoint. Use for LocalStack, ElasticMQ, or other SQS-compatible services.

**Example:**

```go
client, err := sqs.ConnectWithEndpoint(ctx, "us-east-1", "http://localhost:4566", 10*time.Second)
```

---

## ConnectLocalStack

**Location:** `sqs.go`

```go
func ConnectLocalStack(ctx context.Context, endpoint string, timeout time.Duration) (*Client, error)
```

Creates an SQS client pre-configured for LocalStack (test credentials, us-east-1 region).

**Example:**

```go
client, err := sqs.ConnectLocalStack(ctx, "", 10*time.Second) // Uses http://localhost:4566
```

---

## GetQueueURL

**Location:** `sqs.go`

```go
func (c *Client) GetQueueURL(ctx context.Context, queueName string) (string, error)
```

Retrieves the URL for a queue by name. Queue URLs are required for most SQS operations.

**Example:**

```go
queueURL, err := client.GetQueueURL(ctx, "my-queue")
```

---

## CreateQueue

**Location:** `sqs.go`

```go
func (c *Client) CreateQueue(ctx context.Context, name string) (string, error)
```

Creates a standard queue with default settings. Returns the queue URL.

**Example:**

```go
queueURL, err := client.CreateQueue(ctx, "tasks")
```

---

## CreateFIFOQueue

**Location:** `sqs.go`

```go
func (c *Client) CreateFIFOQueue(ctx context.Context, name string) (string, error)
```

Creates a FIFO queue with content-based deduplication. FIFO queue names must end with ".fifo".

**Example:**

```go
queueURL, err := client.CreateFIFOQueue(ctx, "orders.fifo")
```

---

## CreateQueueWithOptions

**Location:** `sqs.go`

```go
func (c *Client) CreateQueueWithOptions(ctx context.Context, name string, opts QueueOptions) (string, error)
```

Creates a queue with custom settings.

**Example:**

```go
queueURL, err := client.CreateQueueWithOptions(ctx, "tasks", sqs.QueueOptions{
    VisibilityTimeout:  60 * time.Second,
    MessageRetention:   7 * 24 * time.Hour, // 7 days
    ReceiveWaitTime:    20 * time.Second,   // Long polling
    DeadLetterQueueARN: dlqARN,
    MaxReceiveCount:    3,
})
```

---

## QueueOptions

**Location:** `sqs.go`

```go
type QueueOptions struct {
    VisibilityTimeout    time.Duration // How long messages are hidden after receive (default: 30s)
    MessageRetention     time.Duration // How long messages are kept (default: 4 days)
    MaxMessageSize       int           // Maximum message size in bytes (default: 256 KB)
    ReceiveWaitTime      time.Duration // Long polling wait time (default: 0)
    DelaySeconds         time.Duration // Default delay before messages are visible (default: 0)
    DeadLetterQueueARN   string        // ARN of dead-letter queue
    MaxReceiveCount      int           // Receives before going to DLQ
    FIFO                 bool          // Enable FIFO behavior
    ContentDeduplication bool          // Content-based deduplication for FIFO
    DeduplicationScope   string        // "messageGroup" or "queue" for FIFO
    FIFOThroughputLimit  string        // "perQueue" or "perMessageGroupId" for FIFO
}
```

---

## Send

**Location:** `sqs.go`

```go
func (c *Client) Send(ctx context.Context, queueURL, body string) (string, error)
```

Sends a message to a queue. Returns the message ID.

**Example:**

```go
msgID, err := client.Send(ctx, queueURL, `{"task": "send-email", "to": "user@example.com"}`)
```

---

## SendWithDelay

**Location:** `sqs.go`

```go
func (c *Client) SendWithDelay(ctx context.Context, queueURL, body string, delay time.Duration) (string, error)
```

Sends a message with a delay before it becomes visible (0 to 15 minutes).

**Example:**

```go
msgID, err := client.SendWithDelay(ctx, queueURL, body, 5*time.Minute)
```

---

## SendFIFO

**Location:** `sqs.go`

```go
func (c *Client) SendFIFO(ctx context.Context, queueURL, body, groupID, deduplicationID string) (string, error)
```

Sends a message to a FIFO queue. MessageGroupId is required.

**Example:**

```go
msgID, err := client.SendFIFO(ctx, queueURL, body, "order-123", "")
```

---

## SendWithAttributes

**Location:** `sqs.go`

```go
func (c *Client) SendWithAttributes(ctx context.Context, queueURL, body string, attrs map[string]string) (string, error)
```

Sends a message with custom attributes.

**Example:**

```go
msgID, err := client.SendWithAttributes(ctx, queueURL, body, map[string]string{
    "type":     "email",
    "priority": "high",
})
```

---

## SendBatch

**Location:** `sqs.go`

```go
func (c *Client) SendBatch(ctx context.Context, queueURL string, messages []string) ([]string, []BatchError, error)
```

Sends up to 10 messages in a single request. Returns successful message IDs and any failures.

**Example:**

```go
ids, failures, err := client.SendBatch(ctx, queueURL, []string{msg1, msg2, msg3})
```

---

## Receive

**Location:** `sqs.go`

```go
func (c *Client) Receive(ctx context.Context, queueURL string, maxMessages int, waitTime time.Duration) ([]Message, error)
```

Receives messages from a queue. Returns up to maxMessages (1-10). Uses long polling with the specified wait time.

**Example:**

```go
messages, err := client.Receive(ctx, queueURL, 10, 20*time.Second)
for _, msg := range messages {
    processTask(msg.Body)
    client.Delete(ctx, queueURL, msg.ReceiptHandle)
}
```

---

## Message

**Location:** `sqs.go`

```go
type Message struct {
    ID            string            // Message ID
    Body          string            // Message body
    ReceiptHandle string            // Handle for delete/visibility operations
    Attributes    map[string]string // Message attributes
    MD5           string            // MD5 of body for verification
}
```

---

## Delete

**Location:** `sqs.go`

```go
func (c *Client) Delete(ctx context.Context, queueURL, receiptHandle string) error
```

Deletes a message after successful processing.

**Example:**

```go
err := client.Delete(ctx, queueURL, msg.ReceiptHandle)
```

---

## DeleteBatch

**Location:** `sqs.go`

```go
func (c *Client) DeleteBatch(ctx context.Context, queueURL string, receiptHandles []string) ([]BatchError, error)
```

Deletes up to 10 messages in a single request.

**Example:**

```go
handles := []string{msg1.ReceiptHandle, msg2.ReceiptHandle}
failures, err := client.DeleteBatch(ctx, queueURL, handles)
```

---

## ChangeVisibility

**Location:** `sqs.go`

```go
func (c *Client) ChangeVisibility(ctx context.Context, queueURL, receiptHandle string, timeout time.Duration) error
```

Changes the visibility timeout of a message. Use to extend processing time or release a message back to the queue (timeout = 0).

**Example:**

```go
// Extend processing time
err := client.ChangeVisibility(ctx, queueURL, msg.ReceiptHandle, 60*time.Second)

// Release message back to queue immediately
err := client.ChangeVisibility(ctx, queueURL, msg.ReceiptHandle, 0)
```

---

## Purge

**Location:** `sqs.go`

```go
func (c *Client) Purge(ctx context.Context, queueURL string) error
```

Removes all messages from a queue. Can only be called once every 60 seconds.

---

## DeleteQueue

**Location:** `sqs.go`

```go
func (c *Client) DeleteQueue(ctx context.Context, queueURL string) error
```

Deletes a queue and all its messages.

---

## GetQueueAttributes

**Location:** `sqs.go`

```go
func (c *Client) GetQueueAttributes(ctx context.Context, queueURL string) (map[string]string, error)
```

Retrieves queue attributes (message count, ARN, etc.).

**Example:**

```go
attrs, err := client.GetQueueAttributes(ctx, queueURL)
fmt.Printf("Messages in queue: %s\n", attrs["ApproximateNumberOfMessages"])
```

---

## HealthCheck

**Location:** `sqs.go`

```go
func HealthCheck(client *Client) func(ctx context.Context) error
```

Returns a health check function compatible with the health package.

**Example:**

```go
health.Mount(r, map[string]health.Check{
    "sqs": sqs.HealthCheck(client),
}, logger)
```

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
type DBDeps struct {
    Pool     *pgxpool.Pool
    SQS      *sqs.Client
    QueueURL string
}

func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.PostgresURI, core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("postgres: %w", err)
    }

    sqsClient, err := sqs.Connect(ctx, appCfg.AWSRegion, core.DBConnectTimeout)
    if err != nil {
        pool.Close()
        return DBDeps{}, fmt.Errorf("sqs: %w", err)
    }

    queueURL, err := sqsClient.GetQueueURL(ctx, appCfg.SQSQueueName)
    if err != nil {
        pool.Close()
        return DBDeps{}, fmt.Errorf("sqs queue url: %w", err)
    }

    logger.Info("connected to PostgreSQL and SQS")

    return DBDeps{
        Pool:     pool,
        SQS:      sqsClient,
        QueueURL: queueURL,
    }, nil
}
```

### Shutdown Hook

SQS doesn't require explicit close — AWS SDK handles connection pooling.

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.Pool != nil {
        db.Pool.Close()
        logger.Info("disconnected from PostgreSQL")
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    AWSRegion    string `conf:"aws_region"`
    SQSQueueName string `conf:"sqs_queue_name"`
}
```

```bash
# Environment variables
AWS_REGION=us-east-1
SQS_QUEUE_NAME=my-tasks
AWS_ACCESS_KEY_ID=...      # Or use IAM role
AWS_SECRET_ACCESS_KEY=...
```

---

## Common Patterns

### Work Queue

```go
// Producer
msgID, _ := client.Send(ctx, queueURL, `{"task": "process-order", "order_id": "123"}`)

// Consumer (polling loop)
for {
    messages, err := client.Receive(ctx, queueURL, 10, 20*time.Second)
    if err != nil {
        logger.Error("receive failed", zap.Error(err))
        continue
    }

    for _, msg := range messages {
        if err := processTask(msg.Body); err != nil {
            logger.Error("task failed", zap.Error(err))
            continue
        }
        client.Delete(ctx, queueURL, msg.ReceiptHandle)
    }
}
```

### Dead Letter Queue

```go
// Create DLQ
dlqURL, _ := client.CreateQueue(ctx, "tasks-dlq")
dlqAttrs, _ := client.GetQueueAttributes(ctx, dlqURL)
dlqARN := dlqAttrs["QueueArn"]

// Create main queue with DLQ
queueURL, _ := client.CreateQueueWithOptions(ctx, "tasks", sqs.QueueOptions{
    DeadLetterQueueARN: dlqARN,
    MaxReceiveCount:    3, // Move to DLQ after 3 failed receives
})
```

### FIFO Queue

```go
// Create FIFO queue
queueURL, _ := client.CreateFIFOQueue(ctx, "orders.fifo")

// Send ordered messages
client.SendFIFO(ctx, queueURL, orderJSON, "customer-123", "") // Same customer = ordered
client.SendFIFO(ctx, queueURL, orderJSON, "customer-456", "") // Different customer = parallel
```

### Batch Processing

```go
// Batch send
messages := []string{task1, task2, task3}
ids, failures, _ := client.SendBatch(ctx, queueURL, messages)

// Batch delete
var handles []string
for _, msg := range processedMessages {
    handles = append(handles, msg.ReceiptHandle)
}
failures, _ := client.DeleteBatch(ctx, queueURL, handles)
```

---

## Best Practices

**Polling:**
- **Use long polling** — Set `ReceiveWaitTime` to 20 seconds to reduce costs and latency
- **Batch operations** — Use `SendBatch` and `DeleteBatch` for efficiency
- **Process in batches** — Receive up to 10 messages at a time

**Reliability:**
- **Delete after processing** — Always delete messages after successful processing
- **Extend visibility** — Use `ChangeVisibility` for long-running tasks
- **Dead letter queues** — Configure `MaxReceiveCount` to handle poison messages

**FIFO Queues:**
- **Use message groups** — Group related messages for ordering within the group
- **Enable deduplication** — Prevent duplicate processing with content-based or explicit IDs
- **Name queues correctly** — FIFO queue names must end with `.fifo`

**Cost Optimization:**
- **Long polling** — Reduces empty responses and API calls
- **Batch operations** — Process multiple messages per API call
- **Right-size visibility timeout** — Match to your processing time

---

## See Also

- [mq](../mq.md) — Message queue package index
- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
