// fcm/message.go
package fcm

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
)

// Message represents an FCM message.
type Message struct {
	// Target (exactly one required)
	Token     string `json:"token,omitempty"`     // Device registration token
	Topic     string `json:"topic,omitempty"`     // Topic name (without /topics/ prefix)
	Condition string `json:"condition,omitempty"` // Condition for topic targeting

	// Notification payload (optional)
	Notification *Notification `json:"notification,omitempty"`

	// Data payload (optional)
	Data map[string]string `json:"data,omitempty"`

	// Platform-specific options
	Android    *AndroidConfig    `json:"android,omitempty"`
	Webpush    *WebpushConfig    `json:"webpush,omitempty"`
	APNS       *APNSConfig       `json:"apns,omitempty"`
	FCMOptions *FCMOptions       `json:"fcm_options,omitempty"`
}

// Notification is the basic notification payload.
type Notification struct {
	Title    string `json:"title,omitempty"`
	Body     string `json:"body,omitempty"`
	ImageURL string `json:"image,omitempty"`
}

// AndroidConfig contains Android-specific options.
type AndroidConfig struct {
	CollapseKey           string               `json:"collapse_key,omitempty"`
	Priority              AndroidPriority      `json:"priority,omitempty"`
	TTL                   string               `json:"ttl,omitempty"` // Duration string like "3.5s"
	RestrictedPackageName string               `json:"restricted_package_name,omitempty"`
	Data                  map[string]string    `json:"data,omitempty"`
	Notification          *AndroidNotification `json:"notification,omitempty"`
	FCMOptions            *AndroidFCMOptions   `json:"fcm_options,omitempty"`
	DirectBootOK          bool                 `json:"direct_boot_ok,omitempty"`
}

// AndroidPriority is the priority for Android messages.
type AndroidPriority string

const (
	AndroidPriorityNormal AndroidPriority = "normal"
	AndroidPriorityHigh   AndroidPriority = "high"
)

// AndroidNotification contains Android-specific notification options.
type AndroidNotification struct {
	Title                 string                  `json:"title,omitempty"`
	Body                  string                  `json:"body,omitempty"`
	Icon                  string                  `json:"icon,omitempty"`
	Color                 string                  `json:"color,omitempty"` // #RRGGBB format
	Sound                 string                  `json:"sound,omitempty"`
	Tag                   string                  `json:"tag,omitempty"`
	ClickAction           string                  `json:"click_action,omitempty"`
	BodyLocKey            string                  `json:"body_loc_key,omitempty"`
	BodyLocArgs           []string                `json:"body_loc_args,omitempty"`
	TitleLocKey           string                  `json:"title_loc_key,omitempty"`
	TitleLocArgs          []string                `json:"title_loc_args,omitempty"`
	ChannelID             string                  `json:"channel_id,omitempty"`
	Ticker                string                  `json:"ticker,omitempty"`
	Sticky                bool                    `json:"sticky,omitempty"`
	EventTime             string                  `json:"event_time,omitempty"` // RFC3339 timestamp
	LocalOnly             bool                    `json:"local_only,omitempty"`
	NotificationPriority  AndroidNotificationPrio `json:"notification_priority,omitempty"`
	DefaultSound          bool                    `json:"default_sound,omitempty"`
	DefaultVibrateTimings bool                    `json:"default_vibrate_timings,omitempty"`
	DefaultLightSettings  bool                    `json:"default_light_settings,omitempty"`
	VibrateTimings        []string                `json:"vibrate_timings,omitempty"` // Duration strings
	Visibility            AndroidVisibility       `json:"visibility,omitempty"`
	NotificationCount     int                     `json:"notification_count,omitempty"`
	LightSettings         *LightSettings          `json:"light_settings,omitempty"`
	Image                 string                  `json:"image,omitempty"`
}

// AndroidNotificationPrio is the notification priority for Android.
type AndroidNotificationPrio string

