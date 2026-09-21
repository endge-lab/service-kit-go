# Testing

`testing` дает общие test helpers для сервисов Endge.

## Что входит в v1

- `AuthFixture` с Ed25519 ключом
- `IssueToken` для генерации тестовых JWT
- `JWKSHandler` для локального mock JWKS endpoint
- `ResolverConfig` для быстрого подключения `auth.Resolver` в тестах

## Назначение

Пакет нужен, чтобы каждый сервис не собирал заново одинаковые auth fixtures для middleware, handler и integration тестов.
