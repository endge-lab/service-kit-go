package redpanda

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestNewMessageInjectsTraceHeaders(t *testing.T) {
	t.Parallel()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	provider := sdktrace.NewTracerProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ctx, span := provider.Tracer("test").Start(context.Background(), "publish")
	defer span.End()

	client := NewClient(Config{Enabled: true}, nil, propagation.TraceContext{})
	message := client.NewMessage(ctx, []byte("key"), []byte("value"), map[string]string{"x-event": "user.created"})

	if len(message.Headers) == 0 {
		t.Fatal("expected headers in message")
	}
}

func TestClientEnabledReaderWriterAndDisabled(t *testing.T) {
	t.Parallel()

	if (*Client)(nil).Enabled() {
		t.Fatal("nil client Enabled() = true, want false")
	}

	disabled := NewClient(Config{Enabled: false}, nil, nil)
	if disabled.Enabled() {
		t.Fatal("disabled client Enabled() = true, want false")
	}
	if reader, err := disabled.NewReader("topic", "group"); !errors.Is(err, ErrDisabled) || reader != nil {
		t.Fatalf("disabled NewReader() reader=%#v err=%v, want ErrDisabled", reader, err)
	}
	if writer, err := disabled.NewWriter("topic"); !errors.Is(err, ErrDisabled) || writer != nil {
		t.Fatalf("disabled NewWriter() writer=%#v err=%v, want ErrDisabled", writer, err)
	}

	client := NewClient(Config{
		Enabled:          true,
		Brokers:          []string{" broker-1:9092 ", "", "broker-1:9092", "broker-2:9092"},
		ClientID:         " client ",
		DialTimeout:      2 * time.Second,
		ReadBatchTimeout: 3 * time.Second,
		WriteTimeout:     4 * time.Second,
	}, nil, nil)
	if !client.Enabled() {
		t.Fatal("enabled client Enabled() = false, want true")
	}
	reader, err := client.NewReader(" topic ", " group ")
	if err != nil {
		t.Fatalf("NewReader() error = %v, want nil", err)
	}
	reader.Close()

	writer, err := client.NewWriter(" topic ")
	if err != nil {
		t.Fatalf("NewWriter() error = %v, want nil", err)
	}
	if writer.Topic != "topic" {
		t.Fatalf("writer.Topic = %q, want topic", writer.Topic)
	}
	if writer.BatchTimeout != 3*time.Second || writer.WriteTimeout != 4*time.Second {
		t.Fatalf("writer timeouts = %s/%s, want 3s/4s", writer.BatchTimeout, writer.WriteTimeout)
	}
	if writer.Transport == nil {
		t.Fatal("writer.Transport = nil, want kafka transport")
	}
	_ = writer.Close()

	defaults := NewClient(Config{Enabled: true, Brokers: []string{"broker:9092"}}, nil, nil)
	defaultWriter, err := defaults.NewWriter("events")
	if err != nil {
		t.Fatalf("default NewWriter() error = %v, want nil", err)
	}
	if defaultWriter.BatchTimeout != time.Second || defaultWriter.WriteTimeout != 5*time.Second {
		t.Fatalf("default writer timeouts = %s/%s, want 1s/5s", defaultWriter.BatchTimeout, defaultWriter.WriteTimeout)
	}
	if defaults.newDialer().Timeout != 5*time.Second {
		t.Fatalf("default dial timeout = %s, want 5s", defaults.newDialer().Timeout)
	}
	_ = defaultWriter.Close()
}

func TestNewMessageSanitizesHeadersAndExtractContext(t *testing.T) {
	t.Parallel()

	client := NewClient(Config{Enabled: true}, nil, propagation.TraceContext{})
	message := client.NewMessage(context.Background(), []byte("key"), []byte("value"), map[string]string{
		" x-event ": " created ",
		"":          "ignored",
		"   ":       "ignored",
	})

	if string(message.Key) != "key" || string(message.Value) != "value" {
		t.Fatalf("message key/value = %q/%q", message.Key, message.Value)
	}
	if message.Time.IsZero() {
		t.Fatal("message.Time is zero")
	}
	if got := headerValue(message.Headers, "x-event"); got != "created" {
		t.Fatalf("x-event header = %q, want created; headers=%#v", got, message.Headers)
	}
	if got := headerValue(message.Headers, ""); got != "" {
		t.Fatalf("empty header = %q, want absent", got)
	}

	ctx := client.ExtractContext(context.Background(), kafka.Message{Headers: []kafka.Header{
		{Key: "Traceparent", Value: []byte("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")},
	}})
	spanContext := oteltrace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() || !spanContext.TraceFlags().IsSampled() {
		t.Fatalf("extracted span context invalid: %#v", spanContext)
	}
	if same := (*Client)(nil).ExtractContext(ctx, kafka.Message{}); same != ctx {
		t.Fatal("nil client ExtractContext() should return original context")
	}
}

func TestKafkaLoggerAndHeaderCarriers(t *testing.T) {
	t.Parallel()

	newKafkaLogger(nil).Printf("ignored %s", "message")

	headers := kafkaHeaderCarrier([]kafka.Header{{Key: "Traceparent", Value: []byte("value")}})
	if got := headers.Get("traceparent"); got != "value" {
		t.Fatalf("case-insensitive Get() = %q, want value", got)
	}
	headers.Set("new", "ignored")
	if got := headers.Get("new"); got != "" {
		t.Fatalf("read-only kafka carrier Get(new) = %q, want empty", got)
	}
	if keys := headers.Keys(); len(keys) != 1 || keys[0] != "Traceparent" {
		t.Fatalf("Keys() = %#v, want Traceparent", keys)
	}

	carrier := headerCarrierFromMap(nil)
	carrier.Set("key", "value")
	if carrier.Get("key") != "value" {
		t.Fatalf("map carrier Get() = %q, want value", carrier.Get("key"))
	}
	if keys := carrier.Keys(); len(keys) != 1 || keys[0] != "key" {
		t.Fatalf("map carrier Keys() = %#v, want key", keys)
	}
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}
