package telemetry

import (
	"context"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/endge-lab/service-kit-go/pkg/logging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Step инкапсулирует span и упрощает его завершение с логированием.
type Step struct {
	span   trace.Span
	name   string
	logger *zap.Logger
}

// StartTrace создает span и сразу обогащает логгер trace-полями.
func StartTrace(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, name string, attrs ...attribute.KeyValue) (context.Context, *Step) {
	if tracer == nil {
		tracer = otel.Tracer("service-kit-go")
	}

	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	logger = logging.WithContext(ctx, logger)
	if logger != nil {
		logger.Debug("span started", zap.String("span", name))
	}

	return ctx, &Step{
		span:   span,
		name:   name,
		logger: logger,
	}
}

// End завершает span и, если есть ошибка, помечает его как failed.
func (s *Step) End(err error) {
	if s == nil || s.span == nil {
		return
	}
	defer s.span.End()

	fields := []zap.Field{zap.String("span", s.name)}
	fields = append(fields, logging.TraceFieldsFromSpan(s.span)...)

	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
		s.span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		if code := serviceerrors.CodeOf(err); code != "" {
			s.span.SetAttributes(attribute.String("error.code", code))
		}
		if s.logger != nil {
			s.logger.Error("span failed", append(fields, zap.Error(err))...)
		}
		return
	}

	s.span.SetStatus(codes.Ok, "success")
	if s.logger != nil {
		s.logger.Debug("span succeeded", fields...)
	}
}

// Fail отмечает span ошибкой, но не завершает его.
func (s *Step) Fail(err error) {
	if s == nil || s.span == nil || err == nil {
		return
	}

	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
	if code := serviceerrors.CodeOf(err); code != "" {
		s.span.SetAttributes(attribute.String("error.code", code))
	}
}

// Event добавляет событие внутрь текущего span.
func (s *Step) Event(name string, attrs ...attribute.KeyValue) {
	if s == nil || s.span == nil {
		return
	}

	s.span.AddEvent(name, trace.WithAttributes(attrs...))
	if s.logger != nil {
		fields := []zap.Field{
			zap.String("span", s.name),
			zap.String("event", name),
		}
		fields = append(fields, logging.TraceFieldsFromSpan(s.span)...)
		s.logger.Debug("span event", fields...)
	}
}
