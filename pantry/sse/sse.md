# sse

Server-Sent Events (SSE) for real-time server-to-client streaming.

## Overview

The `sse` package provides:
- **Event streaming** — Send events from server to client over HTTP
- **Broker & Channels** — Manage multiple clients and targeted broadcasting
- **Automatic reconnection** — Last-Event-ID support for seamless reconnects
- **Keep-alive** — Automatic heartbeats to maintain connections

SSE is ideal for:
- Live notifications
- Real-time dashboards
- Activity feeds
- Stock tickers
- Log streaming

## Import

```go
import "github.com/dalemusser/waffle/sse"
```

---

## Quick Start

### Simple Event Stream

```go
func main() {
    r := chi.NewRouter()

    r.Get("/events", sse.HandlerFunc(func(stream *sse.Stream, r *http.Request) {
        ctx := r.Context()

        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return
            case <-time.After(time.Second):
                stream.SendData(fmt.Sprintf("Event %d", i))
            }
        }
    }))

    http.ListenAndServe(":8080", r)
}
```

### Notifications with Broker

```go
var broker = sse.NewBroker()

func main() {
    r := chi.NewRouter()

    // SSE endpoint
    r.Get("/notifications", broker.HandleFunc(func(client *sse.Client) {
        userID := client.Stream().LastEventID // Or from auth
        client.Subscribe("user:" + userID)
    }))

    // Send notification (from another endpoint or goroutine)
    r.Post("/notify", func(w http.ResponseWriter, r *http.Request) {
        var n Notification
        json.NewDecoder(r.Body).Decode(&n)

        broker.Channel("user:" + n.UserID).BroadcastJSON("notification", n)
        w.WriteHeader(http.StatusOK)
    })

    http.ListenAndServe(":8080", r)
}
```

---

## SSE Protocol

Server-Sent Events use a simple text-based format:

```
event: message
id: 123
data: Hello, world!

event: update
data: {"count": 42}

: this is a comment (used for keep-alive)

```

Key features:
- **event** — Event type (optional, defaults to "message")
- **id** — Event ID for reconnection tracking
- **data** — Event payload (can be multi-line)
- **retry** — Reconnection interval in milliseconds
- Comments start with `:` and are ignored by clients

---

## Events

**Location:** `sse.go`

### Creating Events

```go
// Simple data event
event := sse.NewEvent("Hello, world!")

// Typed event
event := sse.NewEventWithType("notification", `{"title":"New message"}`)

// JSON event
event, err := sse.NewJSONEvent("update", map[string]int{"count": 42})

// Panic on JSON error (for static data)
event := sse.MustJSONEvent("config", config)

// Fluent API
event := sse.NewEvent("data").
    WithID("123").
    WithRetry(5000)  // 5 second reconnect
```

### Event Structure

```go
type Event struct {
    ID    string  // Event ID for reconnection
    Event string  // Event type (e.g., "message", "update")
    Data  string  // Event payload
    Retry int     // Reconnection interval (ms)
}
```

---

## Single Stream

For simple use cases with one client per handler.

### Creating a Stream

```go
func handler(w http.ResponseWriter, r *http.Request) {
    stream, err := sse.NewStream(w, r)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Check for reconnection
    if stream.LastEventID != "" {
        // Client reconnected, resume from last ID
    }

    // Stream events...
}

// Or use the handler wrapper
r.Get("/events", sse.HandlerFunc(func(stream *sse.Stream, r *http.Request) {
    // Stream events...
}))
```

### Sending Events

```go
// Send event struct
stream.Send(&sse.Event{
    ID:    "1",
    Event: "update",
    Data:  `{"value": 42}`,
})

// Send simple data
stream.SendData("Hello!")

// Send typed event
stream.SendEvent("notification", "You have a new message")

// Send JSON
stream.SendJSON("user", User{Name: "Alice"})

// Send keep-alive comment
stream.SendComment("ping")

// Send retry interval
stream.SendRetry(3000)  // 3 seconds
```

### Stream Lifecycle

```go
// Check if closed
if stream.IsClosed() {
    return
}

// Close stream
stream.Close()

// Wait for close
<-stream.Done()
```

### Using Serve Helper

