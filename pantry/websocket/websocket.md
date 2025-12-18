# websocket

WebSocket support for real-time bidirectional communication.

## Overview

The `websocket` package provides:
- **Connection handling** — Upgrade HTTP to WebSocket, read/write messages
- **Hub & Rooms** — Manage multiple connections, broadcast to groups
- **Message routing** — Type-based message handling with JSON support
- **Ping/pong** — Automatic heartbeats to detect dead connections

Built on [github.com/coder/websocket](https://github.com/coder/websocket), the modern WebSocket library recommended by the Go team.

## Import

```go
import "github.com/dalemusser/waffle/websocket"
```

---

## Quick Start

### Simple Echo Server

```go
func main() {
    r := chi.NewRouter()

    r.Get("/ws", websocket.HandlerFunc(nil, func(conn *websocket.Conn) {
        defer conn.Close()

        ctx := context.Background()
        for {
            msgType, data, err := conn.Read(ctx)
            if err != nil {
                return
            }
            conn.Write(ctx, msgType, data)
        }
    }))

    http.ListenAndServe(":8080", r)
}
```

### Chat Room with Hub

```go
var hub = websocket.NewHub()

func main() {
    r := chi.NewRouter()
    r.Get("/ws", handleWebSocket)
    http.ListenAndServe(":8080", r)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, nil)
    if err != nil {
        return
    }

    // Create client and join chat room
    client := hub.NewClient(conn, r.RemoteAddr)
    client.Join("chat")
    defer client.Close()

    ctx := context.Background()
    for {
        _, data, err := conn.Read(ctx)
        if err != nil {
            return
        }

        // Broadcast to everyone in the chat room
        hub.Room("chat").BroadcastText(ctx, string(data))
    }
}
```

---

## Connection Handling

**Location:** `websocket.go`

### Accepting Connections

```go
// Basic accept
conn, err := websocket.Accept(w, r, nil)

// With options
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    Subprotocols:       []string{"chat", "json"},
    OriginPatterns:     []string{"https://example.com"},
    CompressionMode:    websocket.CompressionNoContextTakeover,
})

// As handler
r.Get("/ws", websocket.HandlerFunc(nil, func(conn *websocket.Conn) {
    // Handle connection
}))
```

### Accept Options

```go
type AcceptOptions struct {
    // Subprotocols lists supported subprotocols (first match wins)
    Subprotocols []string

    // InsecureSkipVerify disables origin checking (dev only!)
    InsecureSkipVerify bool

    // OriginPatterns specifies allowed origins
    // Example: []string{"https://example.com", "https://*.example.com"}
    OriginPatterns []string

    // CompressionMode controls message compression
    CompressionMode CompressionMode

    // CompressionThreshold is minimum size to compress (default 512)
    CompressionThreshold int
}
```

### Reading Messages

```go
// Read any message type
msgType, data, err := conn.Read(ctx)

// Read text only
text, err := conn.ReadText(ctx)

// Read binary only
data, err := conn.ReadBinary(ctx)

// Read JSON
var msg MyMessage
err := conn.ReadJSON(ctx, &msg)

// Read structured message
msg, err := conn.ReadMessage(ctx)
```

### Writing Messages

```go
// Write raw message
err := conn.Write(ctx, websocket.MessageText, []byte("hello"))

// Write text
err := conn.WriteText(ctx, "hello")

// Write binary
err := conn.WriteBinary(ctx, data)

// Write JSON
err := conn.WriteJSON(ctx, myStruct)

// Write structured message
err := conn.WriteMessage(ctx, &websocket.Message{
    Type:    "chat",
    Payload: json.RawMessage(`{"text":"hello"}`),
})
```

### Connection Lifecycle

```go
// Close normally
conn.Close()

// Close with reason
conn.CloseWithReason(websocket.StatusGoingAway, "server shutting down")

// Check if closed
if conn.IsClosed() {
    return
}

// Ping for keepalive
err := conn.Ping(ctx)

// Set max message size (default 32KB)
conn.SetReadLimit(64 * 1024)

// Get negotiated subprotocol
proto := conn.Subprotocol()
```

---

## Hub & Rooms

**Location:** `hub.go`

The Hub manages multiple WebSocket connections and enables broadcasting.

### Creating a Hub

```go
hub := websocket.NewHub()

// Optional callbacks
hub.OnConnect = func(client *websocket.Client) {
    log.Printf("Client connected: %s", client.ID())
}

hub.OnDisconnect = func(client *websocket.Client) {
    log.Printf("Client disconnected: %s", client.ID())
}
```

### Managing Clients

```go
// Create client from connection
client := hub.NewClient(conn, "user-123")

// Client properties
id := client.ID()
conn := client.Conn()

// Store data on client
client.Set("username", "alice")
username := client.GetString("username")

// Close client
client.Close()

// Get client count
count := hub.Clients()

// Iterate all clients
hub.ForEach(func(c *websocket.Client) {
    c.SendText(ctx, "hello everyone")
})
```

### Broadcasting

```go
// Broadcast to all clients
hub.BroadcastText(ctx, "hello everyone")
hub.BroadcastBinary(ctx, data)
hub.BroadcastJSON(ctx, myStruct)

// Broadcast except one client
hub.BroadcastExcept(ctx, sender, websocket.MessageText, data)

// Broadcast event
hub.BroadcastEvent(ctx, "user_joined", map[string]string{"user": "alice"})
```

### Rooms

Rooms group clients for targeted broadcasting.

```go
// Client joins a room
client.Join("chat:general")

// Client leaves a room
client.Leave("chat:general")

// Check room membership
if client.InRoom("chat:general") {
    // ...
}

// Get client's rooms
rooms := client.Rooms()

// Get or create room
room := hub.GetRoom("chat:general")

// Get existing room (returns nil if not found)
room := hub.Room("chat:general")

// Room operations
room.Join(client)
room.Leave(client)
size := room.Size()
hasClient := room.Has(client)

// Broadcast to room
room.BroadcastText(ctx, "hello room")
room.BroadcastJSON(ctx, message)
room.BroadcastExcept(ctx, sender, websocket.MessageText, data)
room.BroadcastEvent(ctx, "message", chatMessage)

// Iterate room clients
room.ForEach(func(c *websocket.Client) {
    // ...
})

// Delete room
hub.DeleteRoom("chat:general")

// List all rooms
roomNames := hub.Rooms()
```

---

## Messages & Routing

**Location:** `message.go`

### Structured Messages

```go
// Message structure
type Message struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload,omitempty"`
    ID      string          `json:"id,omitempty"`
}

