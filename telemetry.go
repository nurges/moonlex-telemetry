// Package telemetry wires up structured logging, error reporting, tracing,
// and metrics for Moonlex backends, per moonlex-infra/docs/TELEMETRY.md.
// Sentry and OTel are no-ops when their DSN/endpoint is empty.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config carries plain values — map from your app's own config package.
type Config struct {
	// App is the service name (matches the systemd unit / scrape job).
	App string
	// Version is the build stamp (git short SHA via -ldflags "-X main.version=...").
	Version string
	// LogLevel for the default slog JSON logger.
	LogLevel slog.Level
	// SentryDSN enables Sentry when non-empty.
	SentryDSN string
	// SentryEnvironment tags events ("production" | "development").
	SentryEnvironment string
	// SentryTracesSampleRate defaults to 0.05 when zero.
	SentryTracesSampleRate float64
	// OTELEndpoint enables OTLP/HTTP trace export when non-empty.
	OTELEndpoint string
}

// FromEnv builds a Config from the standard env vars
// (LOG_LEVEL, SENTRY_DSN, SENTRY_ENVIRONMENT, OTEL_EXPORTER_OTLP_ENDPOINT).
func FromEnv(app, version string) Config {
	return Config{
		App:               app,
		Version:           version,
		LogLevel:          parseLevel(os.Getenv("LOG_LEVEL")),
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: envOr("SENTRY_ENVIRONMENT", "production"),
		OTELEndpoint:      os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}

// Stack is the bundle of subsystems initialized at startup.
type Stack struct {
	tracerProvider *sdktrace.TracerProvider
}

// Init sets slog (JSON, canonical time/level/msg keys) as the default logger,
// then initializes Sentry and OTel when configured. Empty values mean the
// corresponding subsystem is a silent no-op (one telemetry.*.disabled line).
func Init(cfg Config) (*Stack, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	stack := &Stack{}

	if cfg.SentryDSN != "" {
		rate := cfg.SentryTracesSampleRate
		if rate == 0 {
			rate = 0.05
		}
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Release:          cfg.Version,
			Environment:      cfg.SentryEnvironment,
			EnableTracing:    true,
			TracesSampleRate: rate,
			AttachStacktrace: true,
		}); err != nil {
			return nil, fmt.Errorf("sentry init: %w", err)
		}
		slog.Info("telemetry.sentry.enabled", "release", cfg.Version, "environment", cfg.SentryEnvironment)
	} else {
		slog.Info("telemetry.sentry.disabled")
	}

	if cfg.OTELEndpoint != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.OTELEndpoint))
		if err != nil {
			return nil, fmt.Errorf("otlp exporter: %w", err)
		}
		tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))
		otel.SetTracerProvider(tp)
		stack.tracerProvider = tp
		slog.Info("telemetry.otel.enabled", "endpoint", cfg.OTELEndpoint)
	} else {
		slog.Info("telemetry.otel.disabled")
	}

	return stack, nil
}

// Shutdown flushes pending events and stops background workers.
func (s *Stack) Shutdown(ctx context.Context) {
	if s.tracerProvider != nil {
		_ = s.tracerProvider.Shutdown(ctx)
	}
	sentry.Flush(2 * time.Second)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