The `Serve` function handles keep-alive and context cancellation:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    stream, _ := sse.NewStream(w, r)

    events := make(chan *sse.Event)
    go func() {
        defer close(events)
        for i := 0; ; i++ {
            time.Sleep(time.Second)
            events <- sse.NewEvent(fmt.Sprintf("tick %d", i)).WithID(fmt.Sprint(i))
        }
    }()

    cfg := sse.DefaultConfig()
    sse.Serve(r.Context(), stream, cfg, events)
}
```

### Using ServeFunc Helper

For more control over event generation:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    stream, _ := sse.NewStream(w, r)

    cfg := sse.DefaultConfig()
    sse.ServeFunc(r.Context(), stream, cfg, func(ctx context.Context, send func(*sse.Event) error) error {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-ticker.C:
                if err := send(sse.NewEvent(fmt.Sprintf("tick %d", i))); err != nil {
                    return err
                }
            }
        }
    })
}
```

---

## Broker

**Location:** `broker.go`

For managing multiple clients with broadcasting.

### Creating a Broker

```go
// Default configuration
broker := sse.NewBroker()

// Custom configuration
broker := sse.NewBrokerWithConfig(sse.BrokerConfig{
    KeepAliveInterval: 30 * time.Second,
    RetryInterval:     3 * time.Second,
    ClientBufferSize:  256,
})

// Callbacks
broker.OnConnect = func(client *sse.Client) {
    log.Printf("Client connected: %s", client.ID())
}

broker.OnDisconnect = func(client *sse.Client) {
    log.Printf("Client disconnected: %s", client.ID())
}
```

### Handling Connections

```go
// As http.Handler
r.Get("/events", broker.Handler())

// As http.HandlerFunc with setup
r.Get("/events", broker.HandleFunc(func(client *sse.Client) {
    // Set up client (subscribe to channels, set data, etc.)
    userID := getUserID(client.Stream().Request)
    client.Set("user_id", userID)
    client.Subscribe("notifications:" + userID)
}))

// Manual handling
func handler(w http.ResponseWriter, r *http.Request) {
    broker.HandleRequest(w, r, func(client *sse.Client) {
        // Setup
    })
}
```

### Broadcasting

```go
// To all clients
broker.Broadcast(event)
broker.BroadcastData("Hello everyone!")
broker.BroadcastEvent("announcement", "Server maintenance at midnight")
broker.BroadcastJSON("stats", Stats{Users: 100})

// Iterate clients
broker.ForEach(func(client *sse.Client) {
    if client.GetString("role") == "admin" {
        client.SendEvent("admin", "Special message")
    }
})

// Get client count
count := broker.Clients()
```

### Channels

Channels group clients for targeted broadcasting.

```go
// Get or create channel
channel := broker.GetChannel("news")

// Get existing channel (nil if not found)
channel := broker.Channel("news")

// Subscribe/unsubscribe
channel.Subscribe(client)
channel.Unsubscribe(client)

// Check subscription
if channel.Has(client) {
    // ...
}

// Broadcast to channel
channel.Broadcast(event)
channel.BroadcastData("Breaking news!")
channel.BroadcastEvent("article", articleJSON)
channel.BroadcastJSON("update", update)

// Channel info
size := channel.Size()
name := channel.Name()

// Iterate subscribers
channel.ForEach(func(client *sse.Client) {
    // ...
})

// Delete channel
broker.DeleteChannel("news")

// List channels
names := broker.Channels()
```

### Client Methods

```go
// Identity
id := client.ID()
lastEventID := client.LastEventID()

// Custom data
client.Set("user_id", "123")
userID := client.GetString("user_id")
value, ok := client.Get("key")

// Send events
client.Send(event)
client.SendData("Hello!")
client.SendEvent("notification", "New message")
client.SendJSON("update", data)

// Channel management
client.Subscribe("news")
client.Unsubscribe("news")
channels := client.Channels()
inChannel := client.InChannel("news")

// Lifecycle
<-client.Done()  // Wait for disconnect
client.Close()   // Force close
```

---

## Configuration

```go
// Stream configuration
type Config struct {
    KeepAliveInterval time.Duration  // Keep-alive interval (default 30s)
    RetryInterval     time.Duration  // Suggested reconnect time (default 3s)
    OnConnect         func(*Stream)
    OnDisconnect      func(*Stream)
}

// Broker configuration
type BrokerConfig struct {
    KeepAliveInterval time.Duration  // Keep-alive interval (default 30s)
    RetryInterval     time.Duration  // Suggested reconnect time (default 3s)
    ClientBufferSize  int            // Event buffer per client (default 256)
}
```