// Create message
msg, err := websocket.NewMessage("chat", ChatPayload{Text: "hello"})

// With ID (for request/response correlation)
msg, err := websocket.NewMessageWithID("req-123", "get_user", params)

// Parse payload
var payload ChatPayload
err := msg.ParsePayload(&payload)
```

### Message Router

Route messages by type to handlers:

```go
router := websocket.NewRouter()

// Register handlers
router.Handle("chat", func(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    var payload ChatMessage
    if err := msg.ParsePayload(&payload); err != nil {
        return err
    }

    // Broadcast to room
    return hub.Room("chat").BroadcastJSON(ctx, payload)
})

router.Handle("ping", func(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    return client.SendTypedMessage(ctx, "pong", nil)
})

// Default handler for unknown types
router.Default(func(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    return client.SendTypedMessage(ctx, "error", map[string]string{
        "message": "unknown message type",
    })
})

// Run client with router
err := router.RunClient(ctx, client)
```

### Events

Simple event pattern for pub/sub:

```go
// Send event to client
client.SendEvent(ctx, "user_joined", map[string]any{
    "user_id": "123",
    "name":    "Alice",
})

// Broadcast event to hub
hub.BroadcastEvent(ctx, "notification", Notification{
    Title: "New message",
    Body:  "You have a new message",
})

// Broadcast event to room
room.BroadcastEvent(ctx, "typing", map[string]string{
    "user": "alice",
})
```

---

## Configuration & Heartbeats

### Connection Configuration

```go
cfg := websocket.Config{
    ReadTimeout:    60 * time.Second,   // Max time to wait for message
    WriteTimeout:   10 * time.Second,   // Max time to write message
    PingInterval:   30 * time.Second,   // How often to ping
    PongTimeout:    10 * time.Second,   // How long to wait for pong
    MaxMessageSize: 64 * 1024,          // Max message size (64KB)
}

