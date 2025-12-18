// websocket/message.go
package websocket

import (
	"context"
	"encoding/json"
)

// Message represents a structured WebSocket message with a type and payload.
type Message struct {
	// Type identifies the message type (e.g., "chat", "notification", "error").
	Type string `json:"type"`

	// Payload contains the message data.
	Payload json.RawMessage `json:"payload,omitempty"`

	// ID is an optional message identifier for request/response correlation.
	ID string `json:"id,omitempty"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType string, payload any) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:    msgType,
		Payload: data,
	}, nil
}

// NewMessageWithID creates a new message with type, payload, and ID.
func NewMessageWithID(id, msgType string, payload any) (*Message, error) {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return nil, err
	}
	msg.ID = id
	return msg, nil
}

// ParsePayload unmarshals the message payload into the provided value.
func (m *Message) ParsePayload(v any) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// Marshal serializes the message to JSON bytes.
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalMessage parses JSON data into a Message.
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ReadJSON reads a JSON message from the connection.
func (c *Conn) ReadJSON(ctx context.Context, v any) error {
	_, data, err := c.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// WriteJSON writes a JSON message to the connection.
func (c *Conn) WriteJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, MessageText, data)
}

// ReadMessage reads a structured Message from the connection.
func (c *Conn) ReadMessage(ctx context.Context) (*Message, error) {
	_, data, err := c.Read(ctx)
	if err != nil {
		return nil, err
	}
	return UnmarshalMessage(data)
}

// WriteMessage writes a structured Message to the connection.
func (c *Conn) WriteMessage(ctx context.Context, msg *Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}
	return c.Write(ctx, MessageText, data)
}

// SendJSON sends a JSON message to the client.
func (c *Client) SendJSON(ctx context.Context, v any) error {
	return c.conn.WriteJSON(ctx, v)
}

// SendMessage sends a structured Message to the client.
func (c *Client) SendMessage(ctx context.Context, msg *Message) error {
	return c.conn.WriteMessage(ctx, msg)
}

// SendTypedMessage creates and sends a message with the given type and payload.
func (c *Client) SendTypedMessage(ctx context.Context, msgType string, payload any) error {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return err
	}
	return c.SendMessage(ctx, msg)
}

// BroadcastJSON sends a JSON message to all connected clients.
func (h *Hub) BroadcastJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return h.Broadcast(ctx, MessageText, data)
}

// BroadcastMessage sends a structured Message to all connected clients.
func (h *Hub) BroadcastMessage(ctx context.Context, msg *Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}
	return h.Broadcast(ctx, MessageText, data)
}

// BroadcastTypedMessage creates and broadcasts a message with the given type and payload.
func (h *Hub) BroadcastTypedMessage(ctx context.Context, msgType string, payload any) error {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return err
	}
	return h.BroadcastMessage(ctx, msg)
}

// BroadcastJSON sends a JSON message to all clients in the room.
func (r *Room) BroadcastJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.Broadcast(ctx, MessageText, data)
}

// BroadcastMessage sends a structured Message to all clients in the room.
func (r *Room) BroadcastMessage(ctx context.Context, msg *Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}
	return r.Broadcast(ctx, MessageText, data)
}

// BroadcastTypedMessage creates and broadcasts a message with the given type and payload.
func (r *Room) BroadcastTypedMessage(ctx context.Context, msgType string, payload any) error {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return err
	}
	return r.BroadcastMessage(ctx, msg)
}

// MessageHandler is a function that handles a specific message type.
type MessageHandler func(ctx context.Context, client *Client, msg *Message) error

// Router routes messages to handlers based on message type.
type Router struct {
	handlers       map[string]MessageHandler
	defaultHandler MessageHandler
}

// NewRouter creates a new message router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]MessageHandler),
	}
}

// Handle registers a handler for a message type.
func (r *Router) Handle(msgType string, handler MessageHandler) {
	r.handlers[msgType] = handler
}

// Default sets the default handler for unregistered message types.
func (r *Router) Default(handler MessageHandler) {
	r.defaultHandler = handler
}

// Route routes a message to the appropriate handler.
func (r *Router) Route(ctx context.Context, client *Client, msg *Message) error {
	handler, exists := r.handlers[msg.Type]
	if !exists {
		if r.defaultHandler != nil {
			return r.defaultHandler(ctx, client, msg)
		}
		return nil // Silently ignore unknown message types
	}
	return handler(ctx, client, msg)
}

// HandleRaw handles raw message data by parsing and routing it.
func (r *Router) HandleRaw(ctx context.Context, client *Client, data []byte) error {
	msg, err := UnmarshalMessage(data)
	if err != nil {
		return err
	}
	return r.Route(ctx, client, msg)
}

// RunClient runs a message loop for a client using the router.
func (r *Router) RunClient(ctx context.Context, client *Client) error {
	for {
		msg, err := client.conn.ReadMessage(ctx)
		if err != nil {
			return err
		}

		if err := r.Route(ctx, client, msg); err != nil {
			return err
		}
	}
}

// Event is a simple event message for pub/sub patterns.
type Event struct {
	Name string `json:"name"`
	Data any    `json:"data,omitempty"`
}

// NewEvent creates a new event.
func NewEvent(name string, data any) *Event {
	return &Event{Name: name, Data: data}
}

// SendEvent sends an event to the client.
func (c *Client) SendEvent(ctx context.Context, name string, data any) error {
	return c.SendJSON(ctx, NewEvent(name, data))
}

// BroadcastEvent broadcasts an event to all connected clients.
func (h *Hub) BroadcastEvent(ctx context.Context, name string, data any) error {
	return h.BroadcastJSON(ctx, NewEvent(name, data))
}

// BroadcastEvent broadcasts an event to all clients in the room.
func (r *Room) BroadcastEvent(ctx context.Context, name string, data any) error {
	return r.BroadcastJSON(ctx, NewEvent(name, data))
}
