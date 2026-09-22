# CHANGELOG

Краткий changelog для `service-kit-go`.

## 0.6.0

- `logging.Config` поддерживает `Format` (`json`/`console`) и `Color`
  (`auto`/`always`/`never`).
- Development по умолчанию использует console-вывод, а остальные окружения —
  JSON; OpenSearch exporter остаётся JSON независимо от stdout-формата.
- Console-режим раскрашивает уровни и детерминированно выделяет `trace_id`.

## 0.5.0

- Добавлено явное `postgres.enabled=false` для stateless applications. Без него обязательная проверка PostgreSQL сохранена.

## 0.4.0

- Добавлены опциональные gRPC server/client components с health service, graceful shutdown, TLS/insecure режимами, gzip, keepalive, deadlines и переносом request/trace metadata.
- Добавлены отдельные getter-интерфейсы для gRPC и service identity без расширения существующего `ServiceConfigGetter`.
- Добавлен конкурентно-безопасный OIDC `client_credentials` token provider.
- Добавлен RSA/EdDSA JWT/JWKS verifier с проверкой issuer, audience, algorithm и caller client ID.
- Добавлены unary/stream interceptors для service-to-service bearer authentication.

## 0.1.0

- Подготовлен публичный Go module `github.com/endge-lab/service-kit-go`.
- Убран старый приватный module path и локальный `replace` из `go.mod`.
- Удален `package.json`: версия Go-модуля задается git tag-ами и `CHANGELOG.md`.
- Добавлен GitHub Actions CI для `go test ./...`.
- Документация переписана под публичную публикацию через GitHub tag и локальную разработку через `go.work`.
- `httpkit/fiber` больше не зависит напрямую от `auth/fiber` для чтения `user_id` и `session_id`; эти значения теперь доступны через нейтральные helpers в `httpkit`.
