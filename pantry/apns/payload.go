// apns/payload.go
package apns

import (
	"encoding/json"
)

// PushType represents the type of push notification.
type PushType string

const (
	// PushTypeAlert for notifications that display an alert.
	PushTypeAlert PushType = "alert"

	// PushTypeBackground for silent background notifications.
	PushTypeBackground PushType = "background"

	// PushTypeLocation for location updates.
	PushTypeLocation PushType = "location"

	// PushTypeVoIP for VoIP notifications.
	PushTypeVoIP PushType = "voip"

	// PushTypeComplication for watchOS complications.
	PushTypeComplication PushType = "complication"

	// PushTypeFileProvider for file provider notifications.
	PushTypeFileProvider PushType = "fileprovider"

	// PushTypeMDM for MDM notifications.
	PushTypeMDM PushType = "mdm"

	// PushTypeLiveActivity for live activity updates.
	PushTypeLiveActivity PushType = "liveactivity"

	// PushTypePushToTalk for push-to-talk notifications.
	PushTypePushToTalk PushType = "pushtotalk"
)

// Priority values for notifications.
const (
	// PriorityLow sends at a time that conserves power.
	PriorityLow = 5

	// PriorityHigh sends immediately.
	PriorityHigh = 10
)

// InterruptionLevel represents the interruption level for iOS 15+.
type InterruptionLevel string

const (
	InterruptionLevelPassive       InterruptionLevel = "passive"
	InterruptionLevelActive        InterruptionLevel = "active"
	InterruptionLevelTimeSensitive InterruptionLevel = "time-sensitive"
	InterruptionLevelCritical      InterruptionLevel = "critical"
)

// Notification represents an APNs notification.
type Notification struct {
	// DeviceToken is the device token to send to.
	DeviceToken string

	// Payload is the notification payload.
	// If set, this overrides all other payload fields.
	Payload *Payload

	// Headers

	// ApnsID is a UUID that identifies the notification.
	ApnsID string

	// CollapseID groups notifications that can be collapsed.
	CollapseID string

	// Expiration is the UNIX timestamp when the notification expires.
	// 0 means APNs attempts to deliver immediately and discards if it can't.
	Expiration int64

	// Priority is the notification priority (5 or 10).
	Priority int

	// Topic is the bundle ID of the app.
	Topic string

	// PushType is the type of push notification.
	PushType PushType
}

// MarshalPayload marshals the notification payload to JSON.
func (n *Notification) MarshalPayload() ([]byte, error) {
	if n.Payload != nil {
		return json.Marshal(n.Payload)
	}
	return nil, ErrPayloadRequired
}

// Payload represents the APNs notification payload.
type Payload struct {
	// Aps is the aps dictionary.
	Aps *Aps `json:"aps"`

	// Custom data fields
	custom map[string]any
}

// NewPayload creates a new payload.
func NewPayload() *Payload {
	return &Payload{
		Aps:    &Aps{},
		custom: make(map[string]any),
	}
}

// MarshalJSON implements json.Marshaler.
func (p *Payload) MarshalJSON() ([]byte, error) {
	// Build the complete payload with aps and custom fields
	data := make(map[string]any)
	data["aps"] = p.Aps

	for k, v := range p.custom {
		data[k] = v
	}

	return json.Marshal(data)
}

// Alert sets a simple alert message.
func (p *Payload) Alert(message string) *Payload {
	p.Aps.Alert = message
	return p
}

// AlertTitle sets the alert title and body.
func (p *Payload) AlertTitle(title, body string) *Payload {
	p.Aps.Alert = &Alert{
		Title: title,
		Body:  body,
	}
	return p
}

// AlertSubtitle sets the alert title, subtitle, and body.
func (p *Payload) AlertSubtitle(title, subtitle, body string) *Payload {
	p.Aps.Alert = &Alert{
		Title:    title,
		Subtitle: subtitle,
		Body:     body,
	}
	return p
}

