// Package echomw provides the Prometheus middleware for echo routers
// (filesynth, stampmeister).
package echomw

import (
	"time"

	"github.com/labstack/echo/v4"

	telemetry "github.com/nurges/moonlex-telemetry"
)

// Prometheus records http_requests_total + http_request_duration_seconds using
// echo's matched route pattern (c.Path()) for bounded cardinality.
func Prometheus() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			route := c.Path()
			if route == "" {
				route = "unmatched"
			}
			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				}
			}
			telemetry.ObserveRequestClient(c.Request().Method, route, status, time.Since(start), telemetry.ClassifyUA(c.Request().UserAgent()))
			return err
		}
	}
}
