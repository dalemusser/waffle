# Webhook Package

Utilities for handling incoming webhooks with signature verification and sending outgoing webhooks with automatic retries.

## Installation

```go
import "github.com/dalemusser/waffle/pantry/webhook"
```

## Features

- **Signature verification** for GitHub, Stripe, Slack, Shopify, and generic HMAC
- **Outgoing webhooks** with automatic retries and exponential backoff
- **Event routing** with pattern matching (wildcards)
- **Subscription management** for fan-out delivery
- **Type-safe handlers** with generics

## Receiving Webhooks

### Basic Signature Verification

```go
// Generic HMAC-SHA256 verification
verifier := webhook.NewHMACVerifier(webhook.HMACConfig{
    Secret:    "your-webhook-secret",
    Algorithm: webhook.SHA256,
    Header:    "X-Webhook-Signature",
    Prefix:    "sha256=", // Optional prefix to strip
})

http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    if err := verifier.Verify(r); err != nil {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    // Process webhook...
})
```

### GitHub Webhooks

```go
verifier := webhook.NewGitHubVerifier("your-github-webhook-secret")

http.HandleFunc("/github/webhook", func(w http.ResponseWriter, r *http.Request) {
    if err := verifier.Verify(r); err != nil {
        webhook.WriteError(w, http.StatusUnauthorized, err)
        return
    }

    // Get the event type
    eventType := r.Header.Get("X-GitHub-Event")

    // Read the payload
    body, _ := webhook.PayloadFromContext(r.Context())
    // or: body, _ := io.ReadAll(r.Body)

    switch eventType {
    case "push":
        // Handle push event
    case "pull_request":
        // Handle PR event
    }

    webhook.WriteSuccess(w, "OK")
})
```

### Stripe Webhooks

```go
verifier := webhook.NewStripeVerifier("whsec_...").
    WithTolerance(5 * time.Minute) // Timestamp tolerance

http.HandleFunc("/stripe/webhook", func(w http.ResponseWriter, r *http.Request) {
    if err := verifier.Verify(r); err != nil {
        webhook.WriteError(w, http.StatusUnauthorized, err)
        return
    }
    // Process Stripe event...
})
```

### Slack Webhooks

```go
verifier := webhook.NewSlackVerifier("your-signing-secret")

http.HandleFunc("/slack/webhook", func(w http.ResponseWriter, r *http.Request) {
    if err := verifier.Verify(r); err != nil {
        webhook.WriteError(w, http.StatusUnauthorized, err)
        return
    }
    // Process Slack event...
})
```

### Verification Middleware

```go
verifier := webhook.NewGitHubVerifier("secret")

// As middleware
mux.Handle("/webhook", webhook.VerifyMiddleware(verifier)(yourHandler))

// As wrapper function
mux.HandleFunc("/webhook", webhook.VerifyFunc(verifier, yourHandlerFunc))
```

## Event Routing

Route incoming webhooks to handlers based on event type.

### Basic Router

```go
router := webhook.NewRouter(webhook.RouterConfig{
    Verifier: webhook.NewGitHubVerifier("secret"),
})

// Exact match
router.On("push", func(ctx context.Context, event *webhook.RawEvent) error {
    log.Printf("Received push event: %s", event.ID)
    return nil
})

// Wildcard match
router.On("pull_request.*", func(ctx context.Context, event *webhook.RawEvent) error {
    log.Printf("PR event: %s", event.Type)
    return nil
})

// Match all events
router.On("*", func(ctx context.Context, event *webhook.RawEvent) error {
    log.Printf("Any event: %s", event.Type)
    return nil
})

// Use as HTTP handler
http.Handle("/webhook", router)
```

### HTTP Handlers

```go
router.OnHTTP("order.created", func(w http.ResponseWriter, r *http.Request, event *webhook.RawEvent) {
    // Full control over the response
    var order Order
    if err := event.ParseData(&order); err != nil {
        http.Error(w, "Invalid order data", http.StatusBadRequest)
        return
    }

    // Process order...

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
})
```

### Type-Safe Handlers

```go
type OrderEvent struct {
    OrderID   string  `json:"order_id"`
    Amount    float64 `json:"amount"`
    Customer  string  `json:"customer"`
}

router.On("order.created", webhook.TypedHandler(func(
    ctx context.Context,
    event *webhook.RawEvent,
    data OrderEvent,
) error {
    log.Printf("Order %s created for $%.2f", data.OrderID, data.Amount)
    return processOrder(ctx, data)
}))
```

