// Package telemetry configures OpenTelemetry tracing and Prometheus metrics.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
)

// Provider holds OTel SDK providers for lifecycle management.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
	logger         *zap.Logger
}

// Setup initialises OpenTelemetry tracing + Prometheus metrics.
// Returns a Provider whose Shutdown() must be deferred.
func Setup(ctx context.Context, cfg config.TelemetryConfig, logger *zap.Logger) (*Provider, error) {
	p := &Provider{logger: logger}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
		resource.WithOS(),
		resource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// ── Tracing ──────────────────────────────────────────────────────────────
	if cfg.TracingEnabled && cfg.OTLPEndpoint != "" {
		exp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			logger.Warn("tracing exporter init failed (non-fatal)", zap.Error(err))
		} else {
			sampler := sdktrace.TraceIDRatioBased(cfg.SamplingRate)
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exp),
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			p.tracerProvider = tp
			logger.Info("OpenTelemetry tracing enabled",
				zap.String("endpoint", cfg.OTLPEndpoint),
				zap.Float64("sampling_rate", cfg.SamplingRate),
			)
		}
	}

	// ── Metrics (Prometheus) ──────────────────────────────────────────────────
	if cfg.MetricsEnabled {
		promExporter, err := prometheus.New()
		if err != nil {
			return nil, fmt.Errorf("prometheus exporter: %w", err)
		}
		mp := metric.NewMeterProvider(
			metric.WithReader(promExporter),
			metric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		p.meterProvider = mp
		logger.Info("Prometheus metrics enabled")
	}

	return p, nil
}

// Tracer returns a named tracer.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Shutdown gracefully shuts down OTel providers.
func (p *Provider) Shutdown(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil {
			p.logger.Error("tracer provider shutdown", zap.Error(err))
		}
	}
	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil {
			p.logger.Error("meter provider shutdown", zap.Error(err))
		}
	}
}
