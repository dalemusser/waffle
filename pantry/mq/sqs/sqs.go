// mq/sqs/sqs.go
package sqs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Client wraps the SQS client with convenience methods.
type Client struct {
	*sqs.Client
}

// Connect creates an SQS client using default AWS credential chain.
// Credentials are loaded from environment variables, shared credentials file,
// IAM role, etc.
//
// The timeout applies to loading the AWS configuration.
func Connect(ctx context.Context, region string, timeout time.Duration) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Client{sqs.NewFromConfig(cfg)}, nil
}

// ConnectWithCredentials creates an SQS client with explicit credentials.
// Use this for testing or when you can't use the default credential chain.
func ConnectWithCredentials(ctx context.Context, region, accessKey, secretKey string, timeout time.Duration) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Client{sqs.NewFromConfig(cfg)}, nil
}

// ConnectWithEndpoint creates an SQS client with a custom endpoint.
// Use this for LocalStack, ElasticMQ, or other SQS-compatible services.
func ConnectWithEndpoint(ctx context.Context, region, endpoint string, timeout time.Duration) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Client{client}, nil
}

// ConnectLocalStack creates an SQS client configured for LocalStack.
// Default LocalStack SQS endpoint is http://localhost:4566.
func ConnectLocalStack(ctx context.Context, endpoint string, timeout time.Duration) (*Client, error) {
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Client{client}, nil
}

// GetQueueURL retrieves the URL for a queue by name.
// Queue URLs are required for most SQS operations.
func (c *Client) GetQueueURL(ctx context.Context, queueName string) (string, error) {
	result, err := c.Client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
	}
	return *result.QueueUrl, nil
}

// CreateQueue creates a standard queue with default settings.
// Returns the queue URL.
func (c *Client) CreateQueue(ctx context.Context, name string) (string, error) {
	return c.CreateQueueWithOptions(ctx, name, QueueOptions{})
}

// CreateFIFOQueue creates a FIFO queue.
// FIFO queue names must end with ".fifo".
// Returns the queue URL.
func (c *Client) CreateFIFOQueue(ctx context.Context, name string) (string, error) {
	return c.CreateQueueWithOptions(ctx, name, QueueOptions{
		FIFO:                  true,
		ContentDeduplication:  true,
		DeduplicationScope:    "messageGroup",
		FIFOThroughputLimit:   "perMessageGroupId",
	})
}

// CreateQueueWithOptions creates a queue with custom settings.
// Returns the queue URL.
func (c *Client) CreateQueueWithOptions(ctx context.Context, name string, opts QueueOptions) (string, error) {
	attrs := make(map[string]string)

	if opts.VisibilityTimeout > 0 {
		attrs["VisibilityTimeout"] = fmt.Sprintf("%d", int(opts.VisibilityTimeout.Seconds()))
	}
	if opts.MessageRetention > 0 {
		attrs["MessageRetentionPeriod"] = fmt.Sprintf("%d", int(opts.MessageRetention.Seconds()))
	}
	if opts.MaxMessageSize > 0 {
		attrs["MaximumMessageSize"] = fmt.Sprintf("%d", opts.MaxMessageSize)
	}
	if opts.ReceiveWaitTime > 0 {
		attrs["ReceiveMessageWaitTimeSeconds"] = fmt.Sprintf("%d", int(opts.ReceiveWaitTime.Seconds()))
	}
	if opts.DelaySeconds > 0 {
		attrs["DelaySeconds"] = fmt.Sprintf("%d", int(opts.DelaySeconds.Seconds()))
	}
	if opts.DeadLetterQueueARN != "" && opts.MaxReceiveCount > 0 {
		attrs["RedrivePolicy"] = fmt.Sprintf(`{"deadLetterTargetArn":"%s","maxReceiveCount":"%d"}`,
			opts.DeadLetterQueueARN, opts.MaxReceiveCount)
	}
	if opts.FIFO {
		attrs["FifoQueue"] = "true"
		if opts.ContentDeduplication {
			attrs["ContentBasedDeduplication"] = "true"
		}
		if opts.DeduplicationScope != "" {
			attrs["DeduplicationScope"] = opts.DeduplicationScope
		}
		if opts.FIFOThroughputLimit != "" {
			attrs["FifoThroughputLimit"] = opts.FIFOThroughputLimit
		}
	}

	result, err := c.Client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName:  aws.String(name),
		Attributes: attrs,
	})
	if err != nil {
		return "", err
	}

	return *result.QueueUrl, nil
}

