// notify/subscription.go
package notify

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Subscription represents a Web Push subscription from the client.
type Subscription struct {
	// Endpoint is the push service URL.
	Endpoint string `json:"endpoint"`

	// Keys contains the encryption keys.
	Keys SubscriptionKeys `json:"keys"`

	// ExpirationTime is when the subscription expires (optional).
	ExpirationTime *int64 `json:"expirationTime,omitempty"`
}

// SubscriptionKeys contains the encryption keys for a subscription.
type SubscriptionKeys struct {
	// P256dh is the client's public key (base64url-encoded).
	P256dh string `json:"p256dh"`

	// Auth is the authentication secret (base64url-encoded).
	Auth string `json:"auth"`
}

// NewSubscription creates a subscription from the JSON received from the browser.
func NewSubscription(jsonData []byte) (*Subscription, error) {
	var sub Subscription
	if err := json.Unmarshal(jsonData, &sub); err != nil {
		return nil, err
	}

	if err := sub.Validate(); err != nil {
		return nil, err
	}

	return &sub, nil
}

// NewSubscriptionFromParts creates a subscription from individual parts.
func NewSubscriptionFromParts(endpoint, p256dh, auth string) (*Subscription, error) {
	sub := &Subscription{
		Endpoint: endpoint,
		Keys: SubscriptionKeys{
			P256dh: p256dh,
			Auth:   auth,
		},
	}

	if err := sub.Validate(); err != nil {
		return nil, err
	}

	return sub, nil
}

// Validate validates the subscription.
func (s *Subscription) Validate() error {
	if s.Endpoint == "" {
		return ErrEndpointRequired
	}

	// Validate endpoint URL
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return ErrInvalidEndpoint
	}

	if u.Scheme != "https" {
		return ErrEndpointNotHTTPS
	}

	if s.Keys.P256dh == "" {
		return ErrP256dhRequired
	}

	if s.Keys.Auth == "" {
		return ErrAuthRequired
	}

	// Validate base64url encoding
	if _, err := base64.RawURLEncoding.DecodeString(s.Keys.P256dh); err != nil {
		return ErrInvalidP256dh
	}

	p256dhBytes, _ := base64.RawURLEncoding.DecodeString(s.Keys.P256dh)
	if len(p256dhBytes) != 65 {
		return ErrInvalidP256dh
	}

	if _, err := base64.RawURLEncoding.DecodeString(s.Keys.Auth); err != nil {
		return ErrInvalidAuth
	}

	authBytes, _ := base64.RawURLEncoding.DecodeString(s.Keys.Auth)
	if len(authBytes) != 16 {
		return ErrInvalidAuth
	}

	return nil
}

// JSON returns the subscription as JSON.
func (s *Subscription) JSON() ([]byte, error) {
	return json.Marshal(s)
}

// PushService returns the push service provider based on the endpoint.
func (s *Subscription) PushService() PushService {
	endpoint := strings.ToLower(s.Endpoint)

	switch {
	case strings.Contains(endpoint, "googleapis.com"):
		return PushServiceGoogle
	case strings.Contains(endpoint, "mozilla.com"):
		return PushServiceMozilla
	case strings.Contains(endpoint, "windows.com") || strings.Contains(endpoint, "microsoft.com"):
		return PushServiceMicrosoft
	case strings.Contains(endpoint, "apple.com"):
		return PushServiceApple
	default:
		return PushServiceUnknown
	}
}

// PushService represents known push service providers.
type PushService string

const (
	PushServiceUnknown   PushService = "unknown"
	PushServiceGoogle    PushService = "google"
	PushServiceMozilla   PushService = "mozilla"
	PushServiceMicrosoft PushService = "microsoft"
	PushServiceApple     PushService = "apple"
)

// SubscriptionStore is an interface for storing subscriptions.
type SubscriptionStore interface {
	// Save saves a subscription.
	Save(userID string, sub *Subscription) error

	// Get retrieves subscriptions for a user.
	Get(userID string) ([]*Subscription, error)

	// Delete removes a subscription.
	Delete(userID string, endpoint string) error

	// DeleteAll removes all subscriptions for a user.
	DeleteAll(userID string) error
}

// MemorySubscriptionStore is an in-memory subscription store.
type MemorySubscriptionStore struct {
	subscriptions map[string]map[string]*Subscription
}

// NewMemorySubscriptionStore creates a new in-memory subscription store.
func NewMemorySubscriptionStore() *MemorySubscriptionStore {
	return &MemorySubscriptionStore{
		subscriptions: make(map[string]map[string]*Subscription),
	}
}

// Save saves a subscription.
func (s *MemorySubscriptionStore) Save(userID string, sub *Subscription) error {
	if s.subscriptions[userID] == nil {
		s.subscriptions[userID] = make(map[string]*Subscription)
	}
	s.subscriptions[userID][sub.Endpoint] = sub
	return nil
}

