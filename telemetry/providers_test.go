package telemetry

import (
	"context"
	"testing"

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