---

## Error Handling

```go
// Common errors
sse.ErrFlushNotSupported  // Response writer can't flush
sse.ErrStreamClosed       // Stream already closed
sse.ErrBrokerClosed       // Broker already closed
sse.ErrChannelNotFound    // Channel doesn't exist
sse.ErrClientNotFound     // Client not registered
```

---

## Complete Examples

### Live Dashboard

```go
package main

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/dalemusser/waffle/sse"
    "github.com/go-chi/chi/v5"
)

var broker = sse.NewBroker()

type Stats struct {
    Users     int     `json:"users"`
    Requests  int     `json:"requests"`
    CPU       float64 `json:"cpu"`
    Memory    float64 `json:"memory"`
    Timestamp int64   `json:"timestamp"`
}

func main() {
    // Start stats broadcaster
    go broadcastStats()

    r := chi.NewRouter()
    r.Get("/dashboard/events", broker.Handler())
    http.ListenAndServe(":8080", r)
}

func broadcastStats() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        stats := collectStats()
        broker.BroadcastJSON("stats", stats)
    }
}

func collectStats() Stats {
    return Stats{
        Users:     getActiveUsers(),
        Requests:  getRequestCount(),
        CPU:       getCPUUsage(),
        Memory:    getMemoryUsage(),
        Timestamp: time.Now().Unix(),
    }
}
```

### User Notifications

```go
package main

import (
    "encoding/json"
    "net/http"
    "sync/atomic"

    "github.com/dalemusser/waffle/sse"
    "github.com/go-chi/chi/v5"
)

var (
    broker  = sse.NewBroker()
    eventID atomic.Uint64
)

type Notification struct {
    ID      string `json:"id"`
    Title   string `json:"title"`
    Message string `json:"message"`
    Type    string `json:"type"` // info, warning, error
}

func main() {
    broker.OnConnect = func(c *sse.Client) {
        // Send pending notifications on reconnect
        if c.LastEventID() != "" {
            sendMissedNotifications(c)
        }
    }

    r := chi.NewRouter()

    // SSE endpoint
    r.Get("/notifications/stream", broker.HandleFunc(func(client *sse.Client) {
        userID := getUserFromAuth(client.Stream())
        client.Set("user_id", userID)
        client.Subscribe("user:" + userID)
        client.Subscribe("broadcast")  // Global notifications
    }))

    // Send notification
    r.Post("/notifications/send", handleSendNotification)

    http.ListenAndServe(":8080", r)
}

func handleSendNotification(w http.ResponseWriter, r *http.Request) {
    var req struct {
        UserID  string `json:"user_id"`  // Empty for broadcast
        Title   string `json:"title"`
        Message string `json:"message"`
        Type    string `json:"type"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    notification := Notification{
        ID:      generateEventID(),
        Title:   req.Title,
        Message: req.Message,
        Type:    req.Type,
    }

    event, _ := sse.NewJSONEvent("notification", notification)
    event.WithID(notification.ID)

    if req.UserID != "" {
        // Send to specific user
        if ch := broker.Channel("user:" + req.UserID); ch != nil {
            ch.Broadcast(event)
        }
    } else {
        // Broadcast to all
        broker.Channel("broadcast").Broadcast(event)
    }

    w.WriteHeader(http.StatusOK)
}

func generateEventID() string {
    return fmt.Sprintf("%d", eventID.Add(1))
}
```

### Log Streaming

```go
package main

import (
    "bufio"
    "os"
    "time"

    "github.com/dalemusser/waffle/sse"
    "github.com/go-chi/chi/v5"
)

func main() {
    r := chi.NewRouter()

    r.Get("/logs/stream", sse.HandlerFunc(func(stream *sse.Stream, r *http.Request) {
        logFile := r.URL.Query().Get("file")
        if logFile == "" {
            logFile = "/var/log/app.log"
        }

        cfg := sse.Config{
            KeepAliveInterval: 30 * time.Second,
        }

        sse.ServeFunc(r.Context(), stream, cfg, func(ctx context.Context, send func(*sse.Event) error) error {
            return tailFile(ctx, logFile, send)
        })
    }))

    http.ListenAndServe(":8080", r)
}

