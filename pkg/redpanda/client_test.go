package redpanda

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestNewMessageInjectsTraceHeaders(t *testing.T) {
	t.Parallel()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	provider := trace.NewTracerProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ctx, span := provider.Tracer("test").Start(context.Background(), "publish")
	defer span.End()

	client := NewClient(Config{Enabled: true}, nil, propagation.TraceContext{})
	message := client.NewMessage(ctx, []byte("key"), []byte("value"), map[string]string{"x-event": "user.created"})

	if len(message.Headers) == 0 {
		t.Fatal("expected headers in message")
	}
}
