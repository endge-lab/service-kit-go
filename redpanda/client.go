package redpanda

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/endge-lab/service-kit-go/logging"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

var ErrDisabled = errors.New("redpanda is disabled")

// Config описывает минимальные настройки клиента Redpanda/Kafka.
type Config struct {
	Enabled          bool
	Brokers          []string
	ClientID         string
	DialTimeout      time.Duration
	ReadBatchTimeout time.Duration
	WriteTimeout     time.Duration
}

// Client создает reader/writer и умеет переносить trace headers в сообщения.
type Client struct {
	cfg        Config
	logger     *zap.Logger
	propagator propagation.TextMapPropagator
}

// NewClient создает обертку над kafka-go с единым логированием и propagator.
func NewClient(cfg Config, logger *zap.Logger, propagator propagation.TextMapPropagator) *Client {
	componentLogger := logging.WithComponent(logger, "redpanda")
	if componentLogger != nil {
		if cfg.Enabled {
			componentLogger.Info("redpanda client configured",
				zap.Strings("brokers", normalizeStrings(cfg.Brokers)),
				zap.String("client_id", strings.TrimSpace(cfg.ClientID)),
			)
		} else {
			componentLogger.Info("redpanda client is disabled")
		}
	}

	if propagator == nil {
		propagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	return &Client{
		cfg: Config{
			Enabled:          cfg.Enabled,
			Brokers:          normalizeStrings(cfg.Brokers),
			ClientID:         strings.TrimSpace(cfg.ClientID),
			DialTimeout:      cfg.DialTimeout,
			ReadBatchTimeout: cfg.ReadBatchTimeout,
			WriteTimeout:     cfg.WriteTimeout,
		},
		logger:     componentLogger,
		propagator: propagator,
	}
}

// Enabled показывает, включен ли Redpanda runtime.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled
}

// NewReader создает kafka reader.
func (c *Client) NewReader(topic string, groupID string) (*kafka.Reader, error) {
	if !c.Enabled() {
		return nil, ErrDisabled
	}

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:         c.cfg.Brokers,
		GroupID:         strings.TrimSpace(groupID),
		Topic:           strings.TrimSpace(topic),
		Dialer:          c.newDialer(),
		MaxWait:         fallbackDuration(c.cfg.ReadBatchTimeout, time.Second),
		ReadLagInterval: -1,
		Logger:          newKafkaLogger(c.logger),
		ErrorLogger:     newKafkaLogger(logging.WithComponent(c.logger, "stream_error")),
	}), nil
}

// NewWriter создает kafka writer.
func (c *Client) NewWriter(topic string) (*kafka.Writer, error) {
	if !c.Enabled() {
		return nil, ErrDisabled
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(c.cfg.Brokers...),
		Topic:        strings.TrimSpace(topic),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: fallbackDuration(c.cfg.ReadBatchTimeout, time.Second),
		WriteTimeout: fallbackDuration(c.cfg.WriteTimeout, 5*time.Second),
		Transport: &kafka.Transport{
			ClientID: c.cfg.ClientID,
		},
		Logger:      newKafkaLogger(c.logger),
		ErrorLogger: newKafkaLogger(logging.WithComponent(c.logger, "stream_error")),
	}, nil
}

// NewMessage собирает сообщение и внедряет в него trace headers.
func (c *Client) NewMessage(ctx context.Context, key []byte, value []byte, headers map[string]string) kafka.Message {
	carrier := headerCarrierFromMap(headers)
	if c != nil && c.propagator != nil {
		c.propagator.Inject(ctx, carrier)
	}

	messageHeaders := make([]kafka.Header, 0, len(carrier))
	for headerKey, headerValue := range carrier {
		messageHeaders = append(messageHeaders, kafka.Header{Key: headerKey, Value: []byte(headerValue)})
	}

	return kafka.Message{
		Key:     key,
		Value:   value,
		Headers: messageHeaders,
		Time:    time.Now().UTC(),
	}
}

// ExtractContext поднимает trace context из kafka headers.
func (c *Client) ExtractContext(ctx context.Context, message kafka.Message) context.Context {
	carrier := kafkaHeaderCarrier(message.Headers)
	if c == nil || c.propagator == nil {
		return ctx
	}

	return c.propagator.Extract(ctx, carrier)
}

func (c *Client) newDialer() *kafka.Dialer {
	return &kafka.Dialer{
		ClientID: c.cfg.ClientID,
		Timeout:  fallbackDuration(c.cfg.DialTimeout, 5*time.Second),
	}
}

type kafkaLogger struct {
	logger *zap.Logger
}

func newKafkaLogger(logger *zap.Logger) kafka.Logger {
	return kafkaLogger{logger: logger}
}

func (l kafkaLogger) Printf(format string, args ...any) {
	if l.logger == nil {
		return
	}

	l.logger.Sugar().Infow("kafka-go", "message", format, "args", args)
}

type kafkaHeaderCarrier []kafka.Header

func (c kafkaHeaderCarrier) Get(key string) string {
	for _, header := range c {
		if strings.EqualFold(header.Key, key) {
			return string(header.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key string, value string) {}

func (c kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for _, header := range c {
		keys = append(keys, header.Key)
	}
	return keys
}

type mapCarrier map[string]string

func headerCarrierFromMap(headers map[string]string) mapCarrier {
	carrier := make(mapCarrier, len(headers))
	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		carrier[trimmedKey] = strings.TrimSpace(value)
	}
	return carrier
}

func (c mapCarrier) Get(key string) string {
	return c[key]
}

func (c mapCarrier) Set(key string, value string) {
	c[key] = value
}

func (c mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

func normalizeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func fallbackDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}