// Run with config (handles pings automatically)
err := websocket.RunWithConfig(ctx, conn, cfg, func(ctx context.Context, msgType websocket.MessageType, data []byte) error {
    // Handle message
    return conn.Write(ctx, msgType, data) // Echo
})
```

### Manual Ping/Pong

```go
// Send ping and wait for pong
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := conn.Ping(ctx)
```

---

## Error Handling

**Location:** `errors.go`

```go
// Common errors
websocket.ErrConnectionClosed
websocket.ErrExpectedTextMessage
websocket.ErrExpectedBinaryMessage
websocket.ErrHubClosed
websocket.ErrRoomNotFound

// Check close errors
if websocket.IsCloseError(err) {
    // Connection was closed
}

if websocket.IsNormalClose(err) {
    // Normal closure (not an error condition)
}
```

### Status Codes

```go
websocket.StatusNormalClosure   // 1000 - Normal close
websocket.StatusGoingAway       // 1001 - Server shutting down
websocket.StatusProtocolError   // 1002 - Protocol error
websocket.StatusUnsupportedData // 1003 - Unsupported data type
websocket.StatusInvalidPayload  // 1007 - Invalid payload
websocket.StatusPolicyViolation // 1008 - Policy violation
websocket.StatusMessageTooBig   // 1009 - Message too big
websocket.StatusInternalError   // 1011 - Internal error
```

---

## Complete Examples

### Chat Application

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"

    "github.com/dalemusser/waffle/websocket"
    "github.com/go-chi/chi/v5"
)

var hub = websocket.NewHub()

type ChatMessage struct {
    Room    string `json:"room"`
    User    string `json:"user"`
    Text    string `json:"text"`
}

func main() {
    hub.OnConnect = func(c *websocket.Client) {
        log.Printf("Connected: %s", c.ID())
    }
    hub.OnDisconnect = func(c *websocket.Client) {
        log.Printf("Disconnected: %s", c.ID())
    }

    r := chi.NewRouter()
    r.Get("/ws", handleWebSocket)

    log.Println("Server starting on :8080")
    http.ListenAndServe(":8080", r)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        InsecureSkipVerify: true, // For development
    })
    if err != nil {
        log.Printf("Accept error: %v", err)
        return
    }

    username := r.URL.Query().Get("user")
    if username == "" {
        username = "anonymous"
    }

    client := hub.NewClient(conn, username)
    client.Set("username", username)
    defer client.Close()

    // Join default room
    client.Join("general")

    // Announce join
    hub.Room("general").BroadcastEvent(context.Background(), "user_joined", map[string]string{
        "user": username,
    })

    ctx := context.Background()
    for {
        msg, err := conn.ReadMessage(ctx)
        if err != nil {
            if !websocket.IsNormalClose(err) {
                log.Printf("Read error: %v", err)
            }
            return
        }

        switch msg.Type {
        case "chat":
            var chat ChatMessage
            if err := msg.ParsePayload(&chat); err != nil {
                continue
            }
            chat.User = username

            room := hub.Room(chat.Room)
            if room != nil {
                room.BroadcastJSON(ctx, chat)
            }

        case "join_room":
            var payload struct{ Room string }
            msg.ParsePayload(&payload)
            client.Join(payload.Room)

        case "leave_room":
            var payload struct{ Room string }
            msg.ParsePayload(&payload)
            client.Leave(payload.Room)
        }
    }
}
```

### Real-time Notifications

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/dalemusser/waffle/websocket"
    "github.com/go-chi/chi/v5"
)

var hub = websocket.NewHub()

type Notification struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Body      string    `json:"body"`
    Timestamp time.Time `json:"timestamp"`
}

func main() {
    r := chi.NewRouter()

    // WebSocket endpoint for receiving notifications
    r.Get("/ws/notifications", handleNotifications)

    // REST endpoint for sending notifications
    r.Post("/api/notify", handleSendNotification)

    http.ListenAndServe(":8080", r)
}

func handleNotifications(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, nil)
    if err != nil {
        return
    }

    userID := r.URL.Query().Get("user_id")
    client := hub.NewClient(conn, userID)

    // Join user-specific room for targeted notifications
    client.Join("user:" + userID)

    // Also join broadcast room
    client.Join("all")

    defer client.Close()

    // Keep connection alive with config
    cfg := websocket.DefaultConfig()
    websocket.RunWithConfig(r.Context(), conn, cfg, func(ctx context.Context, msgType websocket.MessageType, data []byte) error {
        // Client doesn't send messages in this case
        return nil
    })
}

