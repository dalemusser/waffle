// sse/broker.go
package sse

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Broker manages multiple SSE clients and enables broadcasting.
type Broker struct {
	mu       sync.RWMutex
	clients  map[*Client]struct{}
	channels map[string]*Channel
	closed   bool
	nextID   atomic.Uint64
	cfg      BrokerConfig

	// OnConnect is called when a client connects.
	OnConnect func(*Client)

	// OnDisconnect is called when a client disconnects.
	OnDisconnect func(*Client)
}

// BrokerConfig configures the broker.
type BrokerConfig struct {
	// KeepAliveInterval is how often to send keep-alive comments.
	// Default: 30 seconds.
	KeepAliveInterval time.Duration

	// RetryInterval is the reconnection interval to suggest to clients.
	// Default: 3 seconds.
	RetryInterval time.Duration

	// ClientBufferSize is the size of each client's event buffer.
	// Default: 256.
	ClientBufferSize int
}

// DefaultBrokerConfig returns sensible defaults.
func DefaultBrokerConfig() BrokerConfig {
	return BrokerConfig{
		KeepAliveInterval: 30 * time.Second,
		RetryInterval:     3 * time.Second,
		ClientBufferSize:  256,
	}
}

// NewBroker creates a new SSE broker.
func NewBroker() *Broker {
	return NewBrokerWithConfig(DefaultBrokerConfig())
}

// NewBrokerWithConfig creates a broker with custom configuration.
func NewBrokerWithConfig(cfg BrokerConfig) *Broker {
	if cfg.ClientBufferSize <= 0 {
		cfg.ClientBufferSize = 256
	}

	return &Broker{
		clients:  make(map[*Client]struct{}),
		channels: make(map[string]*Channel),
		cfg:      cfg,
	}
}

// Client represents a connected SSE client.
type Client struct {
	broker   *Broker
	stream   *Stream
	id       string
	events   chan *Event
	data     map[string]any
	mu       sync.RWMutex
	channels map[string]*Channel
	done     chan struct{}
	closed   bool
}

// ServeHTTP implements http.Handler for the broker.
// Use this to handle SSE connections directly.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.HandleRequest(w, r, nil)
}

// Handler returns an http.Handler that accepts SSE connections.
func (b *Broker) Handler() http.Handler {
	return http.HandlerFunc(b.ServeHTTP)
}

// HandleRequest handles an SSE connection with optional setup callback.
func (b *Broker) HandleRequest(w http.ResponseWriter, r *http.Request, setup func(*Client)) {
	stream, err := NewStream(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := b.addClient(stream)
	if client == nil {
		http.Error(w, "broker closed", http.StatusServiceUnavailable)
		return
	}

	if setup != nil {
		setup(client)
	}

	// Run the client
	b.runClient(r.Context(), client)
}

// HandleFunc returns an http.HandlerFunc with a setup callback.
func (b *Broker) HandleFunc(setup func(*Client)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.HandleRequest(w, r, setup)
	}
}

// addClient creates and registers a new client.
func (b *Broker) addClient(stream *Stream) *Client {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	id := b.nextID.Add(1)
	client := &Client{
		broker:   b,
		stream:   stream,
		id:       string(rune(id)), // Convert to string
		events:   make(chan *Event, b.cfg.ClientBufferSize),
		data:     make(map[string]any),
		channels: make(map[string]*Channel),
		done:     make(chan struct{}),
	}
	// Use numeric ID
	client.id = formatClientID(id)

	b.clients[client] = struct{}{}

	if b.OnConnect != nil {
		go b.OnConnect(client)
	}

	return client
}

func formatClientID(id uint64) string {
	const digits = "0123456789"
	if id == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = digits[id%10]
		id /= 10
	}
	return string(buf[i:])
}

// removeClient removes a client from the broker.
func (b *Broker) removeClient(client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.clients[client]; !exists {
		return
	}

	// Remove from all channels
	client.mu.Lock()
	for _, ch := range client.channels {
		ch.mu.Lock()
		delete(ch.clients, client)
		ch.mu.Unlock()
	}
	client.channels = make(map[string]*Channel)
	client.closed = true
	client.mu.Unlock()

	close(client.done)
	delete(b.clients, client)

	if b.OnDisconnect != nil {
		go b.OnDisconnect(client)
	}
}

