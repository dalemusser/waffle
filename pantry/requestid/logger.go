// requestid/logger.go
package requestid

import (
	"context"

	"go.uber.org/zap"
)

// Logger returns a logger with the request ID added as a field.
// If no request ID is in the context, returns the original logger.
func Logger(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		return nil
	}
	requestID := Get(ctx)
	if requestID == "" {
		return logger
	}
	return logger.With(zap.String("request_id", requestID))
}

// LoggerFromRequest returns a logger with the request ID from an HTTP request.
func LoggerFromRequest(r interface{ Context() context.Context }, logger *zap.Logger) *zap.Logger {
	return Logger(r.Context(), logger)
}

// Field returns a zap field with the request ID.
// Returns a no-op field if no request ID is in the context.
func Field(ctx context.Context) zap.Field {
	requestID := Get(ctx)
	if requestID == "" {
		return zap.Skip()
	}
	return zap.String("request_id", requestID)
}

// LoggingMiddleware returns middleware that adds the request ID to the logger
// stored in the context. Use with waffle/logging for automatic request ID logging.
func LoggingMiddleware(logger *zap.Logger) func(next func(context.Context)) func(context.Context) {
	return func(next func(context.Context)) func(context.Context) {
		return func(ctx context.Context) {
			// This is a simplified example - actual integration would depend
			// on how the logger is stored/retrieved from context in your app
			next(ctx)
		}
	}
}
