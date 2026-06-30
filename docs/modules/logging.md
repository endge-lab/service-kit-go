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