### Handler Groups

```go
// Group handlers by prefix
orders := router.Group("order")
orders.On("created", handleOrderCreated)   // matches "order.created"
orders.On("updated", handleOrderUpdated)   // matches "order.updated"
orders.On("*", handleAllOrderEvents)       // matches "order.*"
```

### Lifecycle Hooks

```go
router := webhook.NewRouter(webhook.RouterConfig{
    Verifier: verifier,

    BeforeHandler: func(ctx context.Context, event *webhook.RawEvent) error {
        log.Printf("Processing event: %s (type: %s)", event.ID, event.Type)
        return nil
    },

    AfterHandler: func(ctx context.Context, event *webhook.RawEvent, err error) {
        if err != nil {
            log.Printf("Event %s failed: %v", event.ID, err)
        } else {
            log.Printf("Event %s processed successfully", event.ID)
        }
    },

    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
        log.Printf("Webhook error: %v", err)
        webhook.WriteError(w, http.StatusInternalServerError, err)
    },
})
```

## Sending Webhooks

### Basic Sending

```go
sender := webhook.NewSender(webhook.SenderConfig{
    SigningSecret: "your-secret",
    Timeout:       30 * time.Second,
    MaxRetries:    3,
})

// Create an event
event := webhook.NewEvent("order.created", map[string]any{
    "order_id": "12345",
    "amount":   99.99,
})

// Send with automatic retries
err := sender.Send(ctx, "https://example.com/webhook", event)
```

### With Delivery Results

```go
results, err := sender.SendWithResults(ctx, "https://example.com/webhook", event)
for _, result := range results {
    log.Printf("Attempt %d: status=%d, duration=%v, success=%v",
        result.Attempt, result.StatusCode, result.Duration, result.Success)
}
```

### Sender Configuration

```go
sender := webhook.NewSender(webhook.SenderConfig{
    // Signing
    SigningSecret:   "secret",
    SignatureHeader: "X-Webhook-Signature",
    TimestampHeader: "X-Webhook-Timestamp",

    // Timeouts and retries
    Timeout:           30 * time.Second,
    MaxRetries:        3,
    RetryBackoff:      1 * time.Second,
    MaxBackoff:        1 * time.Minute,
    BackoffMultiplier: 2.0,

    // Custom HTTP client
    HTTPClient: &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns: 100,
        },
    },

    // Additional headers
    Headers: map[string]string{
        "X-Custom-Header": "value",
    },

    // Delivery callback
    OnDelivery: func(url string, result webhook.DeliveryResult) {
        log.Printf("Delivery to %s: attempt=%d, status=%d, success=%v",
            url, result.Attempt, result.StatusCode, result.Success)
    },
})
```

### Batch Sending

```go
batchSender := webhook.NewBatchSender(sender, 10) // 10 concurrent

urls := []string{
    "https://service1.example.com/webhook",
    "https://service2.example.com/webhook",
    "https://service3.example.com/webhook",
}

results := batchSender.SendToAll(ctx, urls, event)
for _, result := range results {
    if result.Error != nil {
        log.Printf("Failed to deliver to %s: %v", result.URL, result.Error)
    }
}
```

## Subscription Management

Manage webhook subscriptions and dispatch events to matching endpoints.

```go
sender := webhook.NewSender(webhook.SenderConfig{
    SigningSecret: "default-secret",
})

dispatcher := webhook.NewDispatcher(sender)

// Add subscriptions
dispatcher.Subscribe(&webhook.Subscription{
    ID:     "sub_1",
    URL:    "https://service1.example.com/webhook",
    Events: []string{"order.created", "order.updated"},
    Secret: "service1-secret", // Optional per-subscription secret
    Active: true,
})

dispatcher.Subscribe(&webhook.Subscription{
    ID:     "sub_2",
    URL:    "https://service2.example.com/webhook",
    Events: []string{"*"}, // All events
    Active: true,
})

// Dispatch an event to all matching subscriptions
event := webhook.NewEvent("order.created", orderData)
results := dispatcher.Dispatch(ctx, event)

// Check results
for _, result := range results {
    if result.Error != nil {
        log.Printf("Failed to deliver to %s: %v", result.URL, result.Error)
    }
}

// Async dispatch
resultsChan := make(chan webhook.BatchResult, 10)
dispatcher.DispatchAsync(ctx, event, resultsChan)
for result := range resultsChan {
    log.Printf("Delivered to %s", result.URL)
}
```

