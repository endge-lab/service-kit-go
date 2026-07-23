package logging

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultOpenSearchFlushInterval  = time.Second
	defaultOpenSearchBatchSize      = 100
	defaultOpenSearchQueueSize      = 1000
	defaultOpenSearchRequestTimeout = 5 * time.Second
)

// OpenSearchConfig настраивает прямую выгрузку Zap-событий в OpenSearch Bulk API.
// Endpoint должен быть базовым HTTP(S) URL OpenSearch, например https://search.example.
type OpenSearchConfig struct {
	Level              string
	Endpoint           string
	Index              string
	Username           string
	Password           string
	InsecureSkipVerify bool
	FlushInterval      time.Duration
	BatchSize          int
	QueueSize          int
	RequestTimeout     time.Duration
}

// OpenSearchExporter одновременно является zapcore.Core и владельцем фоновой
// пакетной отправки. Неуспешные записи отбрасываются: exporter не должен
// блокировать прикладной код и не должен логировать через Zap рекурсивно.
type OpenSearchExporter struct {
	level     zapcore.LevelEnabler
	encoder   zapcore.Encoder
	fields    []zap.Field
	entries   chan []byte
	commands  chan openSearchCommand
	client    *http.Client
	bulkURL   string
	username  string
	password  string
	timeout   time.Duration
	batchSize int
	stopped   *atomic.Bool
	stopOnce  *sync.Once
}

type openSearchCommand struct {
	ctx      context.Context
	shutdown bool
	result   chan error
}

// NewOpenSearchExporter создает отдельный log exporter. Он не меняет global
// OpenTelemetry providers и может быть добавлен вторым core к обычному Zap logger.
func NewOpenSearchExporter(cfg OpenSearchConfig) (*OpenSearchExporter, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("opensearch endpoint is required")
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("opensearch endpoint must be an absolute HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("opensearch endpoint must use http or https")
	}

	index := strings.TrimSpace(cfg.Index)
	if index == "" {
		return nil, fmt.Errorf("opensearch index is required")
	}

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(normalizeLevel(cfg.Level))); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = defaultOpenSearchFlushInterval
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultOpenSearchBatchSize
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultOpenSearchQueueSize
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultOpenSearchRequestTimeout
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		// #nosec G402 -- this opt-in is intended only for a trusted development endpoint.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	exporter := &OpenSearchExporter{
		level:     level,
		encoder:   newOpenSearchEncoder(),
		entries:   make(chan []byte, cfg.QueueSize),
		commands:  make(chan openSearchCommand),
		client:    &http.Client{Transport: transport},
		bulkURL:   endpoint + "/" + url.PathEscape(index) + "/_bulk",
		username:  strings.TrimSpace(cfg.Username),
		password:  cfg.Password,
		timeout:   cfg.RequestTimeout,
		batchSize: cfg.BatchSize,
		stopped:   &atomic.Bool{},
		stopOnce:  &sync.Once{},
	}

	go exporter.run(cfg.FlushInterval)
	return exporter, nil
}

func newOpenSearchEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "log.level",
		MessageKey:     "message",
		CallerKey:      "log.caller",
		StacktraceKey:  "error.stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
}

// With реализует zapcore.Core и сохраняет поля дочернего logger-а.
func (e *OpenSearchExporter) With(fields []zap.Field) zapcore.Core {
	if e == nil {
		return nil
	}

	clone := *e
	clone.fields = append(append([]zap.Field(nil), e.fields...), fields...)
	return &clone
}

func (e *OpenSearchExporter) Enabled(level zapcore.Level) bool {
	return e != nil && !e.stopped.Load() && e.level.Enabled(level)
}

func (e *OpenSearchExporter) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if e.Enabled(entry.Level) {
		return checked.AddCore(entry, e)
	}

	return checked
}

// Write сериализует запись и неблокирующе передает ее фоновой очереди.
func (e *OpenSearchExporter) Write(entry zapcore.Entry, fields []zap.Field) error {
	if e == nil || e.stopped.Load() {
		return nil
	}

	allFields := make([]zap.Field, 0, len(e.fields)+len(fields))
	allFields = append(allFields, e.fields...)
	allFields = append(allFields, fields...)

	buffer, err := e.encoder.Clone().EncodeEntry(entry, allFields)
	if err != nil {
		return err
	}
	document := append([]byte(nil), buffer.Bytes()...)
	buffer.Free()

	select {
	case e.entries <- document:
	default:
		// Очередь ограничена намеренно: недоступный OpenSearch не может
		// задерживать обработку запросов или неограниченно расходовать память.
	}
	return nil
}

// Sync принудительно отправляет все записи, принятые exporter-ом до вызова.
func (e *OpenSearchExporter) Sync() error {
	if e == nil || e.stopped.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	return e.command(ctx, false)
}

// Shutdown сначала отправляет накопленные записи, затем останавливает worker.
func (e *OpenSearchExporter) Shutdown(ctx context.Context) error {
	if e == nil {
		return nil
	}

	var shutdownErr error
	e.stopOnce.Do(func() {
		e.stopped.Store(true)
		shutdownErr = e.command(ctx, true)
	})
	return shutdownErr
}

func (e *OpenSearchExporter) command(ctx context.Context, shutdown bool) error {
	result := make(chan error, 1)
	command := openSearchCommand{ctx: ctx, shutdown: shutdown, result: result}
	select {
	case e.commands <- command:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *OpenSearchExporter) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([][]byte, 0, e.batchSize)
	flush := func(ctx context.Context) error {
		if len(batch) == 0 {
			return nil
		}
		current := batch
		batch = make([][]byte, 0, e.batchSize)
		return e.send(ctx, current)
	}

	for {
		select {
		case document := <-e.entries:
			batch = append(batch, document)
			if len(batch) >= e.batchSize {
				flushCtx, cancel := context.WithTimeout(context.Background(), e.timeout)
				_ = flush(flushCtx)
				cancel()
			}
		case command := <-e.commands:
			for {
				select {
				case document := <-e.entries:
					batch = append(batch, document)
				default:
					goto drained
				}
			}
		drained:
			err := flush(command.ctx)
			command.result <- err
			if command.shutdown {
				return
			}
		case <-ticker.C:
			flushCtx, cancel := context.WithTimeout(context.Background(), e.timeout)
			_ = flush(flushCtx)
			cancel()
		}
	}
}

func (e *OpenSearchExporter) send(ctx context.Context, documents [][]byte) error {
	var payload bytes.Buffer
	for _, document := range documents {
		_, _ = payload.WriteString("{\"index\":{}}\n")
		_, _ = payload.Write(document)
		if len(document) == 0 || document[len(document)-1] != '\n' {
			_ = payload.WriteByte('\n')
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.bulkURL, &payload)
	if err != nil {
		return fmt.Errorf("create opensearch bulk request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if e.username != "" {
		req.SetBasicAuth(e.username, e.password)
	}

	response, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("send opensearch bulk request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("opensearch bulk request returned status %d", response.StatusCode)
	}

	var result struct {
		Errors bool `json:"errors"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode opensearch bulk response: %w", err)
	}
	if result.Errors {
		return fmt.Errorf("opensearch rejected one or more log documents")
	}

	return nil
}
