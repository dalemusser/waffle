# APNs - Apple Push Notification Service

The `apns` package provides a complete Apple Push Notification Service client using the HTTP/2-based APNs Provider API with both token-based (JWT) and certificate-based authentication.

## Features

- **HTTP/2 Connection**: Uses HTTP/2 for efficient multiplexed connections
- **Token-Based Auth**: JWT authentication using .p8 auth keys (recommended)
- **Certificate Auth**: Legacy .p12 certificate authentication
- **Automatic Token Refresh**: JWT tokens are refreshed before expiration
- **All Push Types**: Alert, background, VoIP, live activities, and more
- **Notification Builder**: Fluent API for building notifications
- **iOS 15+ Features**: Interruption levels, Focus modes, relevance scores
- **Live Activities**: Support for live activity updates
- **Batch Sending**: Send to multiple devices efficiently
- **Comprehensive Errors**: Typed errors for all APNs error codes

## Installation

```go
import "waffle/apns"
```

## Quick Start

### Token-Based Authentication (Recommended)

```go
// Create client with .p8 auth key
client, err := apns.NewClient(apns.Config{
    AuthKeyFile: "/path/to/AuthKey_XXXXX.p8",
    KeyID:       "XXXXX",      // Key ID from Apple Developer
    TeamID:      "XXXXXXXXXX", // Team ID from Apple Developer
    Endpoint:    apns.ProductionEndpoint,
})
if err != nil {
    log.Fatal(err)
}

// Send notification
notification := apns.NewNotification("device-token-here").
    AlertTitle("Hello", "You have a new message").
    DefaultSound().
    Badge(1).
    Topic("com.example.app").
    Build()

resp, err := client.Send(ctx, notification)
if err != nil {
    if apns.IsUnregistered(err) {
        // Remove invalid device token
        removeToken(notification.DeviceToken)
    }
    log.Printf("Failed: %v", err)
    return
}

log.Printf("Sent! APNs ID: %s", resp.ApnsID)
```

### Certificate-Based Authentication (Legacy)

```go
client, err := apns.NewClient(apns.Config{
    CertificateFile:     "/path/to/certificate.p12",
    CertificatePassword: "password",
    Endpoint:            apns.ProductionEndpoint,
})
```

### Development vs Production

```go
// Development/sandbox environment
devClient, err := apns.Development(apns.Config{
    AuthKeyFile: "AuthKey.p8",
    KeyID:       "XXXXX",
    TeamID:      "XXXXXXXXXX",
})

// Production environment
prodClient, err := apns.Production(apns.Config{
    AuthKeyFile: "AuthKey.p8",
    KeyID:       "XXXXX",
    TeamID:      "XXXXXXXXXX",
})
```

## Notification Types

### Alert Notification

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("Title", "Body text").
    DefaultSound().
    Badge(1).
    Topic("com.example.app").
    Build()
```

### Rich Alert with Subtitle

```go
notification := apns.NewNotification(deviceToken).
    AlertSubtitle("Title", "Subtitle", "Body text").
    Sound("notification.wav").
    Badge(5).
    Category("MESSAGE").
    ThreadID("chat-123").
    Topic("com.example.app").
    Build()
```

### Silent Background Notification

```go
notification := apns.BackgroundNotification(deviceToken).
    Custom("type", "sync").
    Custom("data", map[string]any{"key": "value"}).
    Topic("com.example.app").
    Build()
```

### VoIP Notification

```go
notification := apns.VoIPNotification(deviceToken).
    Custom("caller", "John Doe").
    Custom("callId", "12345").
    Topic("com.example.app.voip").
    Build()
```

### Live Activity Update

```go
contentState := map[string]any{
    "score": "3-2",
    "period": "2nd",
}

notification := apns.LiveActivityNotification(deviceToken, "update", contentState).
    Topic("com.example.app.push-type.liveactivity").
    Build()
