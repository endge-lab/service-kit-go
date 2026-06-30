# HTTP Kit Fiber

`httpkit/fiber` собирает повторяемые Fiber middleware и технические handlers.

## Что входит

- `RequestIDMiddleware`
- `TraceMiddleware`
- `RequestLoggerMiddleware`
- `NewRequestMetricsMiddleware`
- `RecoveryMiddleware`
- `NewFiberErrorHandler`
- `HealthHandler`
- `VersionHandler`

## Зачем отдельный модуль

Такой пакет позволяет не копировать одинаковые middlewares между `service-core`, `service-chat`, `service-engagement` и новыми сервисами.
