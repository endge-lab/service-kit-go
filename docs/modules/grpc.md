# gRPC runtime

Пакет `pkg/grpckit` предоставляет только транспортную инфраструктуру и не содержит protobuf-контрактов приложений.

- `NewServer` создаёт gRPC server со стандартным Health service и bounded graceful shutdown.
- `NewClient` поддерживает TLS или явно включённый локальный insecure transport, gzip и ограничения размера сообщений.
- Unary/stream interceptors переносят `x-request-id`, `traceparent` и `tracestate`.
- Default deadline применяется только когда вызывающий context ещё не содержит deadline.

Service-specific handlers и canonical proto остаются в application-владельце контракта.
