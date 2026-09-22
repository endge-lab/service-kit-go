# Logging

`logging` дает единый `zap`-логгер для Go-сервисов Endge.

## Что входит

- `Config` с полями `service.name`, `deployment.environment`, `service.version`
- `NewLogger` с JSON-выводом для production и цветным console-выводом для local development
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

## Формат stdout

`Format` отвечает только за представление stdout и не изменяет структуру
OpenSearch-документов. При пустом `Format` logger выбирает `console` для
`development` и `json` для остальных окружений.

| Окружение | Пустой `Format` | Пустой `Color` |
| --- | --- | --- |
| `development` | `console` | `auto` |
| `test`, `staging`, `production` и остальные | `json` | `never` |

Явные `Format` и `Color` имеют приоритет над этими значениями. `auto` включает
цвета только для интерактивного терминала; в Docker и других не-TTY выводах
ANSI-коды не добавляются.

```go
logger, err := logging.NewLogger(logging.Config{
    Level:       "debug",
    Format:      "console", // json | console
    Color:       "always",  // auto | always | never
    Environment: "development",
})
```

В `console`-режиме уровни раскрашены, а одинаковые `trace_id` получают один и
тот же цвет. В production используйте `json`; ANSI-последовательности в JSON
не добавляются.

## OpenSearch exporter

`OpenSearchExporter` — опциональный второй Zap core. Exporter всегда строит
JSON-документы, даже если stdout использует console-формат, и асинхронно группирует записи и отправляет их в
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
