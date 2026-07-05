// Package health provides the standard Moonlex health endpoint
// (TELEMETRY.md: {"status":"ok","app":...,"version":...,"uptime_s":...}).
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler returns the canonical /health handler. Pass the process start time.
func Handler(app, version string, start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"app":      app,
			"version":  version,
			"uptime_s": int(time.Since(start).Seconds()),
		})
	}
}
