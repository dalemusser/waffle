// logging/logging.go
package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BootstrapLogger returns a development-friendly logger for early startup.
// It's safe to use before config is loaded and should log to stderr.
func BootstrapLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		// If we can't even build a logger, fall back to a no-op logger to avoid panics.
		return zap.NewNop()
	}
	return logger
}

// ValidLogLevels lists all valid zap log levels for validation.
var ValidLogLevels = []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}

// IsValidLogLevel checks if the given level string is a valid zap log level.
// Comparison is case-insensitive.
func IsValidLogLevel(level string) bool {
	level = strings.ToLower(level)
	for _, valid := range ValidLogLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// BuildLogger constructs the final logger based on log level and env.
// If env is "prod", it uses a JSON encoder; otherwise, it uses the development config.
//
// Valid log levels are: debug, info, warn, error, dpanic, panic, fatal (case-insensitive).
// If an invalid level is provided, it defaults to "info" and logs a warning
// to stderr so the misconfiguration is visible.
func BuildLogger(level, env string) (*zap.Logger, error) {
	var cfg zap.Config
	if env == "prod" {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	// RFC-3339 timestamps.
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Honor desired level (case-insensitive); warn and default to info on bad input.
	if err := cfg.Level.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		// Log warning to stderr so the misconfiguration is visible
		_, _ = os.Stderr.WriteString("WARNING: invalid log level \"" + level +
			"\"; valid levels are: debug, info, warn, error, dpanic, panic, fatal. Defaulting to \"info\".\n")
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Send logs to stderr by default.
	cfg.OutputPaths = []string{"stderr"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	return cfg.Build()
}

// MustBuildLogger is a convenience for main() that wants to fatal on logger build failure.
func MustBuildLogger(level, env string) *zap.Logger {
	logger, err := BuildLogger(level, env)
	if err != nil {
		// Last-resort: log to stderr and exit.
		_, _ = os.Stderr.WriteString("failed to build logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	return logger
}
