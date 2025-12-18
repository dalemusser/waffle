// mq/rabbitmq/rabbitmq.go
package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection wraps an AMQP connection with convenience methods.
type Connection struct {
	*amqp.Connection
}

// Channel wraps an AMQP channel with convenience methods.
type Channel struct {
	*amqp.Channel
}

// Connect opens a RabbitMQ connection using the given URL and timeout.
// It verifies the connection is usable before returning.
//
// The caller is responsible for calling conn.Close() when done.
//
// URL format:
//
//	amqp://user:pass@host:port/vhost
//	amqp://guest:guest@localhost:5672/
//	amqps://user:pass@host:port/vhost (TLS)
func Connect(url string, timeout time.Duration) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create connection with timeout via context
	done := make(chan struct{})
	var conn *amqp.Connection
	var err error

	go func() {
		conn, err = amqp.Dial(url)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			return nil, err
		}
		return &Connection{conn}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("connection timeout: %w", ctx.Err())
	}
}

// ConnectWithConfig opens a RabbitMQ connection with custom configuration.
// Use this when you need TLS, custom heartbeat, or other advanced settings.
//
// The caller is responsible for calling conn.Close() when done.
func ConnectWithConfig(url string, config amqp.Config, timeout time.Duration) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct{})
	var conn *amqp.Connection
	var err error

	go func() {
		conn, err = amqp.DialConfig(url, config)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			return nil, err
		}
		return &Connection{conn}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("connection timeout: %w", ctx.Err())
	}
}

// Channel opens a new channel on the connection.
//
// The caller is responsible for calling ch.Close() when done.
func (c *Connection) Channel() (*Channel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return &Channel{ch}, nil
}

