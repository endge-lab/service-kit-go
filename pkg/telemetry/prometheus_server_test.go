package telemetry

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestPrometheusServerServesOnlyConfiguredPath(t *testing.T) {
	t.Parallel()

	server, err := NewPrometheusServer(
		PrometheusServerConfig{BindAddress: "127.0.0.1:0", HandlerPath: "/internal/metrics"},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "metric_value 1\n")
		}),
		zap.NewNop(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	response, err := http.Get("http://" + server.Address() + "/internal/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	missing, err := http.Get("http://" + server.Address() + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing path status = %d, want %d", missing.StatusCode, http.StatusNotFound)
	}
}
