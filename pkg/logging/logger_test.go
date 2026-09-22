package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLoggerRejectsUnknownLevel(t *testing.T) {
	t.Parallel()

	if _, err := NewLogger(Config{Level: "not-a-level"}); err == nil {
		t.Fatal("expected level parsing error")
	}
}

func TestNewLoggerRejectsUnknownOutputConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewLogger(Config{Format: "plain"}); err == nil {
		t.Fatal("expected unknown format error")
	}
	if _, err := NewLogger(Config{Format: "console", Color: "rainbow"}); err == nil {
		t.Fatal("expected unknown color error")
	}
}

func TestNormalizeOutputConfigDefaultsAndExplicitOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     Config
		wantFormat string
		wantColor  string
	}{
		{
			name:       "development defaults to console with automatic color",
			config:     Config{Environment: "development"},
			wantFormat: "console",
			wantColor:  "auto",
		},
		{
			name:       "test defaults to JSON without color",
			config:     Config{Environment: "test"},
			wantFormat: "json",
			wantColor:  "never",
		},
		{
			name:       "staging defaults to JSON without color",
			config:     Config{Environment: "staging"},
			wantFormat: "json",
			wantColor:  "never",
		},
		{
			name:       "production defaults to JSON without color",
			config:     Config{Environment: "production"},
			wantFormat: "json",
			wantColor:  "never",
		},
		{
			name:       "explicit output settings override environment defaults",
			config:     Config{Environment: "production", Format: "console", Color: "always"},
			wantFormat: "console",
			wantColor:  "always",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			if err := normalizeOutputConfig(&config); err != nil {
				t.Fatalf("normalizeOutputConfig() error = %v", err)
			}
			if config.Format != tt.wantFormat || config.Color != tt.wantColor {
				t.Fatalf("output config = format=%q color=%q, want format=%q color=%q", config.Format, config.Color, tt.wantFormat, tt.wantColor)
			}
		})
	}
}

func TestDefaultLoggerOutputFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Config
		wantJSON bool
	}{
		{name: "development writes console", config: Config{Environment: "development"}, wantJSON: false},
		{name: "production writes JSON", config: Config{Environment: "production"}, wantJSON: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			logger := newLogger(tt.config, zapcore.InfoLevel, zapcore.AddSync(&output))
			logger.Info("service started")

			var document map[string]any
			err := json.Unmarshal(output.Bytes(), &document)
			if tt.wantJSON {
				if err != nil {
					t.Fatalf("production output is not JSON: %v; output=%q", err, output.String())
				}
				return
			}
			if err == nil {
				t.Fatalf("development output unexpectedly parsed as JSON: %q", output.String())
			}
		})
	}
}

func TestShouldColorForOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		color      string
		isTerminal bool
		terminal   string
		want       bool
	}{
		{name: "always", color: "always", want: true},
		{name: "never", color: "never", isTerminal: true, terminal: "xterm", want: false},
		{name: "auto without terminal", color: "auto", terminal: "xterm", want: false},
		{name: "auto with dumb terminal", color: "auto", isTerminal: true, terminal: "dumb", want: false},
		{name: "auto with interactive terminal", color: "auto", isTerminal: true, terminal: "xterm", want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldColorForOutput(tt.color, tt.isTerminal, tt.terminal); got != tt.want {
				t.Fatalf("shouldColorForOutput() = %t, want %t", got, tt.want)
			}
		})
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

func TestConsoleLoggerColorsLevelsAndTraceIDWithoutAffectingJSONCore(t *testing.T) {
	t.Parallel()

	var consoleOutput bytes.Buffer
	var jsonOutput bytes.Buffer
	jsonCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&jsonOutput),
		zapcore.DebugLevel,
	)
	logger := newLogger(
		Config{Format: "console", Color: "always", ServiceName: "backend"},
		zapcore.DebugLevel,
		zapcore.AddSync(&consoleOutput),
		jsonCore,
	)
	logger.Info("request completed", zap.String("trace_id", "0123456789abcdef"))

	if output := consoleOutput.String(); !strings.Contains(output, "\x1b[") {
		t.Fatalf("console output does not contain ANSI colors: %q", output)
	}
	if output := jsonOutput.String(); strings.Contains(output, "\x1b[") {
		t.Fatalf("JSON core contains ANSI colors: %q", output)
	}
	var document map[string]any
	if err := json.Unmarshal(jsonOutput.Bytes(), &document); err != nil {
		t.Fatalf("JSON core output is invalid: %v; output=%s", err, jsonOutput.String())
	}
	if document["trace_id"] != "0123456789abcdef" {
		t.Fatalf("JSON trace_id = %#v, want original value", document["trace_id"])
	}
}

func TestConsoleLoggerColorsTraceIDAddedFromContext(t *testing.T) {
	t.Parallel()

	provider := trace.NewTracerProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ctx, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()
	traceID := span.SpanContext().TraceID().String()

	var output bytes.Buffer
	var jsonOutput bytes.Buffer
	jsonCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&jsonOutput),
		zapcore.DebugLevel,
	)
	logger := newLogger(
		Config{Format: "console", Color: "always"},
		zapcore.InfoLevel,
		zapcore.AddSync(&output),
		jsonCore,
	)
	enriched := WithContext(ctx, logger)
	enriched.Info("first request log")
	enriched.Info("second request log")

	wantColoredTraceID := colorizeTraceID(traceID)
	if got := strings.Count(output.String(), traceID); got != 2 {
		t.Fatalf("trace_id occurrences = %d, want 2; output=%q", got, output.String())
	}
	if got := strings.Count(output.String(), wantColoredTraceID); got != 2 {
		t.Fatalf("colored trace_id occurrences = %d, want 2; output=%q", got, output.String())
	}
	if strings.Contains(jsonOutput.String(), "\x1b[") {
		t.Fatalf("JSON output contains ANSI colors: %q", jsonOutput.String())
	}
	if got := strings.Count(jsonOutput.String(), traceID); got != 2 {
		t.Fatalf("JSON trace_id occurrences = %d, want 2; output=%q", got, jsonOutput.String())
	}
}

func TestConsoleLoggerCanDisableColor(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := newLogger(Config{Format: "console", Color: "never"}, zapcore.InfoLevel, zapcore.AddSync(&output))
	logger.Info("request completed", zap.String("trace_id", "0123456789abcdef"))

	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("color-disabled console output contains ANSI colors: %q", output.String())
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