// AlertLocalized sets a localized alert.
func (p *Payload) AlertLocalized(locKey string, locArgs ...string) *Payload {
	p.Aps.Alert = &Alert{
		LocKey:  locKey,
		LocArgs: locArgs,
	}
	return p
}

// Badge sets the badge count.
func (p *Payload) Badge(count int) *Payload {
	p.Aps.Badge = &count
	return p
}

// ZeroBadge sets the badge to 0.
func (p *Payload) ZeroBadge() *Payload {
	zero := 0
	p.Aps.Badge = &zero
	return p
}

// Sound sets the sound name.
func (p *Payload) Sound(name string) *Payload {
	p.Aps.Sound = name
	return p
}

// DefaultSound sets the default sound.
func (p *Payload) DefaultSound() *Payload {
	p.Aps.Sound = "default"
	return p
}

// CriticalSound sets a critical sound.
func (p *Payload) CriticalSound(name string, volume float64) *Payload {
	p.Aps.Sound = &CriticalSound{
		Critical: 1,
		Name:     name,
		Volume:   volume,
	}
	return p
}

// ContentAvailable sets content-available for silent notifications.
func (p *Payload) ContentAvailable() *Payload {
	p.Aps.ContentAvailable = 1
	return p
}

// MutableContent enables mutable-content for notification service extensions.
func (p *Payload) MutableContent() *Payload {
	p.Aps.MutableContent = 1
	return p
}

// Category sets the notification category.
func (p *Payload) Category(category string) *Payload {
	p.Aps.Category = category
	return p
}

// ThreadID sets the thread identifier for grouping.
func (p *Payload) ThreadID(threadID string) *Payload {
	p.Aps.ThreadID = threadID
	return p
}

// TargetContentID sets the target content ID.
func (p *Payload) TargetContentID(contentID string) *Payload {
	p.Aps.TargetContentID = contentID
	return p
}

// InterruptionLevel sets the interruption level (iOS 15+).
func (p *Payload) InterruptionLevel(level InterruptionLevel) *Payload {
	p.Aps.InterruptionLevel = string(level)
	return p
}

// RelevanceScore sets the relevance score (0.0 to 1.0) for notification summary.
func (p *Payload) RelevanceScore(score float64) *Payload {
	p.Aps.RelevanceScore = &score
	return p
}

// FilterCriteria sets the filter criteria for Focus modes.
func (p *Payload) FilterCriteria(criteria string) *Payload {
	p.Aps.FilterCriteria = criteria
	return p
}

// StaleDate sets when the notification becomes stale (iOS 16+).
func (p *Payload) StaleDate(timestamp int64) *Payload {
	p.Aps.StaleDate = &timestamp
	return p
}

// ContentState sets the content state for live activities.
func (p *Payload) ContentState(state map[string]any) *Payload {
	p.Aps.ContentState = state
	return p
}

// DismissalDate sets the dismissal date for live activities.
func (p *Payload) DismissalDate(timestamp int64) *Payload {
	p.Aps.DismissalDate = &timestamp
	return p
}

// Event sets the event type for live activities.
func (p *Payload) Event(event string) *Payload {
	p.Aps.Event = event
	return p
}

// Custom adds a custom field to the payload.
func (p *Payload) Custom(key string, value any) *Payload {
	p.custom[key] = value
	return p
}