const (
	AndroidNotificationPrioUnspecified AndroidNotificationPrio = "PRIORITY_UNSPECIFIED"
	AndroidNotificationPrioMin         AndroidNotificationPrio = "PRIORITY_MIN"
	AndroidNotificationPrioLow         AndroidNotificationPrio = "PRIORITY_LOW"
	AndroidNotificationPrioDefault     AndroidNotificationPrio = "PRIORITY_DEFAULT"
	AndroidNotificationPrioHigh        AndroidNotificationPrio = "PRIORITY_HIGH"
	AndroidNotificationPrioMax         AndroidNotificationPrio = "PRIORITY_MAX"
)

// AndroidVisibility is the visibility setting for Android notifications.
type AndroidVisibility string

const (
	AndroidVisibilityUnspecified AndroidVisibility = "VISIBILITY_UNSPECIFIED"
	AndroidVisibilityPrivate     AndroidVisibility = "PRIVATE"
	AndroidVisibilityPublic      AndroidVisibility = "PUBLIC"
	AndroidVisibilitySecret      AndroidVisibility = "SECRET"
)

// LightSettings controls the notification LED.
type LightSettings struct {
	Color          *Color `json:"color,omitempty"`
	LightOnDuration  string `json:"light_on_duration,omitempty"`  // Duration string
	LightOffDuration string `json:"light_off_duration,omitempty"` // Duration string
}

// Color represents an RGBA color.
type Color struct {
	Red   float64 `json:"red,omitempty"`
	Green float64 `json:"green,omitempty"`
	Blue  float64 `json:"blue,omitempty"`
	Alpha float64 `json:"alpha,omitempty"`
}

// AndroidFCMOptions contains Android-specific FCM options.
type AndroidFCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// WebpushConfig contains Web Push-specific options.
type WebpushConfig struct {
	Headers      map[string]string    `json:"headers,omitempty"`
	Data         map[string]string    `json:"data,omitempty"`
	Notification *WebpushNotification `json:"notification,omitempty"`
	FCMOptions   *WebpushFCMOptions   `json:"fcm_options,omitempty"`
}

// WebpushNotification contains Web Push notification options.
type WebpushNotification struct {
	Title              string   `json:"title,omitempty"`
	Body               string   `json:"body,omitempty"`
	Icon               string   `json:"icon,omitempty"`
	Badge              string   `json:"badge,omitempty"`
	Image              string   `json:"image,omitempty"`
	Language           string   `json:"lang,omitempty"`
	Tag                string   `json:"tag,omitempty"`
	Direction          string   `json:"dir,omitempty"` // auto, ltr, rtl
	Renotify           bool     `json:"renotify,omitempty"`
	Interaction        bool     `json:"requireInteraction,omitempty"`
	Silent             bool     `json:"silent,omitempty"`
	Timestamp          int64    `json:"timestamp,omitempty"` // Unix timestamp in ms
	Vibrate            []int    `json:"vibrate,omitempty"`
	Actions            []Action `json:"actions,omitempty"`
	Data               any      `json:"data,omitempty"`
}