// QueueOptions configures queue creation.
type QueueOptions struct {
	// VisibilityTimeout is how long a message is hidden after being received.
	// Default: 30 seconds. Range: 0 to 12 hours.
	VisibilityTimeout time.Duration

	// MessageRetention is how long messages are kept in the queue.
	// Default: 4 days. Range: 1 minute to 14 days.
	MessageRetention time.Duration

	// MaxMessageSize is the maximum message size in bytes.
	// Default: 256 KB. Range: 1 KB to 256 KB.
	MaxMessageSize int

	// ReceiveWaitTime enables long polling. Set to 1-20 seconds.
	// Default: 0 (short polling).
	ReceiveWaitTime time.Duration

	// DelaySeconds is the default delay before messages become visible.
	// Default: 0. Range: 0 to 15 minutes.
	DelaySeconds time.Duration

	// DeadLetterQueueARN is the ARN of the dead-letter queue.
	DeadLetterQueueARN string

	// MaxReceiveCount is how many times a message can be received before
	// going to the dead-letter queue.
	MaxReceiveCount int

	// FIFO enables FIFO queue behavior.
	FIFO bool

	// ContentDeduplication enables content-based deduplication for FIFO queues.
	ContentDeduplication bool

	// DeduplicationScope: "messageGroup" or "queue" for FIFO queues.
	DeduplicationScope string

	// FIFOThroughputLimit: "perQueue" or "perMessageGroupId" for FIFO queues.
	FIFOThroughputLimit string
}

// DefaultQueueOptions returns sensible defaults for standard queues.
func DefaultQueueOptions() QueueOptions {
	return QueueOptions{
		VisibilityTimeout: 30 * time.Second,
		MessageRetention:  4 * 24 * time.Hour, // 4 days
		ReceiveWaitTime:   20 * time.Second,   // Long polling
	}
}

// Send sends a message to a queue.
// Returns the message ID.
func (c *Client) Send(ctx context.Context, queueURL, body string) (string, error) {
	result, err := c.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		return "", err
	}
	return *result.MessageId, nil
}

// SendWithDelay sends a message with a delay before it becomes visible.
// Delay range: 0 to 15 minutes.
func (c *Client) SendWithDelay(ctx context.Context, queueURL, body string, delay time.Duration) (string, error) {
	result, err := c.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(queueURL),
		MessageBody:  aws.String(body),
		DelaySeconds: int32(delay.Seconds()),
	})
	if err != nil {
		return "", err
	}
	return *result.MessageId, nil
}

// SendFIFO sends a message to a FIFO queue.
// MessageGroupId is required. DeduplicationId is optional if content-based deduplication is enabled.
func (c *Client) SendFIFO(ctx context.Context, queueURL, body, groupID, deduplicationID string) (string, error) {
	input := &sqs.SendMessageInput{
		QueueUrl:       aws.String(queueURL),
		MessageBody:    aws.String(body),
		MessageGroupId: aws.String(groupID),
	}
	if deduplicationID != "" {
		input.MessageDeduplicationId = aws.String(deduplicationID)
	}

	result, err := c.Client.SendMessage(ctx, input)
	if err != nil {
		return "", err
	}
	return *result.MessageId, nil
}

// SendWithAttributes sends a message with custom attributes.
func (c *Client) SendWithAttributes(ctx context.Context, queueURL, body string, attrs map[string]string) (string, error) {
	msgAttrs := make(map[string]types.MessageAttributeValue)
	for k, v := range attrs {
		msgAttrs[k] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}

	result, err := c.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(queueURL),
		MessageBody:       aws.String(body),
		MessageAttributes: msgAttrs,
	})
	if err != nil {
		return "", err
	}
	return *result.MessageId, nil
}

