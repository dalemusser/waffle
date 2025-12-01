# Examples of AppConfig Patterns  
*How to extend your WAFFLE AppConfig for real applications*

Your WAFFLE project is generated with a very small `AppConfig`:

```go
type AppConfig struct {
    Greeting string
}
```

This is intentional — **AppConfig is your space** to define configuration values unique to your application.  
Below are examples showing common patterns you can use when expanding it.

---

## 🧩 Pattern 1: Add simple string + number configuration

```go
type AppConfig struct {
    Greeting   string
    AppName    string
    MaxPlayers int
}
```

Loaded in `LoadConfig`:

```go
appCfg := AppConfig{
    Greeting:   "Hello from WAFFLE!",
    AppName:    "My First WAFFLE App",
    MaxPlayers: 100,
}
```

Use inside handlers:

```go
w.Write([]byte(fmt.Sprintf("Welcome to %s!", appCfg.AppName)))
```

---

## 🔒 Pattern 2: Configuration for API keys or secrets

You might load secrets from environment variables:

```go
type AppConfig struct {
    Greeting   string
    ExternalAPIKey string
}
```

Load them securely:

```go
key := os.Getenv("EXTERNAL_API_KEY")

appCfg := AppConfig{
    Greeting: "Hello!",
    ExternalAPIKey: key,
}
```

Then use them in your feature code:

```go
client := myservice.NewClient(appCfg.ExternalAPIKey)
```

---

## 🌍 Pattern 3: Feature-specific configuration groups

As apps grow, it helps to group related config values:

```go
type AuthConfig struct {
    EnableGoogle bool
    EnableGuest  bool
}

type AppConfig struct {
    Greeting string
    Auth     AuthConfig
}
```

Load it like:

```go
appCfg := AppConfig{
    Greeting: "Hello",
    Auth: AuthConfig{
        EnableGoogle: true,
        EnableGuest:  false,
    },
}
```

Usage:

```go
if appCfg.Auth.EnableGoogle {
    // register Google login
}
```

---

## 🗄️ Pattern 4: Database connection info

Even though WAFFLE keeps DB connections inside `DBDeps`,  
you might keep DB connection *parameters* in AppConfig.

```go
type AppConfig struct {
    Greeting string
    MongoURI string
    DatabaseName string
}
```

Load from env variables or config files:

```go
appCfg := AppConfig{
    Greeting: "Hello!",
    MongoURI: os.Getenv("MONGO_URI"),
    DatabaseName: "myapp",
}
```

Then in `ConnectDB`, use:

```go
client, err := mongo.Connect(ctx, options.Client().ApplyURI(appCfg.MongoURI))
db := client.Database(appCfg.DatabaseName)
```

---

## 🧠 Pattern 5: Advanced: nested or structured configuration

For complex apps:

```go
type LoggingConfig struct {
    Level string
}

type AnalyticsConfig struct {
    Enabled bool
    APIKey  string
}

type AppConfig struct {
    Greeting  string
    Logging   LoggingConfig
    Analytics AnalyticsConfig
}
```

Loaded in `LoadConfig`:

```go
appCfg := AppConfig{
    Greeting: "Hello!",
    Logging: LoggingConfig{
        Level: "info",
    },
    Analytics: AnalyticsConfig{
        Enabled: true,
        APIKey:  os.Getenv("ANALYTICS_API_KEY"),
    },
}
```

This structure keeps AppConfig readable and maintainable as your application grows.

---

# 📚 Summary

`AppConfig` is **your** configuration layer — WAFFLE gives you a clean place to put settings that belong to your app.

These examples demonstrate how to structure:

- simple values  
- API keys  
- feature-specific groups  
- database parameters  
- nested configuration for larger services  

Your WAFFLE app will grow into these patterns naturally over time.  
Feel free to mix and match them across features.

