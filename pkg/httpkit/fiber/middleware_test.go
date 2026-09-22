package fiber

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/endge-lab/service-kit-go/pkg/httpkit"
	gofiber "github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TestRequestIDMiddlewareUsesExistingOrGenerates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headerName string
		header     string
	}{
		{name: "default generated"},
		{name: "default existing", header: "req-1"},
		{name: "custom existing", headerName: "X-Correlation-ID", header: "corr-1"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := gofiber.New()
			app.Use(RequestIDMiddleware(tt.headerName))
			app.Get("/", func(c *gofiber.Ctx) error {
				requestID, ok := httpkit.RequestIDFromContext(c.UserContext())
				if !ok || requestID == "" {
					t.Fatalf("RequestIDFromContext() = %q/%v, want value", requestID, ok)
				}
				if local := c.Locals("request_id"); local != requestID {
					t.Fatalf("request_id local = %#v, want %q", local, requestID)
				}
				return c.SendString(requestID)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			headerName := tt.headerName
			if headerName == "" {
				headerName = gofiber.HeaderXRequestID
			}
			if tt.header != "" {
				req.Header.Set(headerName, tt.header)
			}
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if got := resp.Header.Get(headerName); got == "" {
				t.Fatal("response request id header is empty")
			} else if tt.header != "" && got != tt.header {
				t.Fatalf("response request id = %q, want %q", got, tt.header)
			}
		})
	}
}

func TestTraceRecoveryErrorAndUtilityHandlers(t *testing.T) {
	t.Parallel()

	app := gofiber.New(gofiber.Config{ErrorHandler: NewFiberErrorHandler(zap.NewNop())})
	app.Use(TraceMiddleware(trace.NewNoopTracerProvider().Tracer("test"), zap.NewNop(), "", attribute.String("component", "test")))
	app.Use(RecoveryMiddleware(zap.NewNop()))
	app.Get("/panic", func(c *gofiber.Ctx) error {
		panic("boom")
	})
	app.Get("/app-error", func(c *gofiber.Ctx) error {
		return serviceerrors.WithDetails(serviceerrors.NotFound("todo.not_found", "missing"), map[string]any{"id": "todo-1"})
	})
	app.Get("/plain-error", func(c *gofiber.Ctx) error {
		return errors.New("plain")
	})
	app.Get("/version", VersionHandler("1.2.3"))
	app.Get("/health", HealthHandler())
	app.Get("/context", func(c *gofiber.Ctx) error {
		ctx := ContextWithRequestID(context.Background(), "req-1")
		requestID, _ := httpkit.RequestIDFromContext(ctx)
		return c.SendString(requestID)
	})

	tests := []struct {
		path       string
		wantStatus int
		wantCode   string
		wantBody   map[string]string
	}{
		{path: "/panic", wantStatus: http.StatusInternalServerError, wantCode: "http.panic_recovered"},
		{path: "/app-error", wantStatus: http.StatusNotFound, wantCode: "todo.not_found"},
		{path: "/plain-error", wantStatus: http.StatusInternalServerError, wantCode: "common.internal"},
		{path: "/version", wantStatus: http.StatusOK, wantBody: map[string]string{"version": "1.2.3"}},
		{path: "/health", wantStatus: http.StatusOK, wantBody: map[string]string{"status": "ok"}},
		{path: "/context", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil), -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantCode != "" {
				var body httpkit.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if body.Code != tt.wantCode {
					t.Fatalf("code = %q, want %q; body=%#v", body.Code, tt.wantCode, body)
				}
			}
			for key, want := range tt.wantBody {
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if got := body[key]; got != want {
					t.Fatalf("body[%q] = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestTraceMiddlewareUsesFinalHTTPStatusForSpanOutcome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		handler    gofiber.Handler
		wantStatus int
		wantSpan   codes.Code
	}{
		{
			name:       "success response",
			path:       "/ok",
			handler:    func(c *gofiber.Ctx) error { return c.SendStatus(gofiber.StatusOK) },
			wantStatus: gofiber.StatusOK,
			wantSpan:   codes.Ok,
		},
		{
			name:       "client error response with nil handler error",
			path:       "/bad",
			handler:    func(c *gofiber.Ctx) error { return c.SendStatus(gofiber.StatusBadRequest) },
			wantStatus: gofiber.StatusBadRequest,
			wantSpan:   codes.Error,
		},
		{
			name:       "handler error",
			path:       "/failure",
			handler:    func(*gofiber.Ctx) error { return errors.New("handler failure") },
			wantStatus: gofiber.StatusInternalServerError,
			wantSpan:   codes.Error,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spans := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
			defer func() { _ = provider.Shutdown(context.Background()) }()

			app := gofiber.New()
			app.Use(TraceMiddleware(provider.Tracer("test"), zap.NewNop(), "handler.test"))
			app.Get(tt.path, tt.handler)

			response, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil), -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("HTTP status = %d, want %d", response.StatusCode, tt.wantStatus)
			}

			ended := spans.Ended()
			if len(ended) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(ended))
			}
			if status := ended[0].Status().Code; status != tt.wantSpan {
				t.Fatalf("span status = %s, want %s", status, tt.wantSpan)
			}
			if !hasIntAttribute(ended[0].Attributes(), "http.response.status_code", tt.wantStatus) {
				t.Fatalf("span attributes missing http.response.status_code=%d: %#v", tt.wantStatus, ended[0].Attributes())
			}
		})
	}
}

func hasIntAttribute(attrs []attribute.KeyValue, key string, value int) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsInt64() == int64(value) {
			return true
		}
	}
	return false
}

func TestRequestMetricsMiddleware(t *testing.T) {
	t.Parallel()

	handler, err := NewRequestMetricsMiddleware(noop.NewMeterProvider().Meter("test"))
	if err != nil {
		t.Fatalf("NewRequestMetricsMiddleware() error = %v, want nil", err)
	}

	app := gofiber.New()
	app.Use(handler)
	app.Get("/todos/:id", func(c *gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusAccepted)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/todos/123?token=secret", nil), -1)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != gofiber.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	t.Parallel()

	app := gofiber.New(gofiber.Config{ErrorHandler: func(c *gofiber.Ctx, err error) error {
		return c.Status(serviceerrors.HTTPStatusOf(err)).SendString(serviceerrors.CodeOf(err))
	}})
	app.Use(RequestIDMiddleware(""))
	app.Use(func(c *gofiber.Ctx) error {
		c.SetUserContext(httpkit.WithSessionID(httpkit.WithUserID(c.UserContext(), "user-1"), "session-1"))
		return c.Next()
	})
	app.Use(RequestLoggerMiddleware(zap.NewNop()))
	app.Get("/ok", func(c *gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusNoContent)
	})
	app.Get("/bad", func(c *gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusBadRequest)
	})
	app.Get("/err", func(c *gofiber.Ctx) error {
		return errors.New("plain")
	})

	tests := []struct {
		path string
		want int
	}{
		{path: "/ok?token=secret", want: gofiber.StatusNoContent},
		{path: "/bad", want: gofiber.StatusBadRequest},
		{path: "/err", want: gofiber.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(gofiber.HeaderUserAgent, "unit-test")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}
