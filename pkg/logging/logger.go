package logging

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Config описывает общие поля и уровень логирования сервиса.
type Config struct {
	Level string
	// Format controls only stdout presentation: json is machine-readable,
	// console is intended for local human-readable development logs.
	Format string
	// Color controls ANSI colors in console output: auto, always or never.
	// It has no effect on JSON or additional cores such as OpenSearch.
	Color       string
	ServiceName string
	Environment string
	Version     string
}

// NewLogger creates a stdout logger. Empty Format derives from Environment:
// development uses console output, every other environment uses JSON.
func NewLogger(cfg Config, additionalCores ...zapcore.Core) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(normalizeLevel(cfg.Level))); err != nil {
		return nil, err
	}
	if err := normalizeOutputConfig(&cfg); err != nil {
		return nil, err
	}

	return newLogger(cfg, level, zapcore.Lock(os.Stdout), additionalCores...), nil
}

func newLogger(cfg Config, level zapcore.Level, sink zapcore.WriteSyncer, additionalCores ...zapcore.Core) *zap.Logger {
	// Tests and internal callers may bypass NewLogger, so keep defaults safe here.
	_ = normalizeOutputConfig(&cfg)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	encoder := stdoutEncoder(cfg, encoderConfig)

	cores := []zapcore.Core{zapcore.NewCore(
		encoder,
		sink,
		level,
	)}
	for _, core := range additionalCores {
		if core != nil {
			cores = append(cores, core)
		}
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1)).With(
		zap.String("service.name", strings.TrimSpace(cfg.ServiceName)),
		zap.String("deployment.environment", strings.TrimSpace(cfg.Environment)),
		zap.String("service.version", strings.TrimSpace(cfg.Version)),
	)
}

func stdoutEncoder(cfg Config, encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.Format != "console" {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	if shouldColor(cfg.Color) {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return &traceColorEncoder{Encoder: zapcore.NewConsoleEncoder(encoderConfig)}
	}

	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func normalizeOutputConfig(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		if strings.EqualFold(strings.TrimSpace(cfg.Environment), "development") {
			format = "console"
		} else {
			format = "json"
		}
	}
	if format != "json" && format != "console" {
		return fmt.Errorf("unknown logger format %q", cfg.Format)
	}

	color := strings.ToLower(strings.TrimSpace(cfg.Color))
	if color == "" {
		if format == "console" {
			color = "auto"
		} else {
			color = "never"
		}
	}
	if color != "auto" && color != "always" && color != "never" {
		return fmt.Errorf("unknown logger color mode %q", cfg.Color)
	}

	cfg.Format = format
	cfg.Color = color
	return nil
}

func shouldColor(color string) bool {
	info, err := os.Stdout.Stat()
	isTerminal := err == nil && info.Mode()&os.ModeCharDevice != 0
	return shouldColorForOutput(color, isTerminal, os.Getenv("TERM"))
}

func shouldColorForOutput(color string, isTerminal bool, terminal string) bool {
	switch color {
	case "always":
		return true
	case "auto":
		return isTerminal && terminal != "dumb"
	default:
		return false
	}
}

// traceColorEncoder colors trace_id values deterministically in console mode.
// It never wraps a JSON encoder, keeping structured outputs free of ANSI bytes.
type traceColorEncoder struct {
	zapcore.Encoder
}

func (e *traceColorEncoder) Clone() zapcore.Encoder {
	return &traceColorEncoder{Encoder: e.Encoder.Clone()}
}

func (e *traceColorEncoder) AddString(key, value string) {
	if key == "trace_id" && value != "" {
		value = colorizeTraceID(value)
	}
	e.Encoder.AddString(key, value)
}

func (e *traceColorEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	coloredFields := append([]zapcore.Field(nil), fields...)
	for index := range coloredFields {
		if coloredFields[index].Key == "trace_id" && coloredFields[index].Type == zapcore.StringType && coloredFields[index].String != "" {
			coloredFields[index].String = colorizeTraceID(coloredFields[index].String)
		}
	}

	encoded, err := e.Encoder.EncodeEntry(entry, coloredFields)
	if err != nil {
		return nil, err
	}

	// ConsoleEncoder writes structured fields as JSON and escapes ANSI as
	// "\\u001b". Restore it only for a trace_id value so terminals can render
	// the color without allowing arbitrary log fields to inject control bytes.
	output := unescapeTraceIDColor(encoded.String())
	encoded.Reset()
	encoded.AppendString(output)
	return encoded, nil
}

func unescapeTraceIDColor(output string) string {
	const fieldPrefix = `"trace_id": "\u001b[` //nolint:gosmopolitan // ANSI is intentional for local console output.
	const fieldValuePrefix = `"trace_id": "`
	const reset = `\u001b[0m`

	var result strings.Builder
	position := 0
	for {
		match := strings.Index(output[position:], fieldPrefix)
		if match < 0 {
			result.WriteString(output[position:])
			return result.String()
		}

		start := position + match
		valueStart := start + len(fieldValuePrefix)
		resetMatch := strings.Index(output[valueStart:], reset)
		if resetMatch < 0 {
			result.WriteString(output[position:])
			return result.String()
		}

		valueEnd := valueStart + resetMatch + len(reset)
		result.WriteString(output[position:valueStart])
		result.WriteString(strings.ReplaceAll(output[valueStart:valueEnd], `\u001b`, "\x1b"))
		position = valueEnd
	}
}

func colorizeTraceID(traceID string) string {
	colors := [...]int{31, 32, 33, 34, 35, 36}
	var hash uint32
	for _, symbol := range traceID {
		hash = hash*31 + uint32(symbol)
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", colors[int(hash)%len(colors)], traceID)
}

// WithComponent добавляет стабильное поле component.
func WithComponent(logger *zap.Logger, component string) *zap.Logger {
	if logger == nil {
		return nil
	}

	return logger.With(zap.String("component", strings.TrimSpace(component)))
}

// WithContext добавляет trace/request поля из context, если они есть.
func WithContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		return nil
	}

	fields := TraceFieldsFromContext(ctx)
	if len(fields) == 0 {
		return logger
	}

	return logger.With(fields...)
}

// TraceFieldsFromContext извлекает trace/span поля из контекста.
func TraceFieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	return TraceFieldsFromSpan(trace.SpanFromContext(ctx))
}

// TraceFieldsFromSpan строит поля логирования из span context.
func TraceFieldsFromSpan(span trace.Span) []zap.Field {
	if span == nil {
		return nil
	}

	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return nil
	}

	fields := []zap.Field{
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	}
	if spanContext.TraceFlags().IsSampled() {
		fields = append(fields, zap.Bool("trace_sampled", true))
	}

	return fields
}

func normalizeLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return "info"
	case "debug":
		return "debug"
	case "warn", "warning":
		return "warn"
	case "error":
		return "error"
	default:
		return strings.ToLower(strings.TrimSpace(level))
	}
}