// SendBatch sends multiple messages in a single request.
// Returns the IDs of successful messages and any failures.
// Maximum 10 messages per batch.
func (c *Client) SendBatch(ctx context.Context, queueURL string, messages []string) ([]string, []BatchError, error) {
	if len(messages) > 10 {
		return nil, nil, fmt.Errorf("batch size %d exceeds maximum of 10", len(messages))
	}

	entries := make([]types.SendMessageBatchRequestEntry, len(messages))
	for i, msg := range messages {
		entries[i] = types.SendMessageBatchRequestEntry{
			Id:          aws.String(fmt.Sprintf("%d", i)),
			MessageBody: aws.String(msg),
		}
	}

	result, err := c.Client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries:  entries,
	})
	if err != nil {
		return nil, nil, err
	}

	var ids []string
	for _, s := range result.Successful {
		ids = append(ids, *s.MessageId)
	}

	var failures []BatchError
	for _, f := range result.Failed {
		failures = append(failures, BatchError{
			ID:      *f.Id,
			Code:    *f.Code,
			Message: *f.Message,
		})
	}

	return ids, failures, nil
}

// BatchError represents a failure in a batch operation.
type BatchError struct {
	ID      string
	Code    string
	Message string
}

// Message represents a received SQS message.
type Message struct {
	ID            string
	Body          string
	ReceiptHandle string
	Attributes    map[string]string
	MD5           string
}

// Receive receives messages from a queue.
// Uses long polling with the configured wait time.
// Returns up to maxMessages (1-10).
func (c *Client) Receive(ctx context.Context, queueURL string, maxMessages int, waitTime time.Duration) ([]Message, error) {
	if maxMessages < 1 {
		maxMessages = 1
	}
	if maxMessages > 10 {
		maxMessages = 10
	}

	result, err := c.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: int32(maxMessages),
		WaitTimeSeconds:     int32(waitTime.Seconds()),
		AttributeNames:      []types.QueueAttributeName{types.QueueAttributeNameAll},
	})
	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(result.Messages))
	for i, m := range result.Messages {
		attrs := make(map[string]string)
		for k, v := range m.Attributes {
			attrs[string(k)] = v
		}
		messages[i] = Message{
			ID:            *m.MessageId,
			Body:          *m.Body,
			ReceiptHandle: *m.ReceiptHandle,
			Attributes:    attrs,
			MD5:           *m.MD5OfBody,
		}
	}

	return messages, nil
}

// Delete deletes a message from the queue after successful processing.
func (c *Client) Delete(ctx context.Context, queueURL, receiptHandle string) error {
	_, err := c.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}

// DeleteBatch deletes multiple messages in a single request.
// Maximum 10 messages per batch.
func (c *Client) DeleteBatch(ctx context.Context, queueURL string, receiptHandles []string) ([]BatchError, error) {
	if len(receiptHandles) > 10 {
		return nil, fmt.Errorf("batch size %d exceeds maximum of 10", len(receiptHandles))
	}

	entries := make([]types.DeleteMessageBatchRequestEntry, len(receiptHandles))
	for i, rh := range receiptHandles {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(fmt.Sprintf("%d", i)),
			ReceiptHandle: aws.String(rh),
		}
	}

	result, err := c.Client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries:  entries,
	})
	if err != nil {
		return nil, err
	}

	var failures []BatchError
	for _, f := range result.Failed {
		failures = append(failures, BatchError{
			ID:      *f.Id,
			Code:    *f.Code,
			Message: *f.Message,
		})
	}

	return failures, nil
}

// ChangeVisibility changes the visibility timeout of a message.
// Use this to extend processing time or release a message back to the queue.
func (c *Client) ChangeVisibility(ctx context.Context, queueURL, receiptHandle string, timeout time.Duration) error {
	_, err := c.Client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: int32(timeout.Seconds()),
	})
	return err
}

// Purge removes all messages from a queue.
// Can only be called once every 60 seconds.
func (c *Client) Purge(ctx context.Context, queueURL string) error {
	_, err := c.Client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return err
}

// DeleteQueue deletes a queue.
func (c *Client) DeleteQueue(ctx context.Context, queueURL string) error {
	_, err := c.Client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return err
}

// GetQueueAttributes retrieves queue attributes.
func (c *Client) GetQueueAttributes(ctx context.Context, queueURL string) (map[string]string, error) {
	result, err := c.Client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameAll},
	})
	if err != nil {
		return nil, err
	}
	return result.Attributes, nil
}

// HealthCheck returns a health check function compatible with the health package.
// It verifies connectivity by listing queues.
//
// Example:
//
//	health.Mount(r, map[string]health.Check{
//	    "sqs": sqs.HealthCheck(client),
//	}, logger)
func HealthCheck(client *Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		_, err := client.Client.ListQueues(ctx, &sqs.ListQueuesInput{
			MaxResults: aws.Int32(1),
		})
		return err
	}
}
