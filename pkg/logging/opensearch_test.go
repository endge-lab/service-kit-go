package logging

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestOpenSearchExporterSendsBulkNDJSON(t *testing.T) {
	t.Parallel()

	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/service-logs/_bulk" {
			t.Errorf("request path = %q, want /service-logs/_bulk", request.URL.Path)
		}
		username, password, ok := request.BasicAuth()
		if !ok || username != "writer" || password != "secret" {
			t.Errorf("unexpected basic auth: user=%q present=%t", username, ok)
		}
		if contentType := request.Header.Get("Content-Type"); contentType != "application/x-ndjson" {
			t.Errorf("Content-Type = %q, want application/x-ndjson", contentType)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests <- string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":false}`))
	}))
	defer server.Close()

	exporter, err := NewOpenSearchExporter(OpenSearchConfig{
		Level:          "debug",
		Endpoint:       server.URL,
		Index:          "service-logs",
		Username:       "writer",
		Password:       "secret",
		FlushInterval:  time.Hour,
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewOpenSearchExporter() error = %v", err)
	}
	defer func() { _ = exporter.Shutdown(context.Background()) }()

	logger, err := NewLogger(Config{Level: "debug", ServiceName: "backend", Environment: "test", Version: "v1"}, exporter)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	logger.With(zap.String("component", "project_usecase")).Info("project created", zap.String("project_id", "p-1"))
	if err := exporter.Sync(); err != nil {
		t.Fatalf("exporter.Sync() error = %v", err)
	}

	select {
	case body := <-requests:
		lines := strings.Split(strings.TrimSpace(body), "\n")
		if len(lines) != 2 || lines[0] != `{"index":{}}` {
			t.Fatalf("unexpected Bulk NDJSON: %q", body)
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(lines[1]), &document); err != nil {
			t.Fatalf("unmarshal document: %v", err)
		}
		if document["message"] != "project created" || document["service.name"] != "backend" || document["component"] != "project_usecase" {
			t.Fatalf("unexpected OpenSearch document: %#v", document)
		}
		if _, ok := document["@timestamp"]; !ok {
			t.Fatalf("OpenSearch document has no @timestamp: %#v", document)
		}
	case <-time.After(time.Second):
		t.Fatal("OpenSearch Bulk request was not sent")
	}
}

func TestNewOpenSearchExporterRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	if _, err := NewOpenSearchExporter(OpenSearchConfig{Index: "logs"}); err == nil {
		t.Fatal("expected empty endpoint error")
	}
	if _, err := NewOpenSearchExporter(OpenSearchConfig{Endpoint: "ftp://logs.example", Index: "logs"}); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}
