package logging

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config описывает общие поля и уровень логирования сервиса.
type Config struct {
	Level       string
	ServiceName string
	Environment string
	Version     string
}

// NewLogger создает единый JSON-логгер, пишущий в stdout.
func NewLogger(cfg Config, additionalCores ...zapcore.Core) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(normalizeLevel(cfg.Level))); err != nil {
		return nil, err
	}

	return newLogger(cfg, level, zapcore.Lock(os.Stdout), additionalCores...), nil
}

func newLogger(cfg Config, level zapcore.Level, sink zapcore.WriteSyncer, additionalCores ...zapcore.Core) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	cores := []zapcore.Core{zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
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
