package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Business metrics keep an app prefix (e.g. "loremark_reveals_total") —
// they're app-specific by nature and the prefix avoids cross-app collisions.
// These thin helpers register on the default registry, matching MetricsHandler.

// Counter registers a business counter. Name it "<app>_<thing>_total".
func Counter(name, help string, labels ...string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, labels)
}

// Gauge registers a business gauge.
func Gauge(name, help string, labels ...string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labels)
}

// Histogram registers a business histogram with explicit buckets.
func Histogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(prometheus.HistogramOpts{Name: name, Help: help, Buckets: buckets}, labels)
}
