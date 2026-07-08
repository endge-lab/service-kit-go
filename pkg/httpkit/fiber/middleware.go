package fiber

import (
	"context"
	"fmt"
	"strconv"
	"time"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/endge-lab/service-kit-go/pkg/httpkit"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RequestIDMiddleware добавляет request id в context, locals и response headers.
func RequestIDMiddleware(headerName string) fiber.Handler {
	name := headerName
	if name == "" {
		name = fiber.HeaderXRequestID
	}

	return func(c *fiber.Ctx) error {
		requestID := c.Get(name)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := httpkit.WithRequestID(c.UserContext(), requestID)
		c.SetUserContext(ctx)
		c.Locals("request_id", requestID)
		c.Set(name, requestID)

		return c.Next()
	}
}

// TraceMiddleware поднимает span на время HTTP-запроса.
func TraceMiddleware(tracer trace.Tracer, logger *zap.Logger, name string, attrs ...attribute.KeyValue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceName := name
		if traceName == "" {
			traceName = c.Method() + " " + c.Path()
		}

		ctx, step := telemetry.StartTrace(c.UserContext(), tracer, logger, traceName, attrs...)
		c.SetUserContext(ctx)

		err := c.Next()
		step.End(err)
		return err
	}
}

// RequestLoggerMiddleware пишет единый structured request log.
func RequestLoggerMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()

		ctx := c.UserContext()
		entry := logging.WithContext(ctx, logger)
		statusCode := c.Response().StatusCode()
		if statusCode == 0 && err != nil {
			statusCode = fiber.StatusInternalServerError
		}

		fields := []zap.Field{
			zap.String("http.method", c.Method()),
			zap.String("http.path", c.Path()),
			zap.String("http.route", routePath(c)),
			zap.String("http.url", httpkit.SanitizeURL(c.OriginalURL())),
			zap.Int("http.status_code", statusCode),
			zap.Duration("http.duration", time.Since(startedAt)),
			zap.String("http.ip", c.IP()),
			zap.String("http.user_agent", c.Get(fiber.HeaderUserAgent)),
		}

		if requestID, ok := httpkit.RequestIDFromContext(ctx); ok && requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}
		if userID, ok := httpkit.UserIDFromContext(ctx); ok && userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}
		if sessionID, ok := httpkit.SessionIDFromContext(ctx); ok && sessionID != "" {
			fields = append(fields, zap.String("session_id", sessionID))
		}

		switch {
		case err != nil || statusCode >= fiber.StatusInternalServerError:
			entry.Error("http request failed", append(fields, zap.Error(err))...)
		case statusCode >= fiber.StatusBadRequest:
			entry.Warn("http request completed with client error", fields...)
		default:
			entry.Info("http request completed", fields...)
		}

		return err
	}
}

type requestMetrics struct {
	requestsTotal    metric.Int64Counter
	requestDuration  metric.Float64Histogram
	inflightRequests metric.Int64UpDownCounter
}

// NewRequestMetricsMiddleware регистрирует базовые HTTP-метрики.
func NewRequestMetricsMiddleware(meter metric.Meter) (fiber.Handler, error) {
	requestsTotal, err := meter.Int64Counter(
		"servicekit.http.server.requests_total",
		metric.WithDescription("Общее число HTTP-запросов"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"servicekit.http.server.request_duration_ms",
		metric.WithDescription("Время обработки HTTP-запроса"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	inflightRequests, err := meter.Int64UpDownCounter(
		"servicekit.http.server.inflight_requests",
		metric.WithDescription("Текущее число активных HTTP-запросов"),
	)
	if err != nil {
		return nil, err
	}

	mw := &requestMetrics{
		requestsTotal:    requestsTotal,
		requestDuration:  requestDuration,
		inflightRequests: inflightRequests,
	}
	return mw.Handler(), nil
}

// Handler возвращает готовый metrics middleware.
func (m *requestMetrics) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		startedAt := time.Now()

		m.inflightRequests.Add(c.UserContext(), 1)
		defer m.inflightRequests.Add(c.UserContext(), -1)

		err := c.Next()
		ctx := c.UserContext()
		statusCode := c.Response().StatusCode()
		if statusCode == 0 && err != nil {
			statusCode = fiber.StatusInternalServerError
		}

		route := routePath(c)
		if route == "" {
			route = c.Path()
		}

		attrs := metric.WithAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.route", route),
			attribute.String("http.status_code", strconv.Itoa(statusCode)),
		)

		m.requestsTotal.Add(ctx, 1, attrs)
		m.requestDuration.Record(ctx, float64(time.Since(startedAt).Milliseconds()), attrs)

		return err
	}
}

// RecoveryMiddleware перехватывает panic и переводит его в app error.
func RecoveryMiddleware(logger *zap.Logger) fiber.Handler {
	componentLogger := logging.WithComponent(logger, "fiber_recovery")

	return func(c *fiber.Ctx) (err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			panicErr := fmt.Errorf("panic recovered: %v", recovered)
			if componentLogger != nil {
				componentLogger.Error("http panic recovered", zap.Any("panic", recovered))
			}
			err = serviceerrors.Wrap(panicErr, "http.panic_recovered", "Внутренняя ошибка сервера", 500)
		}()

		return c.Next()
	}
}

// NewFiberErrorHandler маппит app errors в единый JSON-ответ.
func NewFiberErrorHandler(logger *zap.Logger) fiber.ErrorHandler {
	componentLogger := logging.WithComponent(logger, "fiber_error_handler")

	return func(c *fiber.Ctx, err error) error {
		if err == nil {
			return nil
		}

		ctx := c.UserContext()
		entry := logging.WithContext(ctx, componentLogger)
		if entry != nil && serviceerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
			entry.Error("http handler returned error", zap.Error(err))
		}

		return c.Status(serviceerrors.HTTPStatusOf(err)).JSON(httpkit.ErrorResponse{
			Code:    serviceerrors.CodeOf(err),
			Message: serviceerrors.SafeMessageOf(err),
			Details: serviceerrors.DetailsOf(err),
		})
	}
}

// VersionHandler отдает текущую runtime-версию сервиса.
func VersionHandler(version string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": version})
	}
}

// HealthHandler отдает минимальный технический health response.
func HealthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	}
}

func routePath(c *fiber.Ctx) string {
	if route := c.Route(); route != nil {
		return route.Path
	}

	return ""
}

// ContextWithRequestID возвращает контекст с request id. Нужен для фоновых сценариев вне Fiber.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return httpkit.WithRequestID(ctx, requestID)
}
