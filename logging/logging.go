// logging/logging.go
package logging

import (
	"os"

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

// BuildLogger constructs the final logger based on log level and env.
// If env is "prod", it uses a JSON encoder; otherwise, it uses the development config.
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

	// Honor desired level; default to info on bad input.
	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
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