// runClient runs the event loop for a client.
func (b *Broker) runClient(ctx context.Context, client *Client) {
	defer b.removeClient(client)

	// Send initial retry interval
	if b.cfg.RetryInterval > 0 {
		client.stream.SendRetry(int(b.cfg.RetryInterval.Milliseconds()))
	}

	var keepAliveTicker *time.Ticker
	var keepAliveChan <-chan time.Time

	if b.cfg.KeepAliveInterval > 0 {
		keepAliveTicker = time.NewTicker(b.cfg.KeepAliveInterval)
		defer keepAliveTicker.Stop()
		keepAliveChan = keepAliveTicker.C
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-client.events:
			if !ok {
				return
			}
			if err := client.stream.Send(event); err != nil {
				return
			}

		case <-keepAliveChan:
			if err := client.stream.SendComment("keep-alive"); err != nil {
				return
			}
		}
	}
}

// Close closes the broker and disconnects all clients.
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true

	for client := range b.clients {
		close(client.events)
	}

	b.clients = make(map[*Client]struct{})
	b.channels = make(map[string]*Channel)
}

// Clients returns the number of connected clients.
func (b *Broker) Clients() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Broadcast sends an event to all connected clients.
func (b *Broker) Broadcast(event *Event) {
	b.mu.RLock()
	clients := make([]*Client, 0, len(b.clients))
	for client := range b.clients {
		clients = append(clients, client)
	}
	b.mu.RUnlock()

	for _, client := range clients {
		client.Send(event)
	}
}

// BroadcastData sends a simple data event to all clients.
func (b *Broker) BroadcastData(data string) {
	b.Broadcast(NewEvent(data))
}

// BroadcastEvent sends a typed event to all clients.
func (b *Broker) BroadcastEvent(eventType, data string) {
	b.Broadcast(NewEventWithType(eventType, data))
}

// BroadcastJSON sends a JSON event to all clients.
func (b *Broker) BroadcastJSON(eventType string, v any) error {
	event, err := NewJSONEvent(eventType, v)
	if err != nil {
		return err
	}
	b.Broadcast(event)
	return nil
}

// ForEach iterates over all connected clients.
func (b *Broker) ForEach(fn func(*Client)) {
	b.mu.RLock()
	clients := make([]*Client, 0, len(b.clients))
	for client := range b.clients {
		clients = append(clients, client)
	}
	b.mu.RUnlock()

	for _, client := range clients {
		fn(client)
	}
}

// GetChannel returns a channel by name, creating it if it doesn't exist.
func (b *Broker) GetChannel(name string) *Channel {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, exists := b.channels[name]; exists {
		return ch
	}

	ch := &Channel{
		broker:  b,
		name:    name,
		clients: make(map[*Client]struct{}),
	}
	b.channels[name] = ch
	return ch
}

// Channel returns a channel by name, or nil if it doesn't exist.
func (b *Broker) Channel(name string) *Channel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.channels[name]
}

// DeleteChannel removes a channel.
func (b *Broker) DeleteChannel(name string) {
	b.mu.Lock()
	ch, exists := b.channels[name]
	if !exists {
		b.mu.Unlock()
		return
	}
	delete(b.channels, name)
	b.mu.Unlock()

	// Remove all clients from channel
	ch.mu.Lock()
	for client := range ch.clients {
		client.mu.Lock()
		delete(client.channels, name)
		client.mu.Unlock()
	}
	ch.clients = make(map[*Client]struct{})
	ch.mu.Unlock()
}

// Channels returns the names of all channels.
func (b *Broker) Channels() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	names := make([]string, 0, len(b.channels))
	for name := range b.channels {
		names = append(names, name)
	}
	return names
}

// Channel represents a group of clients for targeted broadcasting.
type Channel struct {
	broker  *Broker
	name    string
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return c.name
}

