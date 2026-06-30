package logging

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestNewLoggerRejectsUnknownLevel(t *testing.T) {
	t.Parallel()

	if _, err := NewLogger(Config{Level: "not-a-level"}); err == nil {
		t.Fatal("expected level parsing error")
	}
}

func TestWithContextAddsTraceFields(t *testing.T) {
	t.Parallel()

	provider := trace.NewTracerProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()

	tracer := provider.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	logger, err := NewLogger(Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	fields := TraceFieldsFromContext(ctx)
	if len(fields) == 0 {
		t.Fatal("expected trace fields")
	}

	enriched := WithContext(ctx, logger)
	if enriched == nil {
		t.Fatal("expected logger instance")
	}

	otel.SetTracerProvider(provider)
}
