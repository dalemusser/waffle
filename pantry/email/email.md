# pantry/email

Simple SMTP email sending for web applications with async queue support.

---

## Overview

The `email` package provides a straightforward interface for sending emails via SMTP. It wraps [github.com/wneessen/go-mail](https://github.com/wneessen/go-mail) with sensible defaults for typical web application use cases like password resets, email verification, and notifications.

**Features:**
- **Sender** — Synchronous SMTP email sending
- **Queue** — Async email delivery with retries and persistence
- **Templates** — HTML and text templates with Go template syntax

---

## Import

```go
import "github.com/dalemusser/waffle/pantry/email"
```

---

## Quick Start

### Synchronous Sending

```go
// Create a sender with SMTP configuration
sender := email.NewSender(email.Config{
    Host:        "smtp.example.com",
    Port:        587,
    Username:    "apikey",
    Password:    "your-api-key",
    FromAddress: "noreply@example.com",
    FromName:    "My App",
})

// Send a simple text email
err := sender.SendSimple(ctx, "user@example.com", "Welcome!", "Thanks for signing up.")

// Send HTML email with text fallback
err := sender.SendHTML(ctx,
    "user@example.com",
    "Password Reset",
    "Click here to reset: https://...",  // plain text fallback
    "<p>Click <a href='https://...'>here</a> to reset.</p>",  // HTML
)
```

### Async Queue

```go
// Create queue with sender and storage
queue := email.NewQueue(email.QueueConfig{
    Sender:  sender,
    Store:   email.NewMemoryQueueStore(),
    Workers: 2,
})

// Start processing
queue.Start()
defer queue.Stop(ctx)

// Queue emails (non-blocking)
id, err := queue.EnqueueSimple(ctx, "user@example.com", "Welcome!", "Thanks!")
```

### With Templates

```go
// Create template store and register templates
store := email.NewTemplateStore()
store.Register(email.WelcomeTemplate)

// Queue with templates
tplQueue := email.NewTemplateQueue(queue, store)
id, err := tplQueue.EnqueueTo(ctx, "user@example.com", "welcome", map[string]any{
    "Name":    "John",
    "AppName": "My App",
})
```

---

## Sender

### Config

SMTP server configuration.

```go
type Config struct {
    Host        string        // SMTP server hostname
    Port        int           // SMTP port (default: 587)
    Username    string        // SMTP auth username
    Password    string        // SMTP auth password
    FromAddress string        // Default sender email
    FromName    string        // Default sender display name (optional)
    UseTLS      bool          // Enable STARTTLS (default: true for port 587)
    UseSSL      bool          // Enable implicit SSL (for port 465)
    Timeout     time.Duration // Operation timeout (default: 30s)
}
```

### Message

An email message to send.

```go
type Message struct {
    To          []string     // Recipient addresses
    Subject     string       // Subject line
    TextBody    string       // Plain text body
    HTMLBody    string       // HTML body
    ReplyTo     string       // Reply-To address (optional)
    Attachments []Attachment // File attachments (optional)
}
```

### Attachment

A file attachment.

```go
type Attachment struct {
    Filename    string // Display filename
    ContentType string // MIME type
    Data        []byte // File contents
}
```

### Sender Methods

```go
sender.Send(ctx, msg Message) error               // Send full message
sender.SendSimple(ctx, to, subject, body) error   // Send text email
sender.SendHTML(ctx, to, subject, text, html) error // Send HTML email
```

---

## Queue

Async email delivery with retries, scheduling, and persistence.

### QueueConfig

```go
type QueueConfig struct {
    Sender       *Sender      // Email sender
    Store        QueueStore   // Storage backend
    Logger       *zap.Logger  // Optional logger
    Workers      int          // Concurrent senders (default: 2)
    PollInterval time.Duration // Check interval (default: 5s)
    OnSent       func(*QueuedEmail)        // Success callback
    OnFailed     func(*QueuedEmail, error) // Failure callback
}
```

### Basic Usage

```go
// Create queue
queue := email.NewQueue(email.QueueConfig{
    Sender:  sender,
    Store:   email.NewMemoryQueueStore(),
    Workers: 2,
    OnSent: func(e *email.QueuedEmail) {
        log.Printf("Email %s sent", e.ID)
    },
    OnFailed: func(e *email.QueuedEmail, err error) {
        log.Printf("Email %s failed: %v", e.ID, err)
    },
})

// Start/stop
queue.Start()
defer queue.Stop(ctx)

// Queue emails
id, _ := queue.EnqueueSimple(ctx, "user@example.com", "Subject", "Body")
id, _ := queue.EnqueueHTML(ctx, "user@example.com", "Subject", "Text", "<p>HTML</p>")
id, _ := queue.EnqueueMessage(ctx, email.Message{...})
```

### QueuedEmail

```go
type QueuedEmail struct {
    ID          string            // Unique tracking ID
    Message     Message           // Email content
    Priority    int               // Higher = more urgent
    ScheduledAt *time.Time        // Scheduled send time
    MaxRetries  int               // Retry attempts (default: 3)
    Metadata    map[string]string // Custom tracking data
    CreatedAt   time.Time
    Attempts    int
    LastError   string
    Status      EmailStatus
}
```

### Email Status

```go
const (
    EmailStatusPending   // Waiting to send
    EmailStatusScheduled // Scheduled for future
    EmailStatusSending   // Currently sending
    EmailStatusSent      // Successfully sent
    EmailStatusFailed    // Failed permanently
)
```

### Scheduling

```go
// Schedule for later
queue.Schedule(ctx, &email.QueuedEmail{
    Message: email.Message{
        To:      []string{"user@example.com"},
        Subject: "Reminder",
        TextBody: "Don't forget!",
    },
}, time.Now().Add(24*time.Hour))
```

### Priority

```go
// High priority email (processed first)
queue.Enqueue(ctx, &email.QueuedEmail{
    Message:  msg,
    Priority: 10, // Higher = more urgent
})
```

### Tracking and Management

```go
// Get email status
queued, _ := queue.Get(ctx, id)
fmt.Println(queued.Status, queued.Attempts)

// Cancel pending email
queue.Cancel(ctx, id)

// Get statistics
stats, _ := queue.Stats(ctx)
fmt.Printf("Pending: %d, Sent: %d, Failed: %d\n",
    stats.Pending, stats.Sent, stats.Failed)
```

### Queue Methods

```go
queue.Start()                                    // Start workers
queue.Stop(ctx) error                            // Graceful shutdown
queue.Enqueue(ctx, *QueuedEmail) error           // Queue email
queue.EnqueueMessage(ctx, Message) (string, error)
queue.EnqueueSimple(ctx, to, subject, body) (string, error)
queue.EnqueueHTML(ctx, to, subject, text, html) (string, error)
queue.Schedule(ctx, *QueuedEmail, at) error      // Schedule for later
queue.Get(ctx, id) (*QueuedEmail, error)         // Get by ID
queue.Cancel(ctx, id) error                      // Cancel pending
queue.Stats(ctx) (*QueueStats, error)            // Get statistics
```

---

## Queue Storage

### Memory Store (Development/Testing)

```go
store := email.NewMemoryQueueStore()

// Cleanup old emails
removed := store.Cleanup(ctx, 24*time.Hour)
```

### Redis Store (Production)

```go
store := email.NewRedisQueueStore(email.RedisQueueConfig{
    Client: redisClient, // Implements RedisQueueClient
    Prefix: "myapp:email:",
})
```

### RedisQueueClient Interface

Implement this with your Redis client:

```go
type RedisQueueClient interface {
    Set(ctx context.Context, key, value string, ttl time.Duration) error
    Get(ctx context.Context, key string) (string, error)
    Del(ctx context.Context, keys ...string) error
    Keys(ctx context.Context, pattern string) ([]string, error)
    ZAdd(ctx context.Context, key string, score float64, member string) error
    ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error)
    ZRem(ctx context.Context, key string, members ...string) error
}
```

---

## Templates

### TemplateStore

```go
// Create store
store := email.NewTemplateStore()

// Register template
store.Register(email.EmailTemplate{
    Name:     "welcome",
    Subject:  "Welcome to {{.AppName}}!",
    TextBody: "Hi {{.Name}},\n\nWelcome!",
    HTMLBody: "<p>Hi {{.Name}},</p><p>Welcome!</p>",
})

// Render to message
msg, _ := store.Render("welcome", map[string]any{
    "Name":    "John",
    "AppName": "My App",
})
msg.To = []string{"john@example.com"}
sender.Send(ctx, *msg)
```

### Built-in Templates

```go
// Register common templates
store.RegisterCommonTemplates()

// Available templates:
// - "welcome" - Welcome email with optional verify link
// - "password_reset" - Password reset with link
// - "email_verification" - Email verification
```

### Template Data

```go
// Welcome template expects:
data := map[string]any{
    "Name":      "John",
    "AppName":   "My App",
    "VerifyURL": "https://...", // optional
}

// Password reset expects:
data := map[string]any{
    "Name":      "John",
    "AppName":   "My App",
    "ResetURL":  "https://...",
    "ExpiresIn": "1 hour",
}

// Email verification expects:
data := map[string]any{
    "Name":      "John",
    "AppName":   "My App",
    "VerifyURL": "https://...",
    "ExpiresIn": "24 hours",
}
```

### TemplateSender

Synchronous sending with templates:

```go
tplSender := email.NewTemplateSender(sender, store)

err := tplSender.SendTo(ctx, "user@example.com", "welcome", map[string]any{
    "Name":    "John",
    "AppName": "My App",
})
```

### TemplateQueue

Async sending with templates:

```go
tplQueue := email.NewTemplateQueue(queue, store)

// Queue templated email
id, _ := tplQueue.EnqueueTo(ctx, "user@example.com", "password_reset", map[string]any{
    "Name":      "John",
    "AppName":   "My App",
    "ResetURL":  resetURL,
    "ExpiresIn": "1 hour",
})

// Schedule templated email
id, _ := tplQueue.Schedule(ctx,
    []string{"user@example.com"},
    "welcome",
    data,
    time.Now().Add(time.Hour),
)
```

### Custom Templates

```go
store.Register(email.EmailTemplate{
    Name:    "order_confirmation",
    Subject: "Order #{{.OrderID}} Confirmed",
    TextBody: `Hi {{.Name}},

Your order #{{.OrderID}} has been confirmed.

Items:
{{range .Items}}- {{.Name}}: ${{.Price}}
{{end}}
Total: ${{.Total}}

Thanks for shopping with us!`,
    HTMLBody: `<!DOCTYPE html>
<html>
<body>
<p>Hi {{.Name}},</p>
<p>Your order <strong>#{{.OrderID}}</strong> has been confirmed.</p>
<table>
{{range .Items}}<tr><td>{{.Name}}</td><td>${{.Price}}</td></tr>{{end}}
</table>
<p><strong>Total: ${{.Total}}</strong></p>
</body>
</html>`,
})
```

---

## Common SMTP Configurations

### Amazon SES

```go
email.Config{
    Host:        "email-smtp.us-east-1.amazonaws.com",
    Port:        587,
    Username:    "AKIA...",  // SES SMTP username
    Password:    "...",      // SES SMTP password
    FromAddress: "noreply@yourdomain.com",
}
```

### SendGrid

```go
email.Config{
    Host:        "smtp.sendgrid.net",
    Port:        587,
    Username:    "apikey",
    Password:    "SG.xxx",  // Your API key
    FromAddress: "noreply@yourdomain.com",
}
```

### Mailgun

```go
email.Config{
    Host:        "smtp.mailgun.org",
    Port:        587,
    Username:    "postmaster@yourdomain.com",
    Password:    "your-mailgun-password",
    FromAddress: "noreply@yourdomain.com",
}
```

### Gmail (for development)

```go
email.Config{
    Host:        "smtp.gmail.com",
    Port:        587,
    Username:    "your@gmail.com",
    Password:    "app-specific-password",
    FromAddress: "your@gmail.com",
}
```

---

## Complete Example

```go
package main

import (
    "context"
    "time"

    "github.com/dalemusser/waffle/pantry/email"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Create sender
    sender := email.NewSender(email.Config{
        Host:        "smtp.sendgrid.net",
        Port:        587,
        Username:    "apikey",
        Password:    os.Getenv("SENDGRID_API_KEY"),
        FromAddress: "noreply@myapp.com",
        FromName:    "My App",
    })

    // Create queue with Redis storage
    queue := email.NewQueue(email.QueueConfig{
        Sender:  sender,
        Store:   email.NewRedisQueueStore(email.RedisQueueConfig{Client: redisClient}),
        Logger:  logger,
        Workers: 4,
        OnSent: func(e *email.QueuedEmail) {
            logger.Info("email sent", zap.String("id", e.ID))
        },
        OnFailed: func(e *email.QueuedEmail, err error) {
            logger.Error("email failed", zap.String("id", e.ID), zap.Error(err))
        },
    })

    // Create template store
    templates := email.NewTemplateStore()
    templates.RegisterCommonTemplates()

    // Create template queue
    tplQueue := email.NewTemplateQueue(queue, templates)

    // Start queue
    queue.Start()
    defer queue.Stop(context.Background())

    // In a handler...
    ctx := context.Background()

    // Send welcome email
    id, err := tplQueue.EnqueueTo(ctx, "newuser@example.com", "welcome", map[string]any{
        "Name":      "John",
        "AppName":   "My App",
        "VerifyURL": "https://myapp.com/verify?token=abc123",
    })
    if err != nil {
        logger.Error("failed to queue email", zap.Error(err))
    }

    // Check status
    email, _ := queue.Get(ctx, id)
    logger.Info("email status", zap.String("status", string(email.Status)))
}
```

---

## Usage with WAFFLE

### Configuration via AppConfig

```go
// internal/app/bootstrap/appconfig.go
type AppConfig struct {
    SMTPHost     string `conf:"smtp_host"`
    SMTPPort     int    `conf:"smtp_port" conf-default:"587"`
    SMTPUsername string `conf:"smtp_username"`
    SMTPPassword string `conf:"smtp_password"`
    SMTPFrom     string `conf:"smtp_from"`
    SMTPFromName string `conf:"smtp_from_name"`
}
```

```toml
# config.toml
smtp_host = "smtp.sendgrid.net"
smtp_port = 587
smtp_username = "apikey"
smtp_password = "SG.xxx"
smtp_from = "noreply@myapp.com"
smtp_from_name = "My App"
```

### In DBDeps

```go
// internal/app/bootstrap/dbdeps.go
type DBDeps struct {
    EmailQueue *email.Queue
    EmailTpl   *email.TemplateQueue
}

// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, cfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    sender := email.NewSender(email.Config{
        Host:        appCfg.SMTPHost,
        Port:        appCfg.SMTPPort,
        Username:    appCfg.SMTPUsername,
        Password:    appCfg.SMTPPassword,
        FromAddress: appCfg.SMTPFrom,
        FromName:    appCfg.SMTPFromName,
    })

    queue := email.NewQueue(email.QueueConfig{
        Sender:  sender,
        Store:   email.NewMemoryQueueStore(),
        Logger:  logger,
        Workers: 2,
    })
    queue.Start()

    templates := email.NewTemplateStore()
    templates.RegisterCommonTemplates()

    return DBDeps{
        EmailQueue: queue,
        EmailTpl:   email.NewTemplateQueue(queue, templates),
    }, nil
}
```

### In a Handler

```go
// internal/app/features/auth/handler.go
type Handler struct {
    email  *email.TemplateQueue
    logger *zap.Logger
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    // ... generate reset token ...

    _, err := h.email.EnqueueTo(r.Context(), user.Email, "password_reset", map[string]any{
        "Name":      user.Name,
        "AppName":   "My App",
        "ResetURL":  resetURL,
        "ExpiresIn": "1 hour",
    })
    if err != nil {
        h.logger.Error("failed to queue password reset email", zap.Error(err))
        // Handle error...
    }
}
```

---

## Sending with Attachments

```go
msg := email.Message{
    To:      []string{"user@example.com"},
    Subject: "Your Report",
    TextBody: "Please find your report attached.",
    Attachments: []email.Attachment{
        {
            Filename:    "report.pdf",
            ContentType: "application/pdf",
            Data:        pdfBytes,
        },
    },
}

// Sync
err := sender.Send(ctx, msg)

// Async
id, err := queue.EnqueueMessage(ctx, msg)
```

---

## Error Handling

The package returns wrapped errors with context:

```go
err := sender.Send(ctx, msg)
if err != nil {
    // Error messages include prefix and cause:
    // "email: no recipients specified"
    // "email: message body is empty"
    // "email: invalid from address: ..."
    // "email: failed to create client: ..."
    // "email: failed to send: ..."
    log.Printf("email error: %v", err)
}
```

---

## TLS vs SSL

| Port | Protocol | Config |
|------|----------|--------|
| 587 | STARTTLS | `UseTLS: true` (default) |
| 465 | Implicit SSL | `UseSSL: true` |
| 25 | Plain (not recommended) | Neither |

The package defaults to TLS on port 587. For port 465, set `UseSSL: true`.

---

## See Also

- [jobs](../jobs/jobs.md) — Background job processing
- [mq](../mq/) — Message queue integrations
