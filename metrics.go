package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RED metrics use bare standard names — the `app` dimension is injected by the
// Prometheus scrape config (monitored_apps in moonlex-infra), so cross-app
// queries like sum by (app) (rate(http_requests_total[5m])) just work.

// httpRequestsTotal counts every HTTP request served, broken out by method,
// the ROUTE PATTERN (bounded cardinality — never the raw URL), and status.
var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests served, partitioned by method, route pattern, and response status.",
	},
	[]string{"method", "route", "status"},
)

// httpRequestDuration captures handler latency. Buckets cover sub-millisecond
// static reads through ~10s slow paths — default exponential buckets miss the
// long tail.
var httpRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request handler duration in seconds, partitioned by method and route pattern.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"method", "route"},
)

// ObserveRequest records one served request. The framework middlewares
// (chimw, echomw) call this; use it directly for exotic setups.
func ObserveRequest(method, route string, status int, duration time.Duration) {
	if status == 0 {
		// Handler didn't explicitly WriteHeader → net/http defaults to 200.
		status = http.StatusOK
	}
	httpRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// MetricsHandler returns the Prometheus scrape endpoint handler. Mount it at
// /metrics. Default registry — process_*, go_*, and promhttp_* come free.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
