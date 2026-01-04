// config/appconfig.go
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AppKey defines a configuration key for an application.
// Apps register their config keys using this type, and WAFFLE handles
// loading from config files, environment variables, and command-line flags.
type AppKey struct {
	// Name is the key name (e.g., "session_name", "mongo_uri").
	// This is used as-is for config files and CLI flags.
	// For env vars, it's uppercased and prefixed (e.g., STRATAHUB_SESSION_NAME).
	Name string

	// Default is the default value if not set elsewhere.
	// Supported types: string, int, int64, bool, []string.
	Default any

	// Desc is a short description for --help output.
	Desc string
}

// AppConfigValues holds the loaded app configuration values.
// Keys are the AppKey.Name values, values are the loaded configuration.
type AppConfigValues map[string]any

// String returns a string value or empty string if not found/wrong type.
func (a AppConfigValues) String(key string) string {
	if v, ok := a[key].(string); ok {
		return v
	}
	return ""
}

// Int returns an int value or 0 if not found/wrong type.
// Handles both int and int64 (TOML/Viper returns int64 for integers).
func (a AppConfigValues) Int(key string) int {
	switch v := a[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// Int64 returns an int64 value or 0 if not found/wrong type.
// Handles both int64 and int for flexibility.
func (a AppConfigValues) Int64(key string) int64 {
	switch v := a[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// Bool returns a bool value or false if not found/wrong type.
func (a AppConfigValues) Bool(key string) bool {
	if v, ok := a[key].(bool); ok {
		return v
	}
	return false
}

// StringSlice returns a []string value or nil if not found/wrong type.
func (a AppConfigValues) StringSlice(key string) []string {
	if v, ok := a[key].([]string); ok {
		return v
	}
	return nil
}

// Duration parses a duration value from the config.
// Accepts:
//   - Duration strings: "10m", "1h30m", "90s", "2h"
//   - Numeric values: interpreted as seconds (e.g., 600 = 10 minutes)
//   - Plain numeric strings: "600" = 600 seconds
//
// Returns the default value if the key is not found, empty, or invalid.
// Use this for timeout, expiry, and interval configurations.
func (a AppConfigValues) Duration(key string, def time.Duration) time.Duration {
	raw := a[key]
	if raw == nil {
		return def
	}
	dur, err := parseDurationFlexible(raw, def)
	if err != nil {
		return def
	}
	return dur
}

// loadAppConfig loads app-specific configuration using the same precedence
// as WAFFLE core config: flags > env > config files > defaults.
//
// The envPrefix is used for environment variables (e.g., "STRATAHUB" means
// the key "session_name" maps to env var "STRATAHUB_SESSION_NAME").
//
// This function should be called after pflags are parsed and config files
// are loaded into the provided viper instance.
func loadAppConfig(logger *zap.Logger, v *viper.Viper, envPrefix string, keys []AppKey) AppConfigValues {
	if len(keys) == 0 {
		return make(AppConfigValues)
	}

	// Create a child viper for app config with the app's env prefix
	appV := viper.New()
	appV.SetEnvPrefix(envPrefix)
	appV.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	appV.AutomaticEnv()

	// Register each key
	for _, key := range keys {
		// Set default
		appV.SetDefault(key.Name, key.Default)

		// Bind env var
		_ = appV.BindEnv(key.Name)

		// Copy value from main viper if it was set in config file
		// (config files are loaded into the main viper instance)
		if v.IsSet(key.Name) {
			appV.Set(key.Name, v.Get(key.Name))
		}

		// Bind pflag if it was explicitly set
		if f := pflag.Lookup(key.Name); f != nil && f.Changed {
			_ = appV.BindPFlag(key.Name, f)
		}
	}

	// Build result map
	result := make(AppConfigValues, len(keys))
	for _, key := range keys {
		result[key.Name] = appV.Get(key.Name)
	}

	if logger != nil {
		// Log loaded app config (be careful not to log secrets)
		fields := make([]zap.Field, 0, len(keys))
		for _, key := range keys {
			// Skip keys that might contain secrets
			nameLower := strings.ToLower(key.Name)
			if strings.Contains(nameLower, "key") ||
				strings.Contains(nameLower, "secret") ||
				strings.Contains(nameLower, "password") ||
				strings.Contains(nameLower, "token") {
				fields = append(fields, zap.String(key.Name, "[REDACTED]"))
			} else {
				fields = append(fields, zap.Any(key.Name, result[key.Name]))
			}
		}
		logger.Info("app config loaded", fields...)
	}

	return result
}

// registerAppFlags registers command-line flags for app config keys.
// Must be called before pflag.Parse().
func registerAppFlags(keys []AppKey) error {
	for _, key := range keys {
		// Check if flag already exists
		if pflag.Lookup(key.Name) != nil {
			return fmt.Errorf("config key %q conflicts with existing flag", key.Name)
		}

		switch d := key.Default.(type) {
		case string:
			pflag.String(key.Name, d, key.Desc)
		case int:
			pflag.Int(key.Name, d, key.Desc)
		case int64:
			pflag.Int64(key.Name, d, key.Desc)
		case bool:
			pflag.Bool(key.Name, d, key.Desc)
		case []string:
			// For string slices, accept JSON array on command line
			pflag.String(key.Name, "", key.Desc+" (JSON array)")
		default:
			return fmt.Errorf("config key %q has unsupported default type %T", key.Name, key.Default)
		}
	}
	return nil
}