### Subscription Pattern Matching

```go
sub := &webhook.Subscription{
    Events: []string{
        "order.created",  // Exact match
        "payment.*",      // Wildcard: matches payment.success, payment.failed, etc.
        "*",              // Matches all events
    },
    Active: true,
}

sub.Matches("order.created")   // true
sub.Matches("order.updated")   // false
sub.Matches("payment.success") // true
sub.Matches("user.created")    // true (matches *)
```

## Event Structure

### Creating Events

```go
// Using helper
event := webhook.NewEvent("order.created", map[string]any{
    "order_id": "12345",
    "amount":   99.99,
})

// Manual creation
event := webhook.Event{
    ID:        "evt_123",
    Type:      "order.created",
    Timestamp: time.Now().UTC(),
    Data:      orderData,
    Metadata: map[string]string{
        "source": "api",
    },
}
```

### Parsing Events

```go
// From request body
body, _ := io.ReadAll(r.Body)
event, err := webhook.ParseEvent(body)
if err != nil {
    // Handle error
}

// Parse typed data
var order Order
if err := event.ParseData(&order); err != nil {
    // Handle error
}
```

## Signing Payloads

```go
// Simple HMAC-SHA256
signature := webhook.SignPayload(payload, "secret")

// With timestamp (Stripe-style)
timestamp := time.Now().Unix()
signature := webhook.SignPayloadWithTimestamp(timestamp, payload, "secret")

// Compute HMAC with different algorithms
sig := webhook.ComputeHMAC(payload, []byte("secret"), webhook.SHA256)
sig := webhook.ComputeHMAC(payload, []byte("secret"), webhook.SHA512)
```

## Error Handling

```go
import "errors"

err := verifier.Verify(r)

switch {
case errors.Is(err, webhook.ErrInvalidSignature):
    // Signature didn't match
case errors.Is(err, webhook.ErrMissingSignature):
    // No signature header present
case errors.Is(err, webhook.ErrTimestampExpired):
    // Timestamp outside tolerance window
case errors.Is(err, webhook.ErrMissingTimestamp):
    // No timestamp in signed payload
case errors.Is(err, webhook.ErrInvalidPayload):
    // Could not parse webhook payload
case errors.Is(err, webhook.ErrDeliveryFailed):
    // All delivery attempts failed
case errors.Is(err, webhook.ErrNoHandlerFound):
    // No handler registered for event type
}
```

## Context Values

```go
// Get event from context (set by router)
event, ok := webhook.EventFromContext(ctx)

// Get raw payload from context
payload, ok := webhook.PayloadFromContext(ctx)
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/dalemusser/waffle/pantry/webhook"
)

type OrderData struct {
    OrderID  string  `json:"order_id"`
    Amount   float64 `json:"amount"`
    Customer string  `json:"customer"`
}

func main() {
    // Create router with GitHub verification
    router := webhook.NewRouter(webhook.RouterConfig{
        Verifier: webhook.NewGitHubVerifier("github-secret"),
        BeforeHandler: func(ctx context.Context, event *webhook.RawEvent) error {
            log.Printf("Processing: %s", event.Type)
            return nil
        },
    })

    // Register handlers
    router.On("push", handlePush)
    router.On("pull_request.*", handlePullRequest)

    // Typed handler
    router.On("order.created", webhook.TypedHandler(handleOrderCreated))

    // Start server
    log.Println("Listening on :8080")
    http.ListenAndServe(":8080", router)
}

func handlePush(ctx context.Context, event *webhook.RawEvent) error {
    log.Printf("Push event received")
    return nil
}

func handlePullRequest(ctx context.Context, event *webhook.RawEvent) error {
    log.Printf("PR event: %s", event.Type)
    return nil
}

func handleOrderCreated(ctx context.Context, event *webhook.RawEvent, data OrderData) error {
    log.Printf("Order %s created: $%.2f from %s",
        data.OrderID, data.Amount, data.Customer)

    // Send webhook to downstream service
    sender := webhook.NewSender(webhook.SenderConfig{
        SigningSecret: "downstream-secret",
    })

    return sender.Send(ctx, "https://downstream.example.com/webhook",
        webhook.NewEvent("order.processed", data))
}
```
