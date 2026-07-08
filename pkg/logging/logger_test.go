package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
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

func TestNewLoggerNormalizesLevelsAndFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
		log   func(*testing.T, *bytes.Buffer)
	}{
		{
			name:  "empty defaults to info",
			level: "",
			log: func(t *testing.T, buf *bytes.Buffer) {
				logger := newLogger(Config{ServiceName: " svc ", Environment: " test ", Version: " v1 "}, zapcore.InfoLevel, zapcore.AddSync(buf))
				logger.Debug("hidden")
				logger.Info("visible")
			},
		},
		{
			name:  "warning maps to warn",
			level: "warning",
			log: func(t *testing.T, buf *bytes.Buffer) {
				level := zapcore.InfoLevel
				if err := level.UnmarshalText([]byte(normalizeLevel("warning"))); err != nil {
					t.Fatalf("level parse: %v", err)
				}
				logger := newLogger(Config{}, level, zapcore.AddSync(buf))
				logger.Info("info")
				logger.Warn("warn")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			tt.log(t, &buf)
			output := buf.String()
			if strings.Contains(output, "hidden") {
				t.Fatalf("debug log was emitted at info level: %s", output)
			}
			if !strings.Contains(output, "service.name") {
				t.Fatalf("logger output missing service fields: %s", output)
			}
		})
	}
}

func TestNilLoggerHelpers(t *testing.T) {
	t.Parallel()

	if WithComponent(nil, "component") != nil {
		t.Fatal("WithComponent(nil) != nil")
	}
	if WithContext(context.Background(), nil) != nil {
		t.Fatal("WithContext(nil logger) != nil")
	}
	if fields := TraceFieldsFromContext(nil); fields != nil {
		t.Fatalf("TraceFieldsFromContext(nil) = %#v, want nil", fields)
	}
	if fields := TraceFieldsFromSpan(nil); fields != nil {
		t.Fatalf("TraceFieldsFromSpan(nil) = %#v, want nil", fields)
	}
}
