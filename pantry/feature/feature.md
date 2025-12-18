# feature - Feature Flags

The `feature` package provides a complete feature flag system for Go applications, supporting gradual rollouts, A/B testing, user targeting, and multiple storage backends.

## Features

- Simple on/off flags
- Percentage-based rollouts with consistent hashing
- User and group targeting
- Rule-based evaluation with multiple operators
- A/B testing with variants
- Multiple storage backends (memory, JSON file, environment variables)
- HTTP middleware for request context
- Admin API for flag management
- Thread-safe design

## Installation

```go
import "github.com/yourusername/waffle/feature"
```

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/yourusername/waffle/feature"
)

func main() {
    // Create a feature flag manager
    manager := feature.NewMemoryManager()

    // Register a simple flag
    manager.RegisterSimple("dark-mode", false)

    // Register a flag with percentage rollout
    manager.Register(&feature.Flag{
        Key:        "new-checkout",
        Name:       "New Checkout Flow",
        Enabled:    false,
        Percentage: 25, // Enable for 25% of users
    })

    // Register a flag with group targeting
    manager.Register(&feature.Flag{
        Key:     "beta-features",
        Enabled: false,
        Groups:  []string{"beta-testers", "employees"},
    })

    // Check flags
    if feature.IsEnabled("dark-mode") {
        fmt.Println("Dark mode is enabled!")
    }

    // Check with user context
    ctx := feature.NewEvalContext().
        WithUserID("user-123").
        WithGroups("beta-testers")

    if manager.IsEnabledFor("beta-features", ctx) {
        fmt.Println("Beta features enabled for this user!")
    }

    // Use in HTTP handler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        evalCtx := feature.FromContext(r.Context())
        if manager.IsEnabledFor("new-checkout", evalCtx) {
            // Show new checkout
        } else {
            // Show old checkout
        }
    })
}
```

## Flag Configuration

### Simple Flags

```go
// Register a simple on/off flag
manager.RegisterSimple("feature-x", true)

// Or with full configuration
manager.Register(&feature.Flag{
    Key:         "feature-x",
    Name:        "Feature X",
    Description: "Enables the new Feature X",
    Enabled:     true,
})
```

### Percentage Rollouts

Gradually roll out features to a percentage of users:

```go
manager.Register(&feature.Flag{
    Key:        "new-ui",
    Enabled:    false,
    Percentage: 10, // Start with 10% of users
})

// Increase rollout over time
manager.SetPercentage("new-ui", 25)
manager.SetPercentage("new-ui", 50)
manager.SetPercentage("new-ui", 100)
```

Percentage rollouts use consistent hashing, so the same user always gets the same result.

### Group Targeting

Enable features for specific user groups:

```go
manager.Register(&feature.Flag{
    Key:     "admin-tools",
    Enabled: false,
    Groups:  []string{"admins", "super-admins"},
})

// Check with groups
ctx := feature.NewEvalContext().
    WithGroups("admins", "editors")

manager.IsEnabledFor("admin-tools", ctx) // true (user is in "admins")
```

### Rule-Based Targeting

Create complex targeting rules:

```go
manager.Register(&feature.Flag{
    Key:     "premium-feature",
    Enabled: false,
    Rules: []feature.Rule{
        // Enable for premium users
        {
            Attribute: "plan",
            Operator:  "eq",
            Value:     "premium",
            Enabled:   true,
        },
        // Enable for users in specific countries
        {
            Attribute: "country",
            Operator:  "in",
            Value:     []string{"US", "CA", "UK"},
            Enabled:   true,
        },
        // Enable for users with version >= 2.0
        {
            Attribute: "app_version",
            Operator:  "gte",
            Value:     2.0,
            Enabled:   true,
        },
    },
})

// Evaluate with attributes
ctx := feature.NewEvalContext().
    WithUserID("user-123").
    WithAttribute("plan", "premium").
    WithAttribute("country", "US").
    WithAttribute("app_version", 2.1)

