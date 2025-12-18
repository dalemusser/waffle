// websocket/hub.go
package websocket

import (
	"context"
	"sync"
)

// Hub manages WebSocket connections and enables broadcasting.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	rooms   map[string]*Room
	closed  bool

	// OnConnect is called when a client connects.
	OnConnect func(*Client)

	// OnDisconnect is called when a client disconnects.
	OnDisconnect func(*Client)
}

// NewHub creates a new connection hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		rooms:   make(map[string]*Room),
	}
}

// Client represents a connected WebSocket client.
type Client struct {
	hub  *Hub
	conn *Conn
	id   string
	data map[string]any

	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewClient creates a new client and registers it with the hub.
func (h *Hub) NewClient(conn *Conn, id string) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	client := &Client{
		hub:   h,
		conn:  conn,
		id:    id,
		data:  make(map[string]any),
		rooms: make(map[string]*Room),
	}

	h.clients[client] = struct{}{}

	// Set up close handler
	conn.OnClose = func() {
		h.removeClient(client)
	}

	if h.OnConnect != nil {
		go h.OnConnect(client)
	}

	return client
}

// removeClient removes a client from the hub and all rooms.
func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client]; !exists {
		return
	}

	// Remove from all rooms
	client.mu.Lock()
	for _, room := range client.rooms {
		room.mu.Lock()
		delete(room.clients, client)
		room.mu.Unlock()
	}
	client.rooms = make(map[string]*Room)
	client.mu.Unlock()

	delete(h.clients, client)

	if h.OnDisconnect != nil {
		go h.OnDisconnect(client)
	}
}

// Close closes the hub and all client connections.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true

	for client := range h.clients {
		client.conn.Close()
	}

	h.clients = make(map[*Client]struct{})
	h.rooms = make(map[string]*Room)
}

// Clients returns the number of connected clients.
func (h *Hub) Clients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(ctx context.Context, msgType MessageType, data []byte) error {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	var lastErr error
	for _, client := range clients {
		if err := client.Send(ctx, msgType, data); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BroadcastText sends a text message to all connected clients.
func (h *Hub) BroadcastText(ctx context.Context, msg string) error {
	return h.Broadcast(ctx, MessageText, []byte(msg))
}

// BroadcastBinary sends a binary message to all connected clients.
func (h *Hub) BroadcastBinary(ctx context.Context, data []byte) error {
	return h.Broadcast(ctx, MessageBinary, data)
}

// BroadcastExcept sends a message to all clients except the specified one.
func (h *Hub) BroadcastExcept(ctx context.Context, except *Client, msgType MessageType, data []byte) error {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		if client != except {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	var lastErr error
	for _, client := range clients {
		if err := client.Send(ctx, msgType, data); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// GetRoom returns a room by name, creating it if it doesn't exist.
func (h *Hub) GetRoom(name string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[name]; exists {
		return room
	}

	room := &Room{
		hub:     h,
		name:    name,
		clients: make(map[*Client]struct{}),
	}
	h.rooms[name] = room
	return room
}

// Room returns a room by name, or nil if it doesn't exist.
func (h *Hub) Room(name string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[name]
}

// DeleteRoom removes a room and removes all clients from it.
func (h *Hub) DeleteRoom(name string) {
	h.mu.Lock()
	room, exists := h.rooms[name]
	if !exists {
		h.mu.Unlock()
		return
	}
	delete(h.rooms, name)
	h.mu.Unlock()

	room.mu.Lock()
	for client := range room.clients {
		client.mu.Lock()
		delete(client.rooms, name)
		client.mu.Unlock()
	}
	room.clients = make(map[*Client]struct{})
	room.mu.Unlock()
}

// Rooms returns the names of all rooms.
func (h *Hub) Rooms() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	names := make([]string, 0, len(h.rooms))
	for name := range h.rooms {
		names = append(names, name)
	}
	return names
}

// ForEach iterates over all connected clients.
func (h *Hub) ForEach(fn func(*Client)) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		fn(client)
	}
}

// Room represents a group of clients that can receive broadcasts.
type Room struct {
	hub     *Hub
	name    string
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// Name returns the room name.
func (r *Room) Name() string {
	return r.name
}

// Join adds a client to the room.
func (r *Room) Join(client *Client) {
	r.mu.Lock()
	r.clients[client] = struct{}{}
	r.mu.Unlock()

	client.mu.Lock()
	client.rooms[r.name] = r
	client.mu.Unlock()
}

// Leave removes a client from the room.
func (r *Room) Leave(client *Client) {
	r.mu.Lock()
	delete(r.clients, client)
	r.mu.Unlock()

	client.mu.Lock()
	delete(client.rooms, r.name)
	client.mu.Unlock()
}

// Has returns true if the client is in the room.
func (r *Room) Has(client *Client) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.clients[client]
	return exists
}

// Size returns the number of clients in the room.
func (r *Room) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Broadcast sends a message to all clients in the room.
func (r *Room) Broadcast(ctx context.Context, msgType MessageType, data []byte) error {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	r.mu.RUnlock()

	var lastErr error
	for _, client := range clients {
		if err := client.Send(ctx, msgType, data); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BroadcastText sends a text message to all clients in the room.
func (r *Room) BroadcastText(ctx context.Context, msg string) error {
	return r.Broadcast(ctx, MessageText, []byte(msg))
}

// BroadcastExcept sends a message to all room clients except the specified one.
func (r *Room) BroadcastExcept(ctx context.Context, except *Client, msgType MessageType, data []byte) error {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		if client != except {
			clients = append(clients, client)
		}
	}
	r.mu.RUnlock()

	var lastErr error
	for _, client := range clients {
		if err := client.Send(ctx, msgType, data); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ForEach iterates over all clients in the room.
func (r *Room) ForEach(fn func(*Client)) {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	r.mu.RUnlock()

	for _, client := range clients {
		fn(client)
	}
}

// Client methods

// ID returns the client's identifier.
func (c *Client) ID() string {
	return c.id
}

// Conn returns the underlying WebSocket connection.
func (c *Client) Conn() *Conn {
	return c.conn
}

// Set stores a value associated with the client.
func (c *Client) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get retrieves a value associated with the client.
func (c *Client) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

// GetString retrieves a string value associated with the client.
func (c *Client) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Send sends a message to the client.
func (c *Client) Send(ctx context.Context, msgType MessageType, data []byte) error {
	return c.conn.Write(ctx, msgType, data)
}

// SendText sends a text message to the client.
func (c *Client) SendText(ctx context.Context, msg string) error {
	return c.conn.WriteText(ctx, msg)
}

// SendBinary sends a binary message to the client.
func (c *Client) SendBinary(ctx context.Context, data []byte) error {
	return c.conn.WriteBinary(ctx, data)
}

// Close closes the client's connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Join adds the client to a room.
func (c *Client) Join(roomName string) {
	room := c.hub.GetRoom(roomName)
	room.Join(c)
}

// Leave removes the client from a room.
func (c *Client) Leave(roomName string) {
	c.mu.RLock()
	room, exists := c.rooms[roomName]
	c.mu.RUnlock()

	if exists {
		room.Leave(c)
	}
}

// Rooms returns the names of rooms the client is in.
func (c *Client) Rooms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.rooms))
	for name := range c.rooms {
		names = append(names, name)
	}
	return names
}

// InRoom returns true if the client is in the specified room.
func (c *Client) InRoom(roomName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.rooms[roomName]
	return exists
}
