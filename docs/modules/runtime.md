# Runtime

`runtime` собирает общий lifecycle сервисов.

## Что входит

- `Lifecycle` с hook-ами `Start`/`Stop`
- `Hook` для явного описания runtime-компонента
- `NotifyContext` для graceful shutdown по `SIGINT` и `SIGTERM`

## Когда использовать

- bootstrap HTTP-сервера
- запуск background worker-ов
- корректное завершение telemetry/exporter-ов и других ресурсов