func handleSendNotification(w http.ResponseWriter, r *http.Request) {
    var req struct {
        UserID string `json:"user_id"` // Empty for broadcast
        Title  string `json:"title"`
        Body   string `json:"body"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    notification := Notification{
        ID:        generateID(),
        Title:     req.Title,
        Body:      req.Body,
        Timestamp: time.Now(),
    }

    ctx := context.Background()

    if req.UserID != "" {
        // Send to specific user
        room := hub.Room("user:" + req.UserID)
        if room != nil {
            room.BroadcastEvent(ctx, "notification", notification)
        }
    } else {
        // Broadcast to all
        hub.Room("all").BroadcastEvent(ctx, "notification", notification)
    }

    w.WriteHeader(http.StatusOK)
}
```

### Game Lobby with Router

```go
package main

import (
    "context"
    "net/http"

    "github.com/dalemusser/waffle/websocket"
    "github.com/go-chi/chi/v5"
)

var (
    hub    = websocket.NewHub()
    router = websocket.NewRouter()
)

func init() {
    router.Handle("join_game", handleJoinGame)
    router.Handle("leave_game", handleLeaveGame)
    router.Handle("game_action", handleGameAction)
    router.Handle("chat", handleChat)

    router.Default(func(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
        return client.SendTypedMessage(ctx, "error", map[string]string{
            "message": "unknown command: " + msg.Type,
        })
    })
}

func main() {
    r := chi.NewRouter()
    r.Get("/ws/game", handleGame)
    http.ListenAndServe(":8080", r)
}

func handleGame(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, nil)
    if err != nil {
        return
    }

    playerID := r.URL.Query().Get("player_id")
    client := hub.NewClient(conn, playerID)
    defer client.Close()

    // Join lobby
    client.Join("lobby")

    // Run with router
    router.RunClient(r.Context(), client)
}

func handleJoinGame(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    var payload struct {
        GameID string `json:"game_id"`
    }
    msg.ParsePayload(&payload)

    // Leave lobby, join game room
    client.Leave("lobby")
    client.Join("game:" + payload.GameID)

    // Notify others in game
    hub.Room("game:" + payload.GameID).BroadcastExcept(ctx, client, websocket.MessageText,
        []byte(`{"type":"player_joined","player":"`+client.ID()+`"}`))

    return client.SendTypedMessage(ctx, "joined", map[string]string{
        "game_id": payload.GameID,
    })
}

func handleLeaveGame(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    for _, room := range client.Rooms() {
        if room != "lobby" {
            client.Leave(room)
        }
    }
    client.Join("lobby")
    return nil
}

func handleGameAction(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    // Forward action to all players in the same game
    for _, room := range client.Rooms() {
        if room != "lobby" {
            r := hub.Room(room)
            if r != nil {
                data, _ := msg.Marshal()
                r.Broadcast(ctx, websocket.MessageText, data)
            }
        }
    }
    return nil
}

func handleChat(ctx context.Context, client *websocket.Client, msg *websocket.Message) error {
    var payload struct {
        Text string `json:"text"`
    }
    msg.ParsePayload(&payload)

    chatMsg := map[string]string{
        "player": client.ID(),
        "text":   payload.Text,
    }

    // Send to all rooms the client is in
    for _, room := range client.Rooms() {
        hub.Room(room).BroadcastJSON(ctx, chatMsg)
    }
    return nil
}
```

---

## Client-Side JavaScript

```javascript
// Connect
const ws = new WebSocket('ws://localhost:8080/ws?user=alice');

// Send structured message
ws.send(JSON.stringify({
    type: 'chat',
    payload: { room: 'general', text: 'Hello!' }
}));

// Handle messages
ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    switch (msg.type) {
        case 'chat':
            console.log(`${msg.payload.user}: ${msg.payload.text}`);
            break;
        case 'user_joined':
            console.log(`${msg.data.user} joined`);
            break;
    }
};

// Handle events
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.name) {
        // It's an event
        console.log(`Event: ${data.name}`, data.data);
    }
};
```

---

## See Also

- [sse](../sse/sse.md) — Server-Sent Events (one-way server-to-client)
- [session](../session/session.md) — Session management for auth
- [requestid](../requestid/requestid.md) — Request ID for tracing
