# Notify - Web Push Notifications

The `notify` package provides Web Push notification support using VAPID (Voluntary Application Server Identification) for authentication. This enables sending push notifications to browsers without requiring Firebase or other third-party services.

## Features

- **VAPID Authentication**: Generate and manage VAPID keys for server identification
- **RFC 8291 Encryption**: Payload encryption using aes128gcm content encoding
- **Standard Web Push**: Works with all major browsers (Chrome, Firefox, Edge, Safari)
- **Subscription Management**: Parse and validate push subscriptions from clients
- **Notification Builder**: Fluent API for building notification payloads
- **Batch Sending**: Send to multiple subscriptions efficiently
- **Error Handling**: Detect expired subscriptions, rate limiting, and retryable errors

## Installation

```go
import "waffle/notify"
```

## Quick Start

### 1. Generate VAPID Keys (one-time setup)

```go
// Generate new VAPID keys
keys, err := notify.GenerateVAPIDKeys()
if err != nil {
    log.Fatal(err)
}

// Set the subject (required - your contact email or URL)
keys.WithSubject("mailto:admin@example.com")

// Save keys for later use
jsonKeys, _ := keys.ExportJSON()
os.WriteFile("vapid-keys.json", jsonKeys, 0600)

// The public key to give to the browser
fmt.Printf("Public Key: %s\n", keys.ApplicationServerKey())
```

### 2. Create a Client

```go
// Load saved keys
jsonKeys, _ := os.ReadFile("vapid-keys.json")
keys, err := notify.LoadVAPIDKeysFromJSON(jsonKeys)
if err != nil {
    log.Fatal(err)
}

// Create client
client, err := notify.NewClient(notify.Config{
    VAPIDKeys: keys,
})
if err != nil {
    log.Fatal(err)
}

// Optionally set as default
notify.SetDefaultClient(client)
```

### 3. Handle Subscription from Browser

When the browser subscribes, it sends a subscription object:

```go
// Receive subscription JSON from client
subscriptionJSON := []byte(`{
    "endpoint": "https://fcm.googleapis.com/fcm/send/...",
    "keys": {
        "p256dh": "BNcR...",
        "auth": "tBH..."
    }
}`)

// Parse subscription
sub, err := notify.NewSubscription(subscriptionJSON)
if err != nil {
    http.Error(w, "Invalid subscription", http.StatusBadRequest)
    return
}

// Save subscription for later use
store.Save(userID, sub)
```

### 4. Send Notifications

```go
// Simple notification
notification := notify.NewNotification("Hello!").
    Body("You have a new message").
    Icon("/icons/message.png").
    Build()

payload, _ := json.Marshal(notification)

msg := &notify.Message{
    Payload: payload,
    TTL:     3600,
    Urgency: notify.UrgencyHigh,
}

resp, err := client.Send(ctx, sub, msg)
if err != nil {
    if notify.IsExpired(err) {
        // Remove invalid subscription
        store.Delete(userID, sub.Endpoint)
        return
    }
    log.Printf("Failed to send: %v", err)
    return
}

log.Printf("Sent! Status: %d", resp.StatusCode)
```

## Client-Side JavaScript

### Service Worker Registration

```javascript
// Register service worker
const registration = await navigator.serviceWorker.register('/sw.js');

// Get VAPID public key from server
const vapidPublicKey = 'YOUR_PUBLIC_KEY_FROM_SERVER';

// Convert to Uint8Array
function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
        .replace(/\-/g, '+')
        .replace(/_/g, '/');
    const rawData = atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    for (let i = 0; i < rawData.length; ++i) {
        outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
}

// Subscribe to push
const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey)
});

// Send subscription to server
await fetch('/api/push/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription)
});
```

### Service Worker (sw.js)

```javascript
self.addEventListener('push', function(event) {
    const data = event.data ? event.data.json() : {};

    const options = {
        body: data.body,
        icon: data.icon,
        badge: data.badge,
        image: data.image,
        tag: data.tag,
        data: data.data,
        actions: data.actions,
        renotify: data.renotify,
        requireInteraction: data.requireInteraction,
        silent: data.silent,
        vibrate: data.vibrate
    };

    event.waitUntil(
        self.registration.showNotification(data.title, options)
    );
});

self.addEventListener('notificationclick', function(event) {
    event.notification.close();

    const data = event.notification.data || {};
    const url = data.url || '/';

    event.waitUntil(
        clients.openWindow(url)
    );
});
```

## Notification Builder

```go
// Build a rich notification
notification := notify.NewNotification("New Order").
    Body("Order #12345 has been placed").
    Icon("/icons/order.png").
    Badge("/icons/badge.png").
    Image("/images/product.jpg").
    Tag("order-12345").
    Data(map[string]any{
        "orderId": "12345",
        "url":     "/orders/12345",
    }).
    Action("view", "View Order", "/icons/view.png").
    Action("dismiss", "Dismiss").
    RequireInteraction().
    Build()

// Convert to message
msg, _ := notify.NewNotification("Hello").
    Body("World").
    Message()

// Or with options
msg, _ := notify.NewNotification("Urgent").
    Body("Action required").
    MessageWithOptions(
        3600,                  // TTL
        notify.UrgencyHigh,    // Urgency
        "alerts",              // Topic
    )
```