// Aps is the aps dictionary in the payload.
type Aps struct {
	// Alert is the alert message or dictionary.
	Alert any `json:"alert,omitempty"`

	// Badge is the badge count.
	Badge *int `json:"badge,omitempty"`

	// Sound is the sound name or critical sound dictionary.
	Sound any `json:"sound,omitempty"`

	// ContentAvailable enables silent notifications.
	ContentAvailable int `json:"content-available,omitempty"`

	// MutableContent enables notification service extensions.
	MutableContent int `json:"mutable-content,omitempty"`

	// Category is the notification category.
	Category string `json:"category,omitempty"`

	// ThreadID groups notifications.
	ThreadID string `json:"thread-id,omitempty"`

	// TargetContentID identifies content to show.
	TargetContentID string `json:"target-content-id,omitempty"`

	// InterruptionLevel is the interruption level (iOS 15+).
	InterruptionLevel string `json:"interruption-level,omitempty"`

	// RelevanceScore is the relevance score (0.0-1.0).
	RelevanceScore *float64 `json:"relevance-score,omitempty"`

	// FilterCriteria for Focus modes.
	FilterCriteria string `json:"filter-criteria,omitempty"`

	// StaleDate when notification becomes stale.
	StaleDate *int64 `json:"stale-date,omitempty"`

	// ContentState for live activities.
	ContentState map[string]any `json:"content-state,omitempty"`

	// DismissalDate for live activities.
	DismissalDate *int64 `json:"dismissal-date,omitempty"`

	// Event type for live activities.
	Event string `json:"event,omitempty"`

	// Timestamp for live activities.
	Timestamp int64 `json:"timestamp,omitempty"`
}

// Alert represents the alert dictionary.
type Alert struct {
	// Title is the alert title.
	Title string `json:"title,omitempty"`

	// Subtitle is the alert subtitle.
	Subtitle string `json:"subtitle,omitempty"`

	// Body is the alert body.
	Body string `json:"body,omitempty"`

	// LaunchImage is the launch image filename.
	LaunchImage string `json:"launch-image,omitempty"`

	// TitleLocKey is the localization key for the title.
	TitleLocKey string `json:"title-loc-key,omitempty"`

	// TitleLocArgs are the localization arguments for the title.
	TitleLocArgs []string `json:"title-loc-args,omitempty"`

	// SubtitleLocKey is the localization key for the subtitle.
	SubtitleLocKey string `json:"subtitle-loc-key,omitempty"`

	// SubtitleLocArgs are the localization arguments for the subtitle.
	SubtitleLocArgs []string `json:"subtitle-loc-args,omitempty"`

	// LocKey is the localization key for the body.
	LocKey string `json:"loc-key,omitempty"`

	// LocArgs are the localization arguments for the body.
	LocArgs []string `json:"loc-args,omitempty"`

	// SummaryArg is the summary argument.
	SummaryArg string `json:"summary-arg,omitempty"`

	// SummaryArgCount is the summary argument count.
	SummaryArgCount int `json:"summary-arg-count,omitempty"`
}

// CriticalSound represents a critical alert sound.
type CriticalSound struct {
	// Critical is 1 for critical alerts.
	Critical int `json:"critical"`

	// Name is the sound filename.
	Name string `json:"name"`

	// Volume is the volume (0.0 to 1.0).
	Volume float64 `json:"volume"`
}

// NotificationBuilder provides a fluent API for building notifications.
type NotificationBuilder struct {
	notification *Notification
	payload      *Payload
}

// NewNotification creates a new notification builder.
func NewNotification(deviceToken string) *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			DeviceToken: deviceToken,
			PushType:    PushTypeAlert,
			Priority:    PriorityHigh,
		},
		payload: NewPayload(),
	}
}

// Alert sets a simple alert message.
func (b *NotificationBuilder) Alert(message string) *NotificationBuilder {
	b.payload.Alert(message)
	return b
}

// AlertTitle sets the alert title and body.
func (b *NotificationBuilder) AlertTitle(title, body string) *NotificationBuilder {
	b.payload.AlertTitle(title, body)
	return b
}

// AlertSubtitle sets the alert title, subtitle, and body.
func (b *NotificationBuilder) AlertSubtitle(title, subtitle, body string) *NotificationBuilder {
	b.payload.AlertSubtitle(title, subtitle, body)
	return b
}

// Badge sets the badge count.
func (b *NotificationBuilder) Badge(count int) *NotificationBuilder {
	b.payload.Badge(count)
	return b
}

// Sound sets the sound name.
func (b *NotificationBuilder) Sound(name string) *NotificationBuilder {
	b.payload.Sound(name)
	return b
}

// DefaultSound sets the default sound.
func (b *NotificationBuilder) DefaultSound() *NotificationBuilder {
	b.payload.DefaultSound()
	return b
}

