// Package chimw provides the Prometheus middleware for chi routers.
package chimw

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	telemetry "github.com/nurges/moonlex-telemetry"
)

// PrometheusMiddleware records http_requests_total + http_request_duration_seconds.
// Mount it on the chi router so the matched route pattern is available;
// unmatched paths are collapsed to "unmatched" (crawler/scanner traffic would
// otherwise create unbounded label combinations).
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		telemetry.ObserveRequestClient(r.Method, routePattern(r), ww.Status(), time.Since(start), telemetry.ClassifyUA(r.UserAgent()))
	})
}

func routePattern(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if p := ctx.RoutePattern(); p != "" {
			return p
		}
	}
	return "unmatched"
}
