# FCM - Firebase Cloud Messaging

The `fcm` package provides a complete Firebase Cloud Messaging (FCM) HTTP v1 API client for sending push notifications to Android, iOS, and web applications.

## Features

- **HTTP v1 API**: Uses the modern FCM HTTP v1 API with OAuth2 authentication
- **Service Account Authentication**: Automatic JWT-based authentication using service account credentials
- **All Platforms**: Support for Android, iOS (APNS), and Web Push configurations
- **Multicast**: Send to up to 500 devices in a single request
- **Topics**: Subscribe/unsubscribe devices to topics, send to topics
- **Conditions**: Target multiple topics with boolean logic
- **Message Builder**: Fluent API for constructing messages
- **Automatic Token Refresh**: OAuth2 access tokens are refreshed automatically
- **Comprehensive Error Handling**: Typed errors with retry support

## Installation

```go
import "waffle/fcm"
```

## Quick Start

### Initialize Client

```go
// From service account JSON file
client, err := fcm.NewClient(fcm.Options{
    CredentialsFile: "/path/to/service-account.json",
})
if err != nil {
    log.Fatal(err)
}

// Or from JSON content
client, err := fcm.NewClient(fcm.Options{
    CredentialsJSON: []byte(`{...}`),
})

// Or with explicit credentials
client, err := fcm.NewClient(fcm.Options{
    ProjectID:    "my-project",
    ClientEmail:  "firebase@my-project.iam.gserviceaccount.com",
    PrivateKey:   privateKeyPEM,
})
```

### Send a Notification

```go
// Simple notification
msg := fcm.NewMessage().
    ToToken("device-token-here").
    WithNotification("Hello!", "This is a push notification").
    Build()

resp, err := client.Send(ctx, msg)
if err != nil {
    log.Printf("Failed to send: %v", err)
    return
}
log.Printf("Message sent: %s", resp.MessageID)
```

### Send Data Message

```go
msg := fcm.NewMessage().
    ToToken("device-token").
    AddData("type", "new_message").
    AddData("sender", "user123").
    AddData("content", "Hello!").
    Build()

resp, err := client.Send(ctx, msg)
```

### Send to Topic

```go
msg := fcm.NewMessage().
    ToTopic("news").
    WithNotification("Breaking News", "Something important happened").
    Build()

resp, err := client.Send(ctx, msg)
```

### Send with Condition

```go
// Send to users subscribed to 'news' AND 'sports'
msg := fcm.NewMessage().
    ToCondition("'news' in topics && 'sports' in topics").
    WithNotification("Sports News", "Big game tonight!").
    Build()

resp, err := client.Send(ctx, msg)
```

## Platform-Specific Configuration

### Android

```go
msg := fcm.NewMessage().
    ToToken("device-token").
    WithNotification("Title", "Body").
    WithAndroidHighPriority().
    WithAndroidTTL("3600s").
    WithAndroidCollapseKey("updates").
    WithAndroid(&fcm.AndroidConfig{
        Notification: &fcm.AndroidNotification{
            Icon:      "notification_icon",
            Color:     "#FF5722",
            ChannelID: "important_channel",
            Sound:     "default",
        },
    }).
    Build()
```

### Web Push

```go
msg := fcm.NewMessage().
    ToToken("device-token").
    WithNotification("Title", "Body").
    WithWebpushLink("https://example.com/page").
    WithWebpushIcon("https://example.com/icon.png").
    WithWebpushBadge("https://example.com/badge.png").
    WithWebpushActions([]fcm.Action{
        {Action: "open", Title: "Open", Icon: "open.png"},
        {Action: "dismiss", Title: "Dismiss"},
    }).
    Build()
```

### iOS (APNS)

```go
badge := 5
msg := &fcm.Message{
    Token: "device-token",
    Notification: &fcm.Notification{
        Title: "Hello",
        Body:  "Message body",
    },
    APNS: &fcm.APNSConfig{
        Headers: map[string]string{
            "apns-priority": "10",
        },
        Payload: &fcm.APNSPayload{
            Aps: &fcm.Aps{
                Badge:    &badge,
                Sound:    "default",
                Category: "MESSAGE",
            },
        },
    },
}
```

## Multicast (Multiple Devices)

```go
msg := &fcm.MulticastMessage{
    Tokens: []string{
        "token1", "token2", "token3", // Up to 500 tokens
    },
    Notification: &fcm.Notification{
        Title: "Announcement",
        Body:  "Message for everyone",
    },
}

resp, err := client.SendMulticast(ctx, msg)
if err != nil {
    // Check for partial failures
    var batchErr *fcm.BatchError
    if errors.As(err, &batchErr) {
        log.Printf("Sent %d, failed %d", batchErr.SuccessCount, batchErr.FailureCount)

        // Remove unregistered tokens
        for _, token := range batchErr.UnregisteredTokens() {
            removeTokenFromDB(token)
        }
    }
}

log.Printf("Success: %d, Failure: %d", resp.SuccessCount, resp.FailureCount)
```

