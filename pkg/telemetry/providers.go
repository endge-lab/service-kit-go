package telemetry

import (
	"context"
	"strings"
	"time"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Config описывает общие OTEL-настройки runtime.
type Config struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	OTLPEndpoint    string
	OTLPInsecure    bool
	MetricsInterval time.Duration
	TraceSampleMode string
}

// Providers объединяет tracer/meter/resource и умеет их корректно завершать.
type Providers struct {
	resource       *resource.Resource
	propagator     propagation.TextMapPropagator
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// NewProviders поднимает tracer и meter provider. При пустом endpoint работает в noop-режиме.
func NewProviders(ctx context.Context, cfg Config, logger *zap.Logger) (*Providers, error) {
	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	tracerProvider, err := newTraceProvider(ctx, cfg, res, logger)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(ctx, cfg, res, logger)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(meterProvider)

	return &Providers{
		resource:       res,
		propagator:     propagator,
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}, nil
}

// Resource возвращает общий telemetry resource.
func (p *Providers) Resource() *resource.Resource {
	if p == nil {
		return nil
	}

	return p.resource
}

// Propagator возвращает text map propagator.
func (p *Providers) Propagator() propagation.TextMapPropagator {
	if p == nil || p.propagator == nil {
		return propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	return p.propagator
}

// Tracer возвращает именованный tracer.
func (p *Providers) Tracer(name string) trace.Tracer {
	if p == nil || p.tracerProvider == nil {
		return otel.Tracer(strings.TrimSpace(name))
	}

	return p.tracerProvider.Tracer(strings.TrimSpace(name))
}

// Meter возвращает именованный meter.
func (p *Providers) Meter(name string) metric.Meter {
	if p == nil || p.meterProvider == nil {
		return otel.Meter(strings.TrimSpace(name))
	}

	return p.meterProvider.Meter(strings.TrimSpace(name))
}

// Shutdown корректно гасит meter и tracer.
func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var firstErr error
	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = serviceerrors.Wrap(err, "telemetry.meter_shutdown_failed", "Не удалось завершить meter provider", 500)
		}
	}

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = serviceerrors.Wrap(err, "telemetry.tracer_shutdown_failed", "Не удалось завершить tracer provider", 500)
		}
	}

	return firstErr
}

func newResource(cfg Config) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(strings.TrimSpace(cfg.ServiceName)),
			semconv.ServiceVersion(strings.TrimSpace(cfg.ServiceVersion)),
			attribute.String("deployment.environment", strings.TrimSpace(cfg.Environment)),
		),
	)
}

func newTraceProvider(ctx context.Context, cfg Config, res *resource.Resource, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	sampler := sdktrace.NeverSample()
	if strings.EqualFold(strings.TrimSpace(cfg.TraceSampleMode), "always") {
		sampler = sdktrace.AlwaysSample()
	}

	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		if logger != nil {
			logger.Warn("trace exporter disabled: endpoint is empty")
		}
		return sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sampler),
		), nil
	}

	clientOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	}
	if cfg.OTLPInsecure {
		clientOptions = append(clientOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(clientOptions...))
	if err != nil {
		return nil, serviceerrors.Wrap(err, "telemetry.trace_exporter_failed", "Не удалось создать trace exporter", 500)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	), nil
}

func newMeterProvider(ctx context.Context, cfg Config, res *resource.Resource, logger *zap.Logger) (*sdkmetric.MeterProvider, error) {
	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		if logger != nil {
			logger.Warn("metric exporter disabled: endpoint is empty")
		}
		return sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		), nil
	}

	clientOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	}
	if cfg.OTLPInsecure {
		clientOptions = append(clientOptions, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, clientOptions...)
	if err != nil {
		return nil, serviceerrors.Wrap(err, "telemetry.metric_exporter_failed", "Не удалось создать metric exporter", 500)
	}

	interval := cfg.MetricsInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(interval))),
	), nil
}
