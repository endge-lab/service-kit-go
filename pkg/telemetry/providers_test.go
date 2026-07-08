package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
)

func TestNewProvidersWithoutEndpoint(t *testing.T) {
	t.Parallel()

	providers, err := NewProviders(context.Background(), Config{
		ServiceName:    "svc",
		ServiceVersion: "0.1.0",
		Environment:    "test",
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("new providers: %v", err)
	}

	if providers.Resource() == nil {
		t.Fatal("expected resource")
	}
	if providers.Tracer("svc") == nil {
		t.Fatal("expected tracer")
	}
	if providers.Meter("svc") == nil {
		t.Fatal("expected meter")
	}
	if err := providers.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestNilProvidersReturnFallbacks(t *testing.T) {
	t.Parallel()

	var providers *Providers
	if providers.Resource() != nil {
		t.Fatal("nil Resource() != nil")
	}
	if providers.Propagator() == nil {
		t.Fatal("nil Propagator() = nil")
	}
	if providers.Tracer(" test ") == nil {
		t.Fatal("nil Tracer() = nil")
	}
	if providers.Meter(" test ") == nil {
		t.Fatal("nil Meter() = nil")
	}
	if err := providers.Shutdown(context.Background()); err != nil {
		t.Fatalf("nil Shutdown() error = %v, want nil", err)
	}
}

func TestStepRecordsSuccessErrorFailAndEvent(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ctx, step := StartTrace(context.Background(), provider.Tracer("test"), zap.NewNop(), "operation", attribute.String("component", "unit"))
	step.Event("checkpoint", attribute.String("phase", "middle"))
	step.End(nil)

	failedErr := serviceerrors.InvalidInput("request.invalid", "bad request")
	_, failedStep := StartTrace(ctx, provider.Tracer("test"), zap.NewNop(), "failed")
	failedStep.Fail(failedErr)
	failedStep.End(failedErr)

	ended := recorder.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(ended))
	}
	if ended[0].Name() != "operation" {
		t.Fatalf("first span = %q, want operation", ended[0].Name())
	}
	if len(ended[0].Events()) != 1 || ended[0].Events()[0].Name != "checkpoint" {
		t.Fatalf("success span events = %#v, want checkpoint", ended[0].Events())
	}
	if ended[1].Name() != "failed" {
		t.Fatalf("second span = %q, want failed", ended[1].Name())
	}
	attrs := ended[1].Attributes()
	if !hasAttribute(attrs, "error.code", "request.invalid") {
		t.Fatalf("failed span attributes missing error.code: %#v", attrs)
	}
}

func TestStepNilAndNoopMethods(t *testing.T) {
	t.Parallel()

	var step *Step
	step.End(errors.New("ignored"))
	step.Fail(errors.New("ignored"))
	step.Event("ignored")

	_, realStep := StartTrace(context.Background(), nil, nil, "noop")
	realStep.Fail(nil)
	realStep.Event("event")
	realStep.End(nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := (&Providers{}).Shutdown(ctx); err != nil {
		t.Fatalf("empty providers Shutdown() error = %v, want nil", err)
	}
}

func hasAttribute(attrs []attribute.KeyValue, key string, value string) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return true
		}
	}
	return false
}