## Sending to Multiple Subscriptions

```go
// Get all subscriptions for a user
subs, _ := store.Get(userID)

// Send to all
result, err := client.SendMultiple(ctx, subs, msg)
if err != nil {
    log.Printf("Batch send failed: %v", err)
    return
}

log.Printf("Success: %d, Failed: %d", result.SuccessCount, result.FailureCount)

// Clean up expired subscriptions
for _, sub := range result.ExpiredSubscriptions() {
    store.Delete(userID, sub.Endpoint)
}
```

## Message Options

```go
msg := &notify.Message{
    // The notification payload (JSON)
    Payload: payload,

    // Time-to-live in seconds (default: 86400 = 24 hours)
    TTL: 3600,

    // Message urgency
    // - UrgencyVeryLow: no wake-up
    // - UrgencyLow: may wait for WiFi
    // - UrgencyNormal: deliver normally
    // - UrgencyHigh: deliver immediately
    Urgency: notify.UrgencyHigh,

    // Topic for message replacement
    // New messages with same topic replace old ones
    Topic: "chat-123",
}
```

## Error Handling

```go
resp, err := client.Send(ctx, sub, msg)
if err != nil {
    // Check for expired subscription (404 or 410)
    if notify.IsExpired(err) {
        store.Delete(userID, sub.Endpoint)
        return
    }

    // Check for rate limiting (429)
    if notify.IsRateLimited(err) {
        // Back off and retry
        time.Sleep(time.Minute)
        return
    }

    // Check if error is retryable
    if notify.IsRetryable(err) {
        // Retry with backoff
        return retryWithBackoff(ctx, sub, msg)
    }

    // Get HTTP error details
    var httpErr *notify.HTTPError
    if errors.As(err, &httpErr) {
        log.Printf("HTTP %d: %s", httpErr.StatusCode, httpErr.Message)
    }

    return
}
```

## VAPID Key Management

### Loading Keys

```go
// From JSON
keys, err := notify.LoadVAPIDKeysFromJSON(jsonData)

// From PEM file
pemData, _ := os.ReadFile("private-key.pem")
keys, err := notify.LoadVAPIDKeysFromPEM(pemData, "mailto:admin@example.com")

// From raw values
keys, err := notify.NewVAPIDKeys(publicKeyB64, privateKeyB64, "mailto:admin@example.com")
```

### Exporting Keys

```go
// To JSON
jsonData, err := keys.ExportJSON()

// To PEM
pemData, err := keys.ExportPEM()
```

## Subscription Store

The package includes a simple in-memory store, but you should implement `SubscriptionStore` for production:

```go
type SubscriptionStore interface {
    Save(userID string, sub *Subscription) error
    Get(userID string) ([]*Subscription, error)
    Delete(userID string, endpoint string) error
    DeleteAll(userID string) error
}
```

Example database implementation:

```go
type DBSubscriptionStore struct {
    db *sql.DB
}

func (s *DBSubscriptionStore) Save(userID string, sub *Subscription) error {
    _, err := s.db.Exec(`
        INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
        VALUES (?, ?, ?, ?)
        ON CONFLICT (endpoint) DO UPDATE
        SET p256dh = ?, auth = ?
    `, userID, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth,
       sub.Keys.P256dh, sub.Keys.Auth)
    return err
}

func (s *DBSubscriptionStore) Get(userID string) ([]*Subscription, error) {
    rows, err := s.db.Query(`
        SELECT endpoint, p256dh, auth
        FROM push_subscriptions
        WHERE user_id = ?
    `, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var subs []*Subscription
    for rows.Next() {
        var endpoint, p256dh, auth string
        if err := rows.Scan(&endpoint, &p256dh, &auth); err != nil {
            return nil, err
        }
        sub, _ := notify.NewSubscriptionFromParts(endpoint, p256dh, auth)
        subs = append(subs, sub)
    }
    return subs, nil
}
```

## Push Service Compatibility

The package works with all Web Push-compatible push services:

| Browser | Push Service |
|---------|--------------|
| Chrome  | FCM (googleapis.com) |
| Firefox | Mozilla (mozilla.com) |
| Edge    | Microsoft (windows.com) |
| Safari  | Apple (apple.com) |

Detect the push service:

```go
service := sub.PushService()
switch service {
case notify.PushServiceGoogle:
    // Chrome/Chromium
case notify.PushServiceMozilla:
    // Firefox
case notify.PushServiceMicrosoft:
    // Edge
case notify.PushServiceApple:
    // Safari
}
```

## Best Practices

1. **VAPID Subject**: Always set a valid contact email or URL
2. **Handle Expiration**: Remove subscriptions that return 404 or 410
3. **Respect Rate Limits**: Implement backoff when receiving 429 errors
4. **Use Topics**: For replaceable notifications (e.g., unread count)
5. **Set Appropriate TTL**: Short for time-sensitive, longer for persistent
6. **Use Urgency**: Low urgency for non-critical to save battery
7. **Keep Payloads Small**: Maximum 4KB, aim for under 2KB

## Payload Size Limits

- Maximum encrypted payload: 4096 bytes
- Recommended: Keep under 2KB for compatibility
- For larger data: Include a URL and fetch details in service worker