func tailFile(ctx context.Context, path string, send func(*sse.Event) error) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    // Seek to end
    file.Seek(0, 2)

    scanner := bufio.NewScanner(file)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            for scanner.Scan() {
                line := scanner.Text()
                if err := send(sse.NewEventWithType("log", line)); err != nil {
                    return err
                }
            }
        }
    }
}
```

### Progress Updates

```go
package main

import (
    "net/http"
    "time"

    "github.com/dalemusser/waffle/sse"
    "github.com/go-chi/chi/v5"
)

var broker = sse.NewBroker()

type Progress struct {
    JobID    string  `json:"job_id"`
    Progress float64 `json:"progress"`
    Status   string  `json:"status"`
    Message  string  `json:"message,omitempty"`
}

func main() {
    r := chi.NewRouter()

    // Subscribe to job progress
    r.Get("/jobs/{jobID}/progress", func(w http.ResponseWriter, r *http.Request) {
        jobID := chi.URLParam(r, "jobID")

        broker.HandleRequest(w, r, func(client *sse.Client) {
            client.Subscribe("job:" + jobID)
        })
    })

    // Start a job
    r.Post("/jobs", handleStartJob)

    http.ListenAndServe(":8080", r)
}

func handleStartJob(w http.ResponseWriter, r *http.Request) {
    jobID := generateJobID()

    // Start job in background
    go runJob(jobID)

    json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

func runJob(jobID string) {
    channel := broker.GetChannel("job:" + jobID)

    for i := 0; i <= 100; i += 10 {
        progress := Progress{
            JobID:    jobID,
            Progress: float64(i),
            Status:   "running",
            Message:  fmt.Sprintf("Processing step %d", i/10),
        }

        channel.BroadcastJSON("progress", progress)
        time.Sleep(500 * time.Millisecond)
    }

    // Final update
    channel.BroadcastJSON("progress", Progress{
        JobID:    jobID,
        Progress: 100,
        Status:   "completed",
        Message:  "Job finished successfully",
    })

    // Clean up channel after a delay
    time.AfterFunc(time.Minute, func() {
        broker.DeleteChannel("job:" + jobID)
    })
}
```

---

## Client-Side JavaScript

```javascript
// Basic connection
const events = new EventSource('/events');

events.onmessage = (e) => {
    console.log('Message:', e.data);
};

events.onerror = (e) => {
    console.log('Error, reconnecting...');
};

// Listen for specific event types
events.addEventListener('notification', (e) => {
    const data = JSON.parse(e.data);
    showNotification(data);
});

events.addEventListener('update', (e) => {
    const data = JSON.parse(e.data);
    updateUI(data);
});

// Close connection
events.close();
```

### With Authentication

```javascript
// SSE doesn't support custom headers, use query params or cookies
const events = new EventSource('/events?token=' + authToken);

// Or ensure cookies are sent
const events = new EventSource('/events', { withCredentials: true });
```

### Reconnection Handling

```javascript
// EventSource automatically reconnects
// The browser sends Last-Event-ID header

events.onopen = () => {
    console.log('Connected');
};

events.onerror = (e) => {
    if (events.readyState === EventSource.CONNECTING) {
        console.log('Reconnecting...');
    }
};
```

---

## SSE vs WebSocket

| Feature | SSE | WebSocket |
|---------|-----|-----------|
| Direction | Server → Client only | Bidirectional |
| Protocol | HTTP | WebSocket |
| Reconnection | Automatic | Manual |
| Binary data | No (text only) | Yes |
| Browser support | All modern | All modern |
| Firewall/proxy | Usually works | May be blocked |
| Complexity | Simple | More complex |

**Use SSE when:**
- You only need server-to-client communication
- You want automatic reconnection
- You need to work through HTTP proxies
- Simplicity is preferred

**Use WebSocket when:**
- You need bidirectional communication
- You need binary data
- Low latency is critical
- You need to send data from client to server

---

## See Also

- [websocket](../websocket/websocket.md) — Bidirectional WebSocket communication
- [cache](../cache/cache.md) — Cache event data
- [requestid](../requestid/requestid.md) — Request ID for tracing
