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
	ObserveRequestClient("GET", "/api/things", 200, time.Millisecond, ClientBot)

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`http_requests_total{client="human",method="GET",route="/api/things/{id}",status="200"}`,
		`http_requests_total{client="human",method="POST",route="/api/things",status="200"}`,
		`http_requests_total{client="bot",method="GET",route="/api/things",status="200"}`,
		`http_request_duration_seconds_bucket`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape output missing %q", want)
		}
	}
}

func TestClassifyUA(t *testing.T) {
	cases := []struct {
		ua   string
		want Client
	}{
		{"", ClientBot},
		{"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.4; +https://openai.com/gptbot)", ClientBot},
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", ClientBot},
		{"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", ClientBot},
		{"curl/8.5.0", ClientBot},
		{"python-requests/2.32.0", ClientBot},
		{"Go-http-client/2.0", ClientBot},
		{"facebookexternalhit/1.1", ClientBot},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.5.2 Safari/605.1.15", ClientHuman},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36", ClientHuman},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 19_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148", ClientHuman},
	}
	for _, c := range cases {
		if got := ClassifyUA(c.ua); got != c.want {
			t.Errorf("ClassifyUA(%q) = %q, want %q", c.ua, got, c.want)
		}
	}
}