```

## Notification Builder

### Basic Options

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("Title", "Body").     // Title and body
    Badge(10).                        // Badge count
    Sound("default").                 // Sound name
    DefaultSound().                   // Use default sound
    Category("CATEGORY_ID").          // Action category
    ThreadID("thread-123").           // Thread for grouping
    Topic("com.example.app").         // Bundle ID
    Build()
```

### Priority and Expiration

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("Urgent", "Action required").
    HighPriority().                           // Priority 10
    Expiration(time.Now().Add(time.Hour).Unix()). // Expires in 1 hour
    Build()

// Low priority for non-urgent
notification := apns.NewNotification(deviceToken).
    AlertTitle("FYI", "Something happened").
    LowPriority().                            // Priority 5
    Build()
```

### Collapse ID

```go
// New notifications with same collapse ID replace previous ones
notification := apns.NewNotification(deviceToken).
    AlertTitle("Score Update", "Team A: 3 - Team B: 2").
    CollapseID("score-update").
    Build()
```

### iOS 15+ Features

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("Breaking News", "Important update").
    TimeSensitive().                 // Time-sensitive interruption
    Build()

// Or specify interruption level explicitly
notification := apns.NewNotification(deviceToken).
    AlertTitle("Reminder", "Check your calendar").
    InterruptionLevel(apns.InterruptionLevelActive).
    Build()
```

Interruption levels:
- `InterruptionLevelPassive` - Silently added to notification list
- `InterruptionLevelActive` - Default, lights up screen
- `InterruptionLevelTimeSensitive` - Breaks through Focus modes
- `InterruptionLevelCritical` - Plays sound even if silenced (requires entitlement)

### Custom Payload Data

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("New Message", "You have a message").
    Custom("messageId", "msg-123").
    Custom("senderId", "user-456").
    Custom("data", map[string]any{
        "type": "chat",
        "roomId": "room-789",
    }).
    Build()
```

### Mutable Content (Notification Service Extension)

```go
notification := apns.NewNotification(deviceToken).
    AlertTitle("Photo", "New photo from John").
    MutableContent().  // Enable modification by extension
    Custom("imageUrl", "https://example.com/photo.jpg").
    Build()
```

## Payload Builder

For more control over the payload:

```go
payload := apns.NewPayload().
    AlertTitle("Title", "Body").
    Badge(5).
    Sound("chime.wav").
    Category("ACTIONS").
    ThreadID("thread-1").
    InterruptionLevel(apns.InterruptionLevelTimeSensitive).
    Custom("key1", "value1").
    Custom("key2", 123)

notification := &apns.Notification{
    DeviceToken: deviceToken,
    Topic:       "com.example.app",
    Payload:     payload,
    Priority:    apns.PriorityHigh,
    PushType:    apns.PushTypeAlert,
}
```

### Localized Alerts

```go
payload := apns.NewPayload().
    AlertLocalized("NEW_MESSAGE_KEY", "John", "Hello!").
    DefaultSound()
```

### Critical Alerts

```go
payload := apns.NewPayload().
    AlertTitle("Emergency", "Critical alert!").
    CriticalSound("alarm.wav", 1.0). // Max volume
    InterruptionLevel(apns.InterruptionLevelCritical)
```

## Batch Sending

```go
notifications := []*apns.Notification{
    apns.NewNotification(token1).AlertTitle("Hi", "Message 1").Build(),
    apns.NewNotification(token2).AlertTitle("Hi", "Message 2").Build(),
    apns.NewNotification(token3).AlertTitle("Hi", "Message 3").Build(),
}

result, err := client.SendMultiple(ctx, notifications)
if err != nil {
    log.Printf("Batch failed: %v", err)
    return
}

log.Printf("Success: %d, Failed: %d", result.SuccessCount, result.FailureCount)

