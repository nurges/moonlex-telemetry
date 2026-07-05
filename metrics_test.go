package telemetry

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestObserveRequestAndScrape(t *testing.T) {
	ObserveRequest("GET", "/api/things/{id}", 200, 42*time.Millisecond)
	ObserveRequest("POST", "/api/things", 0, time.Millisecond) // 0 → 200

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`http_requests_total{method="GET",route="/api/things/{id}",status="200"}`,
		`http_requests_total{method="POST",route="/api/things",status="200"}`,
		`http_request_duration_seconds_bucket`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape output missing %q", want)
		}
	}
}