manager.IsEnabledFor("premium-feature", ctx) // true
```

**Supported Operators:**

| Operator | Description | Example |
|----------|-------------|---------|
| `eq`, `==` | Equals | `"plan" eq "premium"` |
| `neq`, `!=` | Not equals | `"status" neq "banned"` |
| `in` | Value in list | `"country" in ["US", "CA"]` |
| `nin`, `not_in` | Value not in list | `"country" nin ["CN", "RU"]` |
| `gt`, `>` | Greater than | `"age" gt 18` |
| `gte`, `>=` | Greater than or equal | `"version" gte 2.0` |
| `lt`, `<` | Less than | `"items" lt 10` |
| `lte`, `<=` | Less than or equal | `"price" lte 100` |
| `contains` | String/array contains | `"email" contains "@company.com"` |

### A/B Testing with Variants

Run experiments with multiple variants:

```go
manager.Register(&feature.Flag{
    Key:     "checkout-button",
    Enabled: true,
    Variants: []feature.Variant{
        {Key: "control", Value: "Buy Now", Weight: 50},
        {Key: "variant-a", Value: "Purchase", Weight: 25},
        {Key: "variant-b", Value: "Add to Cart", Weight: 25},
    },
})

// Get variant for user
ctx := feature.NewEvalContext().WithUserID("user-123")
variantKey, value := manager.GetVariant("checkout-button", ctx)

fmt.Printf("User gets variant %s: %v\n", variantKey, value)
// Output: User gets variant variant-a: Purchase
```

Variants use consistent hashing, so users always see the same variant.

## Evaluation Context

The `EvalContext` provides information for flag evaluation:

```go
ctx := feature.NewEvalContext().
    WithUserID("user-123").                    // For percentage rollouts
    WithGroups("beta", "premium").             // For group targeting
    WithAttribute("country", "US").            // Custom attributes
    WithAttribute("plan", "pro").
    WithAttributes(map[string]any{             // Multiple at once
        "version": "2.0",
        "platform": "ios",
    })
```

## Storage Backends

### Memory Store (Default)

```go
manager := feature.NewMemoryManager()
```

### JSON File Store

```go
store := feature.NewJSONStore("./flags.json")
manager := feature.NewManager(store)
```

### Environment Variables

```go
// Reads FEATURE_DARK_MODE, FEATURE_NEW_UI, etc.
store := feature.NewEnvStore("FEATURE_")
store.Register("dark-mode")  // Register keys for enumeration
store.Register("new-ui")

manager := feature.NewManager(store)
```

### Static Map

```go
store := feature.NewMapStoreSimple(map[string]bool{
    "feature-a": true,
    "feature-b": false,
})
manager := feature.NewManager(store)
```

### Composite Store (Multiple Sources)

```go
// Check env vars first, then JSON file
store := feature.NewCompositeStore(
    feature.NewEnvStore("FEATURE_"),
    feature.NewJSONStore("./flags.json"),
)
manager := feature.NewManager(store)
```

## HTTP Middleware

### Context Middleware

Automatically extract user context from requests:

```go
cfg := feature.DefaultMiddlewareConfig(manager)
mw := feature.Middleware(cfg)

// Use with your router
r.Use(mw)
```

**Default headers:**
- `X-User-ID`: User identifier
- `X-User-Groups`: Comma-separated groups

**Custom context builder:**

```go
cfg := feature.MiddlewareConfig{
    Manager: manager,
    ContextBuilder: func(r *http.Request) *feature.EvalContext {
        // Extract from JWT, session, etc.
        user := getUserFromRequest(r)
        return feature.NewEvalContext().
            WithUserID(user.ID).
            WithGroups(user.Roles...).
            WithAttribute("plan", user.Plan)
    },
}
```

### Require Feature Middleware

Gate routes behind feature flags:

```go
// Returns 404 if flag is disabled
r.With(feature.RequireFeature(manager, "new-api")).
    Get("/api/v2/users", newUsersHandler)

// Custom handler for disabled
r.With(feature.RequireFeatureFunc(manager, "beta", func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/waitlist", http.StatusFound)
})).Get("/beta", betaHandler)
```

### Feature-Switched Handlers

Serve different handlers based on flag:

```go
r.Get("/checkout", feature.HandlerFunc(manager, "new-checkout",
    newCheckoutHandler,  // Flag enabled
    oldCheckoutHandler,  // Flag disabled
))
```

## Admin API

Manage flags via HTTP:

```go
admin := feature.NewAdminHandler(manager)
r.Mount("/admin/features", admin)
```

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List all flags |
| GET | `/{key}` | Get flag details |
| POST | `/` | Create flag |
| PUT | `/{key}` | Update flag |
| DELETE | `/{key}` | Delete flag |

**Examples:**

```bash
# List flags
curl http://localhost:8080/admin/features

# Get flag
curl http://localhost:8080/admin/features/new-checkout

# Enable flag
curl -X PUT http://localhost:8080/admin/features/new-checkout \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Set rollout percentage
curl -X PUT http://localhost:8080/admin/features/new-checkout \
  -H "Content-Type: application/json" \
  -d '{"percentage": 50}'

