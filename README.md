# moonlex-telemetry

Shared Go telemetry for Moonlex backends — the reference implementation of
`moonlex-infra/docs/TELEMETRY.md`. Extracted from loremark's
`internal/observability` package.

```go
import (
    telemetry "github.com/nurges/moonlex-telemetry"
    "github.com/nurges/moonlex-telemetry/chimw"   // or echomw for echo apps
    "github.com/nurges/moonlex-telemetry/health"
    "github.com/nurges/moonlex-telemetry/posthog"
)

obs, err := telemetry.Init(telemetry.FromEnv("myapp", version))
defer obs.Shutdown(ctx)

r.Use(chimw.PrometheusMiddleware)                     // bare-name RED metrics
r.Method("GET", "/metrics", telemetry.MetricsHandler())
r.Get("/health", health.Handler("myapp", version, startTime))

ph := posthog.New("myapp", version)                   // no-op without POSTHOG_API_KEY
defer ph.Close()
ph.CapturePurchase(userID, posthog.Purchase{ ... })   // from webhook handlers
```

Env vars (empty = subsystem disabled): `LOG_LEVEL`, `SENTRY_DSN`,
`SENTRY_ENVIRONMENT`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `POSTHOG_API_KEY`,
`POSTHOG_HOST`.

Business metrics: `telemetry.Counter("myapp_things_total", "...")` — keep the
app prefix; only HTTP RED metrics use bare names (the scrape config injects
the `app` label).