// Category sets the notification category.
func (b *NotificationBuilder) Category(category string) *NotificationBuilder {
	b.payload.Category(category)
	return b
}

// ThreadID sets the thread identifier.
func (b *NotificationBuilder) ThreadID(threadID string) *NotificationBuilder {
	b.payload.ThreadID(threadID)
	return b
}

// ContentAvailable makes this a silent notification.
func (b *NotificationBuilder) ContentAvailable() *NotificationBuilder {
	b.payload.ContentAvailable()
	b.notification.PushType = PushTypeBackground
	b.notification.Priority = PriorityLow
	return b
}

// MutableContent enables notification service extensions.
func (b *NotificationBuilder) MutableContent() *NotificationBuilder {
	b.payload.MutableContent()
	return b
}

// CollapseID sets the collapse ID.
func (b *NotificationBuilder) CollapseID(collapseID string) *NotificationBuilder {
	b.notification.CollapseID = collapseID
	return b
}

// Topic sets the APNs topic (bundle ID).
func (b *NotificationBuilder) Topic(topic string) *NotificationBuilder {
	b.notification.Topic = topic
	return b
}

// Expiration sets when the notification expires.
func (b *NotificationBuilder) Expiration(timestamp int64) *NotificationBuilder {
	b.notification.Expiration = timestamp
	return b
}

// Priority sets the priority.
func (b *NotificationBuilder) Priority(priority int) *NotificationBuilder {
	b.notification.Priority = priority
	return b
}

// HighPriority sets high priority.
func (b *NotificationBuilder) HighPriority() *NotificationBuilder {
	b.notification.Priority = PriorityHigh
	return b
}

// LowPriority sets low priority.
func (b *NotificationBuilder) LowPriority() *NotificationBuilder {
	b.notification.Priority = PriorityLow
	return b
}

// PushType sets the push type.
func (b *NotificationBuilder) PushType(pushType PushType) *NotificationBuilder {
	b.notification.PushType = pushType
	return b
}

// ApnsID sets the APNs ID.
func (b *NotificationBuilder) ApnsID(id string) *NotificationBuilder {
	b.notification.ApnsID = id
	return b
}

// Custom adds a custom payload field.
func (b *NotificationBuilder) Custom(key string, value any) *NotificationBuilder {
	b.payload.Custom(key, value)
	return b
}

// InterruptionLevel sets the interruption level (iOS 15+).
func (b *NotificationBuilder) InterruptionLevel(level InterruptionLevel) *NotificationBuilder {
	b.payload.InterruptionLevel(level)
	return b
}

// TimeSensitive sets the interruption level to time-sensitive.
func (b *NotificationBuilder) TimeSensitive() *NotificationBuilder {
	b.payload.InterruptionLevel(InterruptionLevelTimeSensitive)
	return b
}

// Build returns the notification.
func (b *NotificationBuilder) Build() *Notification {
	b.notification.Payload = b.payload
	return b.notification
}

// BackgroundNotification creates a silent background notification.
func BackgroundNotification(deviceToken string) *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			DeviceToken: deviceToken,
			PushType:    PushTypeBackground,
			Priority:    PriorityLow,
		},
		payload: NewPayload().ContentAvailable(),
	}
}

// VoIPNotification creates a VoIP notification.
func VoIPNotification(deviceToken string) *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			DeviceToken: deviceToken,
			PushType:    PushTypeVoIP,
			Priority:    PriorityHigh,
		},
		payload: NewPayload(),
	}
}

// LiveActivityNotification creates a live activity notification.
func LiveActivityNotification(deviceToken string, event string, contentState map[string]any) *NotificationBuilder {
	payload := NewPayload()
	payload.Event(event)
	payload.ContentState(contentState)
	payload.Aps.Timestamp = 0 // Will be set to current time

	return &NotificationBuilder{
		notification: &Notification{
			DeviceToken: deviceToken,
			PushType:    PushTypeLiveActivity,
			Priority:    PriorityHigh,
		},
		payload: payload,
	}
}
