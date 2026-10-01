package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

)

var (
	TotalRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total Http requests processed, partitioned by status code and method. ",
		},
		[]string{"method","status","path"},
	)
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_second",
			Help: "Histogram of latencies for Http requests.",
			Buckets: prometheus.DefBuckets,

		},
		[]string{"method","path"},
	)
	RateLimitBlocks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "httpd_rate_limit_block_total",
			Help: "Total requests block due to rate limiting.",
		},
		[]string{"client_ip"},
	)
)
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		TotalRequests.WithLabelValues(r.Method, strconv.Itoa(wrapped.statusCode), path).Inc()
		RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}