// Clean up invalid tokens
for _, token := range result.InvalidTokens() {
    removeToken(token)
}
```

## Error Handling

```go
resp, err := client.Send(ctx, notification)
if err != nil {
    // Check for invalid device token
    if apns.IsBadToken(err) {
        removeToken(notification.DeviceToken)
        return
    }

    // Check for unregistered device
    if apns.IsUnregistered(err) {
        removeToken(notification.DeviceToken)
        return
    }

    // Check for rate limiting
    if apns.IsRateLimited(err) {
        time.Sleep(time.Second)
        return retryLater()
    }

    // Check if retryable
    if apns.IsRetryable(err) {
        return retryWithBackoff()
    }

    // Check for auth errors
    if apns.IsAuthError(err) {
        log.Printf("Authentication error: %v", err)
        return
    }

    // Get detailed error
    var apnsErr *apns.Error
    if errors.As(err, &apnsErr) {
        log.Printf("APNs error: %s (status: %d)", apnsErr.Reason, apnsErr.StatusCode)

        // For unregistered, get the timestamp when it became invalid
        if apnsErr.Reason == "Unregistered" && apnsErr.Timestamp > 0 {
            log.Printf("Token invalid since: %v", time.Unix(apnsErr.Timestamp/1000, 0))
        }
    }

    return
}

log.Printf("Success! APNs ID: %s", resp.ApnsID)
```

## Push Types

| Push Type | Description | Priority |
|-----------|-------------|----------|
| `PushTypeAlert` | Displays notification | 10 (high) |
| `PushTypeBackground` | Silent background update | 5 (low) |
| `PushTypeVoIP` | VoIP call notification | 10 (high) |
| `PushTypeLiveActivity` | Live activity update | 10 (high) |
| `PushTypeLocation` | Location update | 5 (low) |
| `PushTypeComplication` | watchOS complication | 5 (low) |
| `PushTypeFileProvider` | File provider update | 5 (low) |
| `PushTypeMDM` | MDM command | 10 (high) |
| `PushTypePushToTalk` | Push-to-talk | 10 (high) |

## Topics

The topic should match your app's bundle ID:

```go
// Regular notifications
notification.Topic = "com.example.app"

// VoIP notifications
notification.Topic = "com.example.app.voip"

// Complications
notification.Topic = "com.example.app.complication"

// Live activities
notification.Topic = "com.example.app.push-type.liveactivity"
```

## Global Client

```go
// Set default client at startup
apns.SetDefaultClient(client)

// Use anywhere
resp, err := apns.Send(ctx, notification)
```

## Authentication Setup

### Token-Based (.p8 Key)

1. Go to [Apple Developer](https://developer.apple.com/account)
2. Navigate to Certificates, Identifiers & Profiles > Keys
3. Create a new key with Apple Push Notifications enabled
4. Download the .p8 file (only available once!)
5. Note the Key ID shown on the page
6. Find your Team ID in Membership details

```go
client, err := apns.NewClient(apns.Config{
    AuthKeyFile: "AuthKey_XXXXXXXX.p8",
    KeyID:       "XXXXXXXX",   // From the key page
    TeamID:      "XXXXXXXXXX", // From membership
})
```

### Certificate-Based (.p12)

1. Go to Apple Developer
2. Create an APNs certificate for your app
3. Export as .p12 from Keychain Access
4. Use the certificate:

```go
client, err := apns.NewClient(apns.Config{
    CertificateFile:     "apns-cert.p12",
    CertificatePassword: "password",
})
```

## Best Practices

1. **Use Token Auth**: .p8 keys are simpler and don't expire
2. **Handle Token Errors**: Remove tokens that return BadDeviceToken or Unregistered
3. **Set Appropriate Priority**: Use low priority for non-urgent notifications
4. **Use Collapse IDs**: For notifications that update (scores, status)
5. **Set Expiration**: For time-sensitive notifications
6. **Use Topics**: Always set the correct topic for your push type
7. **Handle Rate Limits**: Implement backoff when receiving 429 errors

## Payload Size Limits

- Maximum payload size: 4KB (4096 bytes)
- VoIP notifications: 5KB (5120 bytes)

## Response Codes

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 400 | Bad request (check payload) |
| 403 | Authentication error |
| 404 | Bad path |
| 405 | Method not allowed |
| 410 | Device token expired |
| 413 | Payload too large |
| 429 | Too many requests |
| 500 | Internal server error |
| 503 | Service unavailable |