// Action is an action button for web push notifications.
type Action struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// WebpushFCMOptions contains Web Push-specific FCM options.
type WebpushFCMOptions struct {
	Link           string `json:"link,omitempty"`
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// APNSConfig contains Apple Push Notification Service options.
type APNSConfig struct {
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    *APNSPayload      `json:"payload,omitempty"`
	FCMOptions *APNSFCMOptions   `json:"fcm_options,omitempty"`
}

// APNSPayload is the APNS payload.
type APNSPayload struct {
	Aps *Aps `json:"aps,omitempty"`
	// Custom keys can be added as needed
}

// Aps is the aps dictionary in the APNS payload.
type Aps struct {
	Alert            any    `json:"alert,omitempty"` // String or ApsAlert
	Badge            *int   `json:"badge,omitempty"`
	Sound            any    `json:"sound,omitempty"` // String or CriticalSound
	ContentAvailable int    `json:"content-available,omitempty"`
	MutableContent   int    `json:"mutable-content,omitempty"`
	Category         string `json:"category,omitempty"`
	ThreadID         string `json:"thread-id,omitempty"`
	TargetContentID  string `json:"target-content-id,omitempty"`
}

// ApsAlert is the alert dictionary for APNS.
type ApsAlert struct {
	Title           string   `json:"title,omitempty"`
	Subtitle        string   `json:"subtitle,omitempty"`
	Body            string   `json:"body,omitempty"`
	LaunchImage     string   `json:"launch-image,omitempty"`
	TitleLocKey     string   `json:"title-loc-key,omitempty"`
	TitleLocArgs    []string `json:"title-loc-args,omitempty"`
	SubtitleLocKey  string   `json:"subtitle-loc-key,omitempty"`
	SubtitleLocArgs []string `json:"subtitle-loc-args,omitempty"`
	LocKey          string   `json:"loc-key,omitempty"`
	LocArgs         []string `json:"loc-args,omitempty"`
}

// CriticalSound is for critical alerts on iOS.
type CriticalSound struct {
	Critical int     `json:"critical,omitempty"`
	Name     string  `json:"name,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
}

// APNSFCMOptions contains APNS-specific FCM options.
type APNSFCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
	Image          string `json:"image,omitempty"`
}

// FCMOptions contains FCM-specific options.
type FCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// MulticastMessage is a message sent to multiple tokens.
type MulticastMessage struct {
	Tokens       []string          `json:"-"` // Up to 500 tokens
	Notification *Notification     `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *AndroidConfig    `json:"android,omitempty"`
	Webpush      *WebpushConfig    `json:"webpush,omitempty"`
	APNS         *APNSConfig       `json:"apns,omitempty"`
	FCMOptions   *FCMOptions       `json:"fcm_options,omitempty"`
}

// MessageBuilder provides a fluent API for building messages.
type MessageBuilder struct {
	message *Message
}

// NewMessage creates a new message builder.
func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		message: &Message{},
	}
}

// ToToken sets the target device token.
func (b *MessageBuilder) ToToken(token string) *MessageBuilder {
	b.message.Token = token
	b.message.Topic = ""
	b.message.Condition = ""
	return b
}

// ToTopic sets the target topic.
func (b *MessageBuilder) ToTopic(topic string) *MessageBuilder {
	b.message.Topic = topic
	b.message.Token = ""
	b.message.Condition = ""
	return b
}

// ToCondition sets the target condition.
func (b *MessageBuilder) ToCondition(condition string) *MessageBuilder {
	b.message.Condition = condition
	b.message.Token = ""
	b.message.Topic = ""
	return b
}

// WithNotification sets the notification payload.
func (b *MessageBuilder) WithNotification(title, body string) *MessageBuilder {
	b.message.Notification = &Notification{
		Title: title,
		Body:  body,
	}
	return b
}

// WithNotificationImage sets the notification with an image.
func (b *MessageBuilder) WithNotificationImage(title, body, imageURL string) *MessageBuilder {
	b.message.Notification = &Notification{
		Title:    title,
		Body:     body,
		ImageURL: imageURL,
	}
	return b
}

// WithData sets the data payload.
func (b *MessageBuilder) WithData(data map[string]string) *MessageBuilder {
	b.message.Data = data
	return b
}

// AddData adds a key-value pair to the data payload.
func (b *MessageBuilder) AddData(key, value string) *MessageBuilder {
	if b.message.Data == nil {
		b.message.Data = make(map[string]string)
	}
	b.message.Data[key] = value
	return b
}

// WithAndroid sets Android-specific options.
func (b *MessageBuilder) WithAndroid(cfg *AndroidConfig) *MessageBuilder {
	b.message.Android = cfg
	return b
}