// Subscribe adds a client to the channel.
func (c *Channel) Subscribe(client *Client) {
	c.mu.Lock()
	c.clients[client] = struct{}{}
	c.mu.Unlock()

	client.mu.Lock()
	client.channels[c.name] = c
	client.mu.Unlock()
}

// Unsubscribe removes a client from the channel.
func (c *Channel) Unsubscribe(client *Client) {
	c.mu.Lock()
	delete(c.clients, client)
	c.mu.Unlock()

	client.mu.Lock()
	delete(client.channels, c.name)
	client.mu.Unlock()
}

// Has returns true if the client is subscribed to the channel.
func (c *Channel) Has(client *Client) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.clients[client]
	return exists
}

// Size returns the number of subscribers.
func (c *Channel) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clients)
}

// Broadcast sends an event to all channel subscribers.
func (c *Channel) Broadcast(event *Event) {
	c.mu.RLock()
	clients := make([]*Client, 0, len(c.clients))
	for client := range c.clients {
		clients = append(clients, client)
	}
	c.mu.RUnlock()

	for _, client := range clients {
		client.Send(event)
	}
}

// BroadcastData sends a simple data event to all subscribers.
func (c *Channel) BroadcastData(data string) {
	c.Broadcast(NewEvent(data))
}

// BroadcastEvent sends a typed event to all subscribers.
func (c *Channel) BroadcastEvent(eventType, data string) {
	c.Broadcast(NewEventWithType(eventType, data))
}

// BroadcastJSON sends a JSON event to all subscribers.
func (c *Channel) BroadcastJSON(eventType string, v any) error {
	event, err := NewJSONEvent(eventType, v)
	if err != nil {
		return err
	}
	c.Broadcast(event)
	return nil
}

// ForEach iterates over all subscribers.
func (c *Channel) ForEach(fn func(*Client)) {
	c.mu.RLock()
	clients := make([]*Client, 0, len(c.clients))
	for client := range c.clients {
		clients = append(clients, client)
	}
	c.mu.RUnlock()

	for _, client := range clients {
		fn(client)
	}
}

// Client methods

// ID returns the client's unique identifier.
func (c *Client) ID() string {
	return c.id
}

// Stream returns the underlying SSE stream.
func (c *Client) Stream() *Stream {
	return c.stream
}

// LastEventID returns the last event ID from the client (for reconnection).
func (c *Client) LastEventID() string {
	return c.stream.LastEventID
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

// Send sends an event to this client.
func (c *Client) Send(event *Event) bool {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false
	}
	c.mu.RUnlock()

	select {
	case c.events <- event:
		return true
	default:
		// Buffer full, drop event
		return false
	}
}

// SendData sends a simple data event.
func (c *Client) SendData(data string) bool {
	return c.Send(NewEvent(data))
}

// SendEvent sends a typed event.
func (c *Client) SendEvent(eventType, data string) bool {
	return c.Send(NewEventWithType(eventType, data))
}

// SendJSON sends a JSON event.
func (c *Client) SendJSON(eventType string, v any) error {
	event, err := NewJSONEvent(eventType, v)
	if err != nil {
		return err
	}
	c.Send(event)
	return nil
}

// Subscribe subscribes the client to a channel.
func (c *Client) Subscribe(channelName string) {
	ch := c.broker.GetChannel(channelName)
	ch.Subscribe(c)
}

// Unsubscribe unsubscribes the client from a channel.
func (c *Client) Unsubscribe(channelName string) {
	c.mu.RLock()
	ch, exists := c.channels[channelName]
	c.mu.RUnlock()

	if exists {
		ch.Unsubscribe(c)
	}
}

// Channels returns the names of channels the client is subscribed to.
func (c *Client) Channels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.channels))
	for name := range c.channels {
		names = append(names, name)
	}
	return names
}

// InChannel returns true if the client is subscribed to the channel.
func (c *Client) InChannel(channelName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.channels[channelName]
	return exists
}

// Done returns a channel that's closed when the client disconnects.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Close closes the client connection.
func (c *Client) Close() {
	c.broker.removeClient(c)
}