## Topic Management

### Subscribe Devices to Topic

```go
tokens := []string{"token1", "token2", "token3"}
resp, err := client.SubscribeToTopic(ctx, tokens, "news")
if err != nil {
    log.Printf("Subscribe failed: %v", err)
    return
}
log.Printf("Subscribed %d tokens, failed %d", resp.SuccessCount, resp.FailureCount)
```

### Unsubscribe from Topic

```go
resp, err := client.UnsubscribeFromTopic(ctx, tokens, "news")
```

## Error Handling

```go
resp, err := client.Send(ctx, msg)
if err != nil {
    // Check specific error types
    if fcm.IsUnregistered(err) {
        // Token is no longer valid - remove from database
        removeToken(msg.Token)
        return
    }

    if fcm.IsQuotaExceeded(err) {
        // Rate limited - back off and retry
        time.Sleep(time.Minute)
        return
    }

    if fcm.IsRetryable(err) {
        // Temporary error - retry with backoff
        return retryWithBackoff(ctx, msg)
    }

    if fcm.IsSenderMismatch(err) {
        // Token belongs to different sender
        log.Printf("Sender mismatch for token")
        return
    }

    // Get detailed API error
    var apiErr *fcm.APIError
    if errors.As(err, &apiErr) {
        log.Printf("API error: %s (code: %s, status: %d)",
            apiErr.Message, apiErr.Code, apiErr.StatusCode)
    }

    return
}
```

## Global Client

```go
// Initialize once at startup
fcm.SetDefaultClient(client)

// Use anywhere
resp, err := fcm.Send(ctx, msg)
```

## Message Types Reference

### Message

The main message structure:

```go
type Message struct {
    Token        string            // Device registration token
    Topic        string            // Topic name (without /topics/ prefix)
    Condition    string            // Topic condition expression
    Notification *Notification     // Notification payload
    Data         map[string]string // Custom data payload
    Android      *AndroidConfig    // Android-specific options
    Webpush      *WebpushConfig    // Web Push options
    APNS         *APNSConfig       // iOS options
    FCMOptions   *FCMOptions       // FCM-specific options
}
```

### Notification

Basic notification payload:

```go
type Notification struct {
    Title    string // Notification title
    Body     string // Notification body
    ImageURL string // Image URL (optional)
}
```

### AndroidConfig

Android-specific options:

```go
type AndroidConfig struct {
    CollapseKey           string               // Collapse key for message grouping
    Priority              AndroidPriority      // "normal" or "high"
    TTL                   string               // Time-to-live (e.g., "3600s")
    RestrictedPackageName string               // Target package name
    Data                  map[string]string    // Android-specific data
    Notification          *AndroidNotification // Android notification options
    FCMOptions            *AndroidFCMOptions   // Analytics options
    DirectBootOK          bool                 // Allow during direct boot
}
```

### WebpushConfig

Web Push options:

```go
type WebpushConfig struct {
    Headers      map[string]string    // Web Push headers
    Data         map[string]string    // Custom data
    Notification *WebpushNotification // Web notification options
    FCMOptions   *WebpushFCMOptions   // Link and analytics
}
```

### APNSConfig

iOS options:

```go
type APNSConfig struct {
    Headers    map[string]string // APNS headers
    Payload    *APNSPayload      // APNS payload
    FCMOptions *APNSFCMOptions   // Image and analytics
}
```

## Service Account Setup

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Select your project
3. Go to Project Settings > Service Accounts
4. Click "Generate New Private Key"
5. Save the JSON file securely
6. Use the file path or contents with `NewClient`

## Best Practices

1. **Token Management**: Remove tokens when you receive `UNREGISTERED` errors
2. **Retry Logic**: Implement exponential backoff for retryable errors
3. **Batch Sends**: Use `SendMulticast` for multiple devices (up to 500)
4. **Topics**: Use topics for broadcasting to many devices
5. **Data Messages**: For background handling, use data-only messages
6. **Priority**: Use high priority sparingly (time-sensitive only)
7. **Collapse Keys**: Use to prevent notification spam

## Payload Size Limits

- **Data message**: 4KB maximum
- **Notification message**: 4KB maximum (platform-specific payloads included)

## Rate Limits

FCM has rate limits that vary by message type:
- Topic messages: 1000/second
- Device messages: No hard limit, but be reasonable
- Multicast: 500 tokens per request

When rate limited, you'll receive `ErrQuotaExceeded`. Implement backoff and retry.