// DeclareQueue declares a queue with common defaults for reliable messaging.
// The queue is durable (survives broker restart) and not auto-deleted.
//
// Returns the queue for inspection (message count, consumer count).
func (ch *Channel) DeclareQueue(name string) (amqp.Queue, error) {
	return ch.QueueDeclare(
		name,  // name
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

// DeclareQueueWithOptions declares a queue with custom options.
func (ch *Channel) DeclareQueueWithOptions(name string, opts QueueOptions) (amqp.Queue, error) {
	return ch.QueueDeclare(
		name,
		opts.Durable,
		opts.AutoDelete,
		opts.Exclusive,
		opts.NoWait,
		opts.Args,
	)
}

// QueueOptions configures queue declaration.
type QueueOptions struct {
	// Durable queues survive broker restart. Default: true
	Durable bool
	// AutoDelete removes the queue when last consumer disconnects. Default: false
	AutoDelete bool
	// Exclusive queues are only accessible by the declaring connection. Default: false
	Exclusive bool
	// NoWait doesn't wait for server confirmation. Default: false
	NoWait bool
	// Args are additional queue arguments (e.g., x-message-ttl, x-dead-letter-exchange)
	Args amqp.Table
}

// DefaultQueueOptions returns sensible defaults for reliable messaging.
func DefaultQueueOptions() QueueOptions {
	return QueueOptions{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	}
}

// TransientQueueOptions returns options for temporary queues that don't need persistence.
func TransientQueueOptions() QueueOptions {
	return QueueOptions{
		Durable:    false,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
	}
}

// DeclareExchange declares an exchange with common defaults.
// The exchange is durable and not auto-deleted.
func (ch *Channel) DeclareExchange(name, kind string) error {
	return ch.ExchangeDeclare(
		name,  // name
		kind,  // type: direct, fanout, topic, headers
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

// DeclareExchangeWithOptions declares an exchange with custom options.
func (ch *Channel) DeclareExchangeWithOptions(name, kind string, opts ExchangeOptions) error {
	return ch.ExchangeDeclare(
		name,
		kind,
		opts.Durable,
		opts.AutoDelete,
		opts.Internal,
		opts.NoWait,
		opts.Args,
	)
}

// ExchangeOptions configures exchange declaration.
type ExchangeOptions struct {
	// Durable exchanges survive broker restart. Default: true
	Durable bool
	// AutoDelete removes the exchange when last queue is unbound. Default: false
	AutoDelete bool
	// Internal exchanges cannot receive messages directly from publishers. Default: false
	Internal bool
	// NoWait doesn't wait for server confirmation. Default: false
	NoWait bool
	// Args are additional exchange arguments
	Args amqp.Table
}

// DefaultExchangeOptions returns sensible defaults for reliable messaging.
func DefaultExchangeOptions() ExchangeOptions {
	return ExchangeOptions{
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
}

// Publish sends a message to an exchange with the given routing key.
// The message is marked as persistent for reliable delivery.
func (ch *Channel) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	return ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/octet-stream",
			Body:         body,
		},
	)
}

// PublishJSON sends a JSON message to an exchange with the given routing key.
// The message is marked as persistent with JSON content type.
func (ch *Channel) PublishJSON(ctx context.Context, exchange, routingKey string, body []byte) error {
	return ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}

// PublishWithOptions sends a message with full control over publishing options.
func (ch *Channel) PublishWithOptions(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
}

// Consume starts consuming messages from a queue.
// Returns a channel of deliveries. The consumer auto-acknowledges messages.
//
// For manual acknowledgment, use ConsumeWithOptions with AutoAck: false.
func (ch *Channel) Consume(queue, consumer string) (<-chan amqp.Delivery, error) {
	return ch.Channel.Consume(
		queue,
		consumer,
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

// ConsumeWithOptions starts consuming messages with custom options.
func (ch *Channel) ConsumeWithOptions(queue, consumer string, opts ConsumeOptions) (<-chan amqp.Delivery, error) {
	return ch.Channel.Consume(
		queue,
		consumer,
		opts.AutoAck,
		opts.Exclusive,
		opts.NoLocal,
		opts.NoWait,
		opts.Args,
	)
}

// ConsumeOptions configures message consumption.
type ConsumeOptions struct {
	// AutoAck automatically acknowledges messages. Default: true
	// Set to false for manual acknowledgment (recommended for reliability)
	AutoAck bool
	// Exclusive ensures only this consumer can access the queue. Default: false
	Exclusive bool
	// NoLocal prevents delivery of messages published on the same connection. Default: false
	NoLocal bool
	// NoWait doesn't wait for server confirmation. Default: false
	NoWait bool
	// Args are additional consumer arguments
	Args amqp.Table
}

// DefaultConsumeOptions returns options for simple auto-ack consumption.
func DefaultConsumeOptions() ConsumeOptions {
	return ConsumeOptions{
		AutoAck:   true,
		Exclusive: false,
		NoLocal:   false,
		NoWait:    false,
	}
}

// ReliableConsumeOptions returns options for reliable consumption with manual acknowledgment.
func ReliableConsumeOptions() ConsumeOptions {
	return ConsumeOptions{
		AutoAck:   false,
		Exclusive: false,
		NoLocal:   false,
		NoWait:    false,
	}
}

// SetQos sets the prefetch count for the channel.
// This controls how many messages the server will send before waiting for acknowledgments.
// Recommended for manual-ack consumers to prevent overwhelming the consumer.
func (ch *Channel) SetQos(prefetchCount int) error {
	return ch.Qos(
		prefetchCount, // prefetch count
		0,             // prefetch size (0 = no limit)
		false,         // global (false = per-consumer)
	)
}

// BindQueue binds a queue to an exchange with a routing key.
func (ch *Channel) BindQueue(queue, routingKey, exchange string) error {
	return ch.QueueBind(queue, routingKey, exchange, false, nil)
}

// HealthCheck returns a health check function compatible with the health package.
//
// Example:
//
//	health.Mount(r, map[string]health.Check{
//	    "rabbitmq": rabbitmq.HealthCheck(conn),
//	}, logger)
func HealthCheck(conn *Connection) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if conn.IsClosed() {
			return fmt.Errorf("connection closed")
		}
		return nil
	}
}

// Exchange type constants for convenience.
const (
	ExchangeDirect  = "direct"
	ExchangeFanout  = "fanout"
	ExchangeTopic   = "topic"
	ExchangeHeaders = "headers"
)
