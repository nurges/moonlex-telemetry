// Package posthog provides env-gated server-side PostHog capture, including
// the unified purchase event stream (TELEMETRY.md): webhook handlers call
// CapturePurchase so every app has one consistent revenue event shape.
package posthog

import (
	"log/slog"
	"os"

	ph "github.com/posthog/posthog-go"
)

// Client wraps posthog-go; the zero value (or a nil *Client) is a no-op.
type Client struct {
	ph  ph.Client
	app string
}

// New creates a client from POSTHOG_API_KEY / POSTHOG_HOST. Returns a no-op
// client (never nil, never an error) when the key is unset.
func New(app, version string) *Client {
	key := os.Getenv("POSTHOG_API_KEY")
	if key == "" {
		slog.Info("telemetry.posthog.disabled")
		return &Client{}
	}
	host := os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://eu.i.posthog.com"
	}
	client, err := ph.NewWithConfig(key, ph.Config{Endpoint: host})
	if err != nil {
		slog.Error("telemetry.posthog.init_failed", "error", err.Error())
		return &Client{}
	}
	slog.Info("telemetry.posthog.enabled", "host", host)
	return &Client{ph: client, app: app}
}

// Capture sends one event with the standard super properties.
func (c *Client) Capture(distinctID, event string, props map[string]any) {
	if c == nil || c.ph == nil {
		return
	}
	p := ph.NewProperties().Set("app", c.app).Set("environment", envName())
	for k, v := range props {
		p.Set(k, v)
	}
	_ = c.ph.Enqueue(ph.Capture{DistinctId: distinctID, Event: event, Properties: p})
}

// Purchase is the unified purchase_completed payload.
type Purchase struct {
	Revenue       float64
	Currency      string
	ProductID     string
	Source        string // "revenuecat" | "polar" | "stripe" | "storekit"
	TransactionID string
}

// CapturePurchase emits purchase_completed with $insert_id = transaction id,
// making webhook retries dedupe-safe on the PostHog side.
func (c *Client) CapturePurchase(distinctID string, p Purchase) {
	c.Capture(distinctID, "purchase_completed", map[string]any{
		"revenue":        p.Revenue,
		"currency":       p.Currency,
		"product_id":     p.ProductID,
		"source":         p.Source,
		"transaction_id": p.TransactionID,
		"$insert_id":     p.TransactionID,
	})
}

// Close flushes the queue; call on shutdown.
func (c *Client) Close() {
	if c != nil && c.ph != nil {
		_ = c.ph.Close()
	}
}

func envName() string {
	if v := os.Getenv("SENTRY_ENVIRONMENT"); v != "" {
		return v
	}
	return "production"
}