# Create flag
curl -X POST http://localhost:8080/admin/features \
  -H "Content-Type: application/json" \
  -d '{"key": "new-feature", "enabled": false, "percentage": 10}'
```

## Client-Side Evaluation

Evaluate flags for client-side use:

```go
// Create evaluate endpoint
eval := feature.NewEvaluateHandler(manager)
r.Post("/api/features/evaluate", eval.ServeHTTP)
```

**Client request:**

```bash
curl -X POST http://localhost:8080/api/features/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "flags": ["dark-mode", "new-checkout"],
    "context": {
      "UserID": "user-123",
      "Groups": ["beta"]
    }
  }'
```

**Response:**

```json
{
  "dark-mode": true,
  "new-checkout": false
}
```

**Get all flags:**

```json
{"all": true, "context": {"UserID": "user-123"}}
```

## Global Manager

For simple applications:

```go
manager := feature.NewMemoryManager()
feature.SetGlobal(manager)

// Use anywhere
if feature.IsEnabled("my-feature") {
    // ...
}

// With context
ctx := feature.NewEvalContext().WithUserID("user-123")
if feature.IsEnabledFor("my-feature", ctx) {
    // ...
}

// From request context
if feature.IsEnabledCtx(r.Context(), "my-feature") {
    // ...
}
```

## Evaluation Hooks

Track flag evaluations for analytics:

```go
manager.OnEvaluate(func(flag string, ctx *feature.EvalContext, result bool) {
    log.Printf("Flag %s evaluated to %v for user %s",
        flag, result, ctx.UserID)

    // Send to analytics
    analytics.Track("feature_flag_evaluated", map[string]any{
        "flag":    flag,
        "result":  result,
        "user_id": ctx.UserID,
    })
})
```

## Default Values

Set defaults for unregistered flags:

```go
// Single default
manager.SetDefault("unregistered-flag", false)

// Multiple defaults
manager.SetDefaults(map[string]bool{
    "feature-a": true,
    "feature-b": false,
})
```

## Best Practices

1. **Use descriptive flag keys**: `new-checkout-flow` not `flag1`

2. **Clean up old flags**: Remove flags once fully rolled out

3. **Use percentage rollouts**: Gradually increase from 1% → 10% → 50% → 100%

4. **Monitor flag usage**: Use evaluation hooks for observability

5. **Default to safe values**: Disabled by default for new features

6. **Document flags**: Use Name and Description fields

7. **Group related flags**: Use naming conventions like `checkout.new-flow`, `checkout.skip-review`

## Manager Methods Reference

| Method | Description |
|--------|-------------|
| `NewManager(store)` | Create manager with store |
| `NewMemoryManager()` | Create manager with memory store |
| `Register(flag)` | Register a flag |
| `RegisterSimple(key, enabled)` | Register simple flag |
| `Get(key)` | Get flag by key |
| `All()` | Get all flags |
| `IsEnabled(key)` | Check if enabled (no context) |
| `IsEnabledFor(key, ctx)` | Check if enabled for context |
| `GetVariant(key, ctx)` | Get variant for flag |
| `Enable(key)` | Enable a flag |
| `Disable(key)` | Disable a flag |
| `SetEnabled(key, enabled)` | Set enabled state |
| `SetPercentage(key, pct)` | Set rollout percentage |
| `Delete(key)` | Delete a flag |
| `Reload()` | Reload from store |
| `SetDefault(key, enabled)` | Set default for unregistered flag |
| `SetDefaults(defaults)` | Set multiple defaults |
| `OnEvaluate(fn)` | Set evaluation callback |

## EvalContext Methods Reference

| Method | Description |
|--------|-------------|
| `NewEvalContext()` | Create new context |
| `WithUserID(id)` | Set user ID |
| `WithGroups(groups...)` | Set groups |
| `WithAttribute(key, value)` | Set attribute |
| `WithAttributes(attrs)` | Set multiple attributes |

## Store Interface

Implement custom stores:

```go
type Store interface {
    Load(key string) (*Flag, error)
    LoadAll() ([]*Flag, error)
    Save(flag *Flag) error
    Delete(key string) error
}
```

## Thread Safety

All manager operations are thread-safe. Flags can be read and modified from multiple goroutines safely.

```go
// Safe concurrent access
go manager.IsEnabled("flag-a")
go manager.IsEnabled("flag-b")
go manager.SetEnabled("flag-c", true)
```
