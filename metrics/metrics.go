// metrics/metrics.go
package metrics

import (
	"net/http"
	"strconv"
	"time"

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
func RegisterDefault(logger *zap.Logger) {
	// Go runtime metrics
	if err := prometheus.Register(collectors.NewGoCollector()); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			if logger != nil {
				logger.Fatal("failed to register Go collector", zap.Error(err))
			}
		}
	}

	// Process metrics
	if err := prometheus.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			if logger != nil {
				logger.Fatal("failed to register process collector", zap.Error(err))
			}
		}
	}

	// HTTP request histogram
	if err := prometheus.Register(reqDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			if logger != nil {
				logger.Fatal("failed to register HTTP request histogram", zap.Error(err))
			}
		}
	}
}

// HTTPMetrics is a middleware that records request duration into the
// http_request_duration_seconds histogram.
func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		statusCode := ww.Status()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		reqDuration.WithLabelValues(
			r.URL.Path,
			r.Method,
			strconv.Itoa(statusCode),
		).Observe(duration)
	})
}

// Handler returns an http.Handler that exposes the Prometheus metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}
