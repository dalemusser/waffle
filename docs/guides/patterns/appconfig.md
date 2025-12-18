# Examples of AppConfig Patterns

*How to extend your WAFFLE AppConfig for real applications*

Your WAFFLE project is generated with a minimal `AppConfig`:

```go
type AppConfig struct {
    Greeting string
}
```

This is intentional — **AppConfig is your space** to define configuration values unique to your application.

WAFFLE's `CoreConfig` handles framework-level settings (HTTP ports, TLS, CORS, logging, timeouts). AppConfig is where you put everything specific to YOUR application:

- Database connection strings
- External service API keys and endpoints
- Feature flags and application modes
- Business logic configuration

Below are common patterns you can use when expanding AppConfig.

---

## Configuration Sources

WAFFLE supports multiple configuration sources, merged with this precedence (highest to lowest):

1. **Command-line flags** — Override everything
2. **Environment variables** — `WAFFLE_*` prefix for core config
3. **Configuration files** — `config.yaml`, `config.json`, or `config.toml`
4. **`.env` files** — Loaded automatically if present
5. **Defaults** — Built-in sensible defaults

For app-specific configuration, you can use any of these methods in your `LoadConfig` function.

---

## Pattern 1: Simple Values

Add basic configuration fields:

```go
type AppConfig struct {
    Greeting   string
    AppName    string
    MaxPlayers int
}
```

Load in `LoadConfig`:

```go
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load(logger)
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        Greeting:   "Hello from WAFFLE!",
        AppName:    "My First WAFFLE App",
        MaxPlayers: 100,
    }

    return coreCfg, appCfg, nil
}
```

Use in handlers:

```go
w.Write([]byte(fmt.Sprintf("Welcome to %s!", appCfg.AppName)))
```

---

## Pattern 2: Environment Variables for Secrets

Load sensitive values from environment variables:

```go
type AppConfig struct {
    Greeting       string
    ExternalAPIKey string
}
```

Load securely in `LoadConfig`:

```go
appCfg := AppConfig{
    Greeting:       "Hello!",
    ExternalAPIKey: os.Getenv("EXTERNAL_API_KEY"),
}
```

You can also use a `.env` file in development:

```env
EXTERNAL_API_KEY=sk-abc123...
```

Then use in your code:

```go
client := myservice.NewClient(appCfg.ExternalAPIKey)
```

---

## Pattern 3: Feature Groups

Group related configuration values as your app grows:

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

Load:

```go
appCfg := AppConfig{
    Greeting: "Hello",
    Auth: AuthConfig{
        EnableGoogle: true,
        EnableGuest:  false,
    },
}
```

Use:

```go
if appCfg.Auth.EnableGoogle {
    // register Google login
}
```

---

## Pattern 4: Database Connection Parameters

Store database connection parameters in AppConfig, then use them in `ConnectDB`.

### Basic Approach

```go
type AppConfig struct {
    Greeting     string
    PostgresDSN  string
    DatabaseName string
}
```

Load from environment:

```go
appCfg := AppConfig{
    Greeting:     "Hello!",
    PostgresDSN:  os.Getenv("POSTGRES_DSN"),
    DatabaseName: "myapp",
}
```

### Using Pantry Helpers

WAFFLE Pantry provides database connection helpers that handle timeouts and connection verification:

```go
import (
    "github.com/dalemusser/waffle/pantry/db/postgres"
)

func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    pool, err := postgres.ConnectPool(appCfg.PostgresDSN, coreCfg.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("postgres connect: %w", err)
    }

    logger.Info("connected to PostgreSQL")
    return DBDeps{DB: pool}, nil
}
```

Available Pantry database helpers:

| Package | Import Path |
|---------|-------------|
| PostgreSQL | `github.com/dalemusser/waffle/pantry/db/postgres` |
| MySQL | `github.com/dalemusser/waffle/pantry/db/mysql` |
| SQLite | `github.com/dalemusser/waffle/pantry/db/sqlite` |
| MongoDB | `github.com/dalemusser/waffle/pantry/db/mongo` |
| Redis | `github.com/dalemusser/waffle/pantry/db/redis` |

---

## Pattern 5: Nested Configuration

For complex applications, use nested structs to organize configuration:

```go
type LoggingConfig struct {
    Level  string
    Format string
}

type AnalyticsConfig struct {
    Enabled bool
    APIKey  string
}

type DatabaseConfig struct {
    PostgresDSN string
    MaxConns    int
}

type AppConfig struct {
    Greeting  string
    Logging   LoggingConfig
    Analytics AnalyticsConfig
    Database  DatabaseConfig
}
```

Load in `LoadConfig`:

```go
appCfg := AppConfig{
    Greeting: "Hello!",
    Logging: LoggingConfig{
        Level:  "info",
        Format: "json",
    },
    Analytics: AnalyticsConfig{
        Enabled: true,
        APIKey:  os.Getenv("ANALYTICS_API_KEY"),
    },
    Database: DatabaseConfig{
        PostgresDSN: os.Getenv("POSTGRES_DSN"),
        MaxConns:    20,
    },
}
```

This structure keeps AppConfig readable and maintainable as your application grows.

---

## Pattern 6: Validation in ValidateConfig

Use `ValidateConfig` to enforce required fields and invariants:

```go
func ValidateConfig(coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) error {
    if appCfg.Database.PostgresDSN == "" {
        return fmt.Errorf("POSTGRES_DSN is required")
    }

    if appCfg.Analytics.Enabled && appCfg.Analytics.APIKey == "" {
        return fmt.Errorf("ANALYTICS_API_KEY is required when analytics is enabled")
    }

    if appCfg.Database.MaxConns < 1 || appCfg.Database.MaxConns > 100 {
        return fmt.Errorf("Database.MaxConns must be between 1 and 100")
    }

    return nil
}
```

Validation runs after `LoadConfig` but before `ConnectDB`, so the app fails fast with a clear error if configuration is invalid.

---

## Summary

`AppConfig` is **your** configuration layer — WAFFLE gives you a clean place to put settings that belong to your app.

These patterns demonstrate how to structure:

- Simple values (strings, numbers, booleans)
- API keys and secrets (from environment variables)
- Feature-specific groups
- Database connection parameters (with Pantry helpers)
- Nested configuration for larger services
- Validation to fail fast on bad configuration

Your WAFFLE app will grow into these patterns naturally over time. Mix and match them as needed.

---

## See Also

- [Configuration Reference](../../core/configuration.md) — CoreConfig options and loading
- [PostgreSQL Guide](../databases/postgres.md) — Database connection patterns
- [MySQL Guide](../databases/mysql.md) — MySQL with WAFFLE
- [MongoDB Guide](../databases/mongo.md) — MongoDB with WAFFLE

