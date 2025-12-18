// metrics/metrics.go
package metrics

import (
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// reqDuration is a histogram of HTTP request durations in seconds, labeled
// by path, method, and status code.
var reqDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests.",
		// buckets in seconds
		Buckets: []float64{0.01, 0.1, 0.3, 1.2, 5},
	},
	[]string{"path", "method", "status"},
)

// RegisterDefault registers the default Go runtime and process collectors,
// plus the HTTP request duration histogram. It is safe (and intended) to call
// this once at startup.
//
// This function will panic if registration fails for reasons other than
// the collector already being registered. This ensures configuration errors
// are caught early rather than silently ignored.
func RegisterDefault(logger *zap.Logger) {
	// Go runtime metrics
	mustRegister(logger, "Go collector", collectors.NewGoCollector())

	// Process metrics
	mustRegister(logger, "process collector", collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// HTTP request histogram
	mustRegister(logger, "HTTP request histogram", reqDuration)
}

// mustRegister attempts to register a Prometheus collector. If registration
// fails for a reason other than AlreadyRegisteredError, it logs a fatal error
// (which calls os.Exit) or panics if no logger is provided.
func mustRegister(logger *zap.Logger, name string, c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// Already registered is fine - this can happen in tests or if
			// RegisterDefault is called multiple times.
			return
		}
		// Serious registration failure - this indicates a configuration problem
		// that should be fixed before the application can run properly.
		if logger != nil {
			logger.Fatal("failed to register "+name, zap.Error(err))
		} else {
			// No logger available - panic to ensure the error isn't silently ignored
			panic("metrics: failed to register " + name + ": " + err.Error())
		}
	}
}

// maxPathLabelLength is the maximum length for the path label to prevent
// unbounded cardinality and memory issues in Prometheus.
const maxPathLabelLength = 256

// HTTPMetrics is a middleware that records request duration into the
// http_request_duration_seconds histogram.
//
// It uses the chi route pattern (e.g., "/users/{id}") instead of the actual
// request path (e.g., "/users/123") to prevent label cardinality explosion.
// Paths longer than 256 characters are truncated with "..." to prevent
// unbounded memory growth in the metrics registry.
func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Default to HTTP/1.x if ProtoMajor is invalid (e.g., malformed request).
		protoMajor := r.ProtoMajor
		if protoMajor < 1 {
			protoMajor = 1
		}
		ww := middleware.NewWrapResponseWriter(w, protoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		statusCode := ww.Status()
		// Status 0 means WriteHeader was never called. Per net/http semantics,
		// this indicates a successful response (200 OK) since the handler completed
		// without explicitly setting a status. This is standard Go behavior:
		// handlers that write a body without calling WriteHeader get 200.
		// Note: If a panic occurs before WriteHeader, the recovery middleware
		// (logging.Recoverer) will set 500. This middleware should be placed
		// AFTER the recovery middleware in the chain to record accurate statuses.
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		// Clamp status code to valid HTTP range to prevent unbounded label cardinality.
		// While Go's standard library limits status codes to 3 digits (100-599),
		// a buggy handler could theoretically set an invalid value.
		if statusCode < 100 || statusCode > 599 {
			statusCode = http.StatusInternalServerError
		}

		// Use route pattern to avoid cardinality explosion from path parameters.
		// Falls back to raw path if route context is unavailable (non-chi routers).
		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				path = pattern
			}
		}

		// Truncate extremely long paths to prevent unbounded label cardinality.
		// Use truncateUTF8 to avoid splitting multi-byte characters.
		// Note: We don't log truncation because (1) it would require a logger dependency
		// and (2) it would log on every request for long paths. If you need to debug
		// truncated paths, check for labels ending in "..." in your metrics.
		if len(path) > maxPathLabelLength {
			// Ensure we have room for at least 1 char + "..."
			truncateLen := maxPathLabelLength - 3
			if truncateLen < 1 {
				truncateLen = 1
			}
			path = truncateUTF8(path, truncateLen) + "..."
		}

		reqDuration.WithLabelValues(
			path,
			r.Method,
			strconv.Itoa(statusCode),
		).Observe(duration)
	})
}

// Handler returns an http.Handler that exposes the Prometheus metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// truncateUTF8 truncates s to at most maxBytes bytes without splitting
// multi-byte UTF-8 characters. If s is already <= maxBytes, it is returned
// unchanged. Otherwise, it truncates at the last valid rune boundary.
// If maxBytes <= 0, returns an empty string.
func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	// At this point len(s) > maxBytes, so s[maxBytes] is a valid index.
	// Work backwards from maxBytes to find a valid rune boundary.
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