// WithAndroidHighPriority sets high priority for Android.
func (b *MessageBuilder) WithAndroidHighPriority() *MessageBuilder {
	if b.message.Android == nil {
		b.message.Android = &AndroidConfig{}
	}
	b.message.Android.Priority = AndroidPriorityHigh
	return b
}

// WithAndroidTTL sets the time-to-live for Android.
func (b *MessageBuilder) WithAndroidTTL(ttl string) *MessageBuilder {
	if b.message.Android == nil {
		b.message.Android = &AndroidConfig{}
	}
	b.message.Android.TTL = ttl
	return b
}

// WithAndroidCollapseKey sets the collapse key for Android.
func (b *MessageBuilder) WithAndroidCollapseKey(key string) *MessageBuilder {
	if b.message.Android == nil {
		b.message.Android = &AndroidConfig{}
	}
	b.message.Android.CollapseKey = key
	return b
}

// WithWebpush sets Web Push-specific options.
func (b *MessageBuilder) WithWebpush(cfg *WebpushConfig) *MessageBuilder {
	b.message.Webpush = cfg
	return b
}

// WithWebpushLink sets the click link for web push.
func (b *MessageBuilder) WithWebpushLink(link string) *MessageBuilder {
	if b.message.Webpush == nil {
		b.message.Webpush = &WebpushConfig{}
	}
	if b.message.Webpush.FCMOptions == nil {
		b.message.Webpush.FCMOptions = &WebpushFCMOptions{}
	}
	b.message.Webpush.FCMOptions.Link = link
	return b
}

// WithWebpushIcon sets the icon for web push notifications.
func (b *MessageBuilder) WithWebpushIcon(icon string) *MessageBuilder {
	if b.message.Webpush == nil {
		b.message.Webpush = &WebpushConfig{}
	}
	if b.message.Webpush.Notification == nil {
		b.message.Webpush.Notification = &WebpushNotification{}
	}
	b.message.Webpush.Notification.Icon = icon
	return b
}

// WithWebpushBadge sets the badge for web push notifications.
func (b *MessageBuilder) WithWebpushBadge(badge string) *MessageBuilder {
	if b.message.Webpush == nil {
		b.message.Webpush = &WebpushConfig{}
	}
	if b.message.Webpush.Notification == nil {
		b.message.Webpush.Notification = &WebpushNotification{}
	}
	b.message.Webpush.Notification.Badge = badge
	return b
}

// WithWebpushActions adds action buttons to web push notifications.
func (b *MessageBuilder) WithWebpushActions(actions []Action) *MessageBuilder {
	if b.message.Webpush == nil {
		b.message.Webpush = &WebpushConfig{}
	}
	if b.message.Webpush.Notification == nil {
		b.message.Webpush.Notification = &WebpushNotification{}
	}
	b.message.Webpush.Notification.Actions = actions
	return b
}

// WithAPNS sets Apple Push Notification Service options.
func (b *MessageBuilder) WithAPNS(cfg *APNSConfig) *MessageBuilder {
	b.message.APNS = cfg
	return b
}

// WithFCMOptions sets FCM-specific options.
func (b *MessageBuilder) WithFCMOptions(opts *FCMOptions) *MessageBuilder {
	b.message.FCMOptions = opts
	return b
}

// WithAnalyticsLabel sets the analytics label.
func (b *MessageBuilder) WithAnalyticsLabel(label string) *MessageBuilder {
	if b.message.FCMOptions == nil {
		b.message.FCMOptions = &FCMOptions{}
	}
	b.message.FCMOptions.AnalyticsLabel = label
	return b
}

// Build returns the constructed message.
func (b *MessageBuilder) Build() *Message {
	return b.message
}

// Helper functions for JWT and crypto operations.

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func signRS256(data []byte, key *rsa.PrivateKey) ([]byte, error) {
	hash := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
}

