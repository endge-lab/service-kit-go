# Logging

`logging` дает единый `zap`-логгер для Go-сервисов Endge.

## Что входит

- `Config` с полями `service.name`, `deployment.environment`, `service.version`
- `NewLogger` для стандартного JSON-логирования в `stdout`
- `WithComponent` для стабильного поля `component`
- `WithContext` и trace helpers для автоматического добавления `trace_id` и `span_id`

## Когда использовать

- при инициализации runtime сервиса
- внутри middleware и background worker-ов
- в связке с `telemetry.StartTrace`, чтобы логи автоматически получали trace-поля

## Пример

```go
logger, err := logging.NewLogger(logging.Config{
    Level:       "info",
    ServiceName: "service-core",
    Environment: "production",
    Version:     "1.2.3",
})
if err != nil {
    return err
}

logger = logging.WithComponent(logger, "bootstrap")
logger.Info("service started")
```

## OpenSearch exporter

`OpenSearchExporter` — опциональный второй Zap core. JSON-логи продолжают
писаться в stdout, а exporter асинхронно группирует записи и отправляет их в
OpenSearch Bulk API. Недоступность OpenSearch не блокирует обработку запросов:
очередь ограничена, а неотправленные записи отбрасываются.

```go
import (
    "context"
    "time"
)

exporter, err := logging.NewOpenSearchExporter(logging.OpenSearchConfig{
    Level:          "info",
    Endpoint:       "https://opensearch.example",
    Index:          "service-logs",
    FlushInterval:  time.Second,
    RequestTimeout: 5 * time.Second,
})
if err != nil {
    return err
}

logger, err := logging.NewLogger(logging.Config{Level: "info"}, exporter)
if err != nil {
    return err
}
defer func() {
    _ = logger.Sync()
    _ = exporter.Shutdown(context.Background())
}()
```

Документы содержат `@timestamp`, `log.level`, `message`, caller, stacktrace и
все поля Zap, включая `service.name`, `trace_id` и `span_id` при наличии.
