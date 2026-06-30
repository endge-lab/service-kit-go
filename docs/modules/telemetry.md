# Telemetry

`telemetry` собирает общую OTEL-настройку для сервисов: `resource`, `tracer`, `meter`, propagator и helpers для span lifecycle.

## Что входит

- `Config` для OTLP endpoint, sample mode и интервала метрик
- `NewProviders` для создания tracer/meter provider
- `Providers.Shutdown` для корректного завершения exporter-ов
- `StartTrace` и `Step` для единообразной работы со span-ами

## Поведение

- если `OTLPEndpoint` пустой, пакет поднимает локальные provider-ы без внешнего exporter-а
- propagator регистрируется глобально через `otel.SetTextMapPropagator`
- ошибки shutdown/экспорта заворачиваются в единый `errors.AppError`

## Пример

```go
providers, err := telemetry.NewProviders(ctx, telemetry.Config{
    ServiceName:     "service-core",
    ServiceVersion:  "1.2.3",
    Environment:     "production",
    OTLPEndpoint:    "otel-collector:4317",
    OTLPInsecure:    true,
    TraceSampleMode: "always",
}, logger)
if err != nil {
    return err
}
defer providers.Shutdown(context.Background())

ctx, step := telemetry.StartTrace(ctx, providers.Tracer("http"), logger, "request")
defer step.End(nil)
```