// Get retrieves subscriptions for a user.
func (s *MemorySubscriptionStore) Get(userID string) ([]*Subscription, error) {
	userSubs := s.subscriptions[userID]
	if userSubs == nil {
		return nil, nil
	}

	result := make([]*Subscription, 0, len(userSubs))
	for _, sub := range userSubs {
		result = append(result, sub)
	}
	return result, nil
}

// Delete removes a subscription.
func (s *MemorySubscriptionStore) Delete(userID string, endpoint string) error {
	if s.subscriptions[userID] != nil {
		delete(s.subscriptions[userID], endpoint)
	}
	return nil
}

// DeleteAll removes all subscriptions for a user.
func (s *MemorySubscriptionStore) DeleteAll(userID string) error {
	delete(s.subscriptions, userID)
	return nil
}

// SubscriptionBuilder provides a fluent API for building notifications.
type SubscriptionBuilder struct {
	notification *Notification
}

// NewNotification creates a new notification builder.
func NewNotification(title string) *SubscriptionBuilder {
	return &SubscriptionBuilder{
		notification: &Notification{
			Title: title,
		},
	}
}

// Body sets the notification body.
func (b *SubscriptionBuilder) Body(body string) *SubscriptionBuilder {
	b.notification.Body = body
	return b
}

// Icon sets the notification icon URL.
func (b *SubscriptionBuilder) Icon(icon string) *SubscriptionBuilder {
	b.notification.Icon = icon
	return b
}

// Badge sets the notification badge URL.
func (b *SubscriptionBuilder) Badge(badge string) *SubscriptionBuilder {
	b.notification.Badge = badge
	return b
}

// Image sets the notification image URL.
func (b *SubscriptionBuilder) Image(image string) *SubscriptionBuilder {
	b.notification.Image = image
	return b
}

// Tag sets the notification tag for grouping.
func (b *SubscriptionBuilder) Tag(tag string) *SubscriptionBuilder {
	b.notification.Tag = tag
	return b
}

// Data sets custom data to pass to the service worker.
func (b *SubscriptionBuilder) Data(data any) *SubscriptionBuilder {
	b.notification.Data = data
	return b
}

// Action adds an action button.
func (b *SubscriptionBuilder) Action(action, title string, icon ...string) *SubscriptionBuilder {
	a := Action{Action: action, Title: title}
	if len(icon) > 0 {
		a.Icon = icon[0]
	}
	b.notification.Actions = append(b.notification.Actions, a)
	return b
}

// Renotify enables renotification when using the same tag.
func (b *SubscriptionBuilder) Renotify() *SubscriptionBuilder {
	b.notification.Renotify = true
	return b
}

// RequireInteraction makes the notification stay until user interacts.
func (b *SubscriptionBuilder) RequireInteraction() *SubscriptionBuilder {
	b.notification.RequireInteraction = true
	return b
}

// Silent makes the notification silent.
func (b *SubscriptionBuilder) Silent() *SubscriptionBuilder {
	b.notification.Silent = true
	return b
}

// Timestamp sets the notification timestamp.
func (b *SubscriptionBuilder) Timestamp(ts int64) *SubscriptionBuilder {
	b.notification.Timestamp = ts
	return b
}

// Dir sets the text direction.
func (b *SubscriptionBuilder) Dir(dir string) *SubscriptionBuilder {
	b.notification.Dir = dir
	return b
}

// Lang sets the language.
func (b *SubscriptionBuilder) Lang(lang string) *SubscriptionBuilder {
	b.notification.Lang = lang
	return b
}

// Vibrate sets the vibration pattern.
func (b *SubscriptionBuilder) Vibrate(pattern ...int) *SubscriptionBuilder {
	b.notification.Vibrate = pattern
	return b
}

// Build returns the notification.
func (b *SubscriptionBuilder) Build() *Notification {
	return b.notification
}

// JSON returns the notification as JSON bytes.
func (b *SubscriptionBuilder) JSON() ([]byte, error) {
	return json.Marshal(b.notification)
}

// Message creates a Message with the notification payload.
func (b *SubscriptionBuilder) Message() (*Message, error) {
	payload, err := b.JSON()
	if err != nil {
		return nil, err
	}
	return &Message{Payload: payload}, nil
}

// MessageWithOptions creates a Message with the notification payload and options.
func (b *SubscriptionBuilder) MessageWithOptions(ttl int, urgency Urgency, topic string) (*Message, error) {
	payload, err := b.JSON()
	if err != nil {
		return nil, err
	}
	return &Message{
		Payload: payload,
		TTL:     ttl,
		Urgency: urgency,
		Topic:   topic,
	}, nil
}
