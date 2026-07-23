# Telemetry

`telemetry` собирает общую OTEL-настройку для сервисов: `resource`, `tracer`, `meter`, propagator и helpers для span lifecycle.

## Что входит

- `Config` для OTLP endpoint, sample mode, интервала метрик и Prometheus reader
- `NewProviders` для создания tracer/meter provider
- `Providers.Shutdown` для корректного завершения exporter-ов
- `StartTrace` и `Step` для единообразной работы со span-ами

## Поведение

- если `PrometheusEnabled=true`, meter provider использует pull reader и
  `Providers.PrometheusHandler()` отдаёт scrape handler; OTLP metric reader в
  этом режиме не создаётся, чтобы не дублировать серии
- если `OTLPEndpoint` пустой и Prometheus выключен, пакет поднимает локальный
  meter provider без внешнего exporter-а
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

## Prometheus

`PrometheusEnabled` подключает официальный Prometheus exporter к тому же
`MeterProvider`, поэтому HTTP, use case и repository instruments автоматически
попадают в scrape. Сам HTTP-сервер создаётся отдельно через `PrometheusServer`:

```go
providers, err := telemetry.NewProviders(ctx, telemetry.Config{
    ServiceName:       "service-core",
    PrometheusEnabled: true,
}, logger)
if err != nil {
    return err
}

server, err := telemetry.NewPrometheusServer(
    telemetry.PrometheusServerConfig{BindAddress: ":9090", HandlerPath: "/metrics"},
    providers.PrometheusHandler(),
    logger,
)
if err != nil {
    return err
}
if err := server.Start(); err != nil {
    return err
}
defer server.Shutdown(context.Background())
```

Scrape endpoint должен быть доступен Prometheus, но не публичному интернету.
