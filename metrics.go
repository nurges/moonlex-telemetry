package telemetry

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RED metrics use bare standard names — the `app` dimension is injected by the
// Prometheus scrape config (monitored_apps in moonlex-infra), so cross-app
// queries like sum by (app) (rate(http_requests_total[5m])) just work.

// httpRequestsTotal counts every HTTP request served, broken out by method,
// the ROUTE PATTERN (bounded cardinality — never the raw URL), status, and
// client kind (human vs self-identified bot).
var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests served, partitioned by method, route pattern, response status, and client kind.",
	},
	[]string{"method", "route", "status", "client"},
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

// Client is the value of the `client` label on http_requests_total.
type Client string

const (
	ClientHuman Client = "human"
	ClientBot   Client = "bot"
)

// botUATokens match self-identified crawlers, previews, and script clients in
// a lowercased User-Agent. Classification is honest-bot only: a scraper faking
// a browser UA counts as human — Cloudflare-level bot signals never reach the
// app, so this is the best split available at this layer.
var botUATokens = []string{
	"bot", "crawl", "spider", "slurp", "headless", "scrapy",
	"python-requests", "python-urllib", "go-http-client", "curl/", "wget/",
	"okhttp", "axios/", "node-fetch", "libwww", "httpclient",
	"facebookexternalhit", "externalagent", "ia_archiver", "bingpreview",
}

// ClassifyUA maps a User-Agent header to a Client. An empty UA is a bot —
// every real browser sends one.
func ClassifyUA(userAgent string) Client {
	if userAgent == "" {
		return ClientBot
	}
	ua := strings.ToLower(userAgent)
	for _, tok := range botUATokens {
		if strings.Contains(ua, tok) {
			return ClientBot
		}
	}
	return ClientHuman
}

// ObserveRequestClient records one served request with an explicit client
// kind. The framework middlewares (chimw, echomw) call this with
// ClassifyUA(userAgent); use it directly for exotic setups.
func ObserveRequestClient(method, route string, status int, duration time.Duration, client Client) {
	if status == 0 {
		// Handler didn't explicitly WriteHeader → net/http defaults to 200.
		status = http.StatusOK
	}
	httpRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status), string(client)).Inc()
	httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// ObserveRequest is the pre-client-label API, kept so existing callers keep
// compiling. Requests recorded through it count as human — the same as every
// request did before the label existed.
func ObserveRequest(method, route string, status int, duration time.Duration) {
	ObserveRequestClient(method, route, status, duration, ClientHuman)
}

// MetricsHandler returns the Prometheus scrape endpoint handler. Mount it at
// /metrics. Default registry — process_*, go_*, and promhttp_* come free.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
