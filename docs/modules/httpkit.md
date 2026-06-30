# HTTP Kit

`httpkit` содержит transport-agnostic helpers для HTTP runtime.

## Что входит

- `ErrorResponse` для единого JSON-формата ошибок
- `WithRequestID` и `RequestIDFromContext`
- `SanitizeURL` для логирования URL без query string

## Назначение

Пакет остается минимальным и не зависит от Fiber. Все Fiber-специфичные middleware лежат отдельно в `httpkit/fiber`.
