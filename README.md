# Endge Service Kit Go

`service-kit-go` — публичный Go toolkit для повторяемой инфраструктуры backend-сервисов.

Модуль публикуется как обычный Go module:

```go
module github.com/endge-lab/service-kit-go
```

## Что входит

- `errors` — единый machine-readable error contract.
- `logging` — JSON-логгер на `zap`.
- `telemetry` — OpenTelemetry traces/metrics helpers.
- `httpkit` — общие HTTP payload-и и context helpers.
- `httpkit/fiber` — optional Fiber middleware: request id, logs, metrics, recovery, error handler.
- `auth` и `auth/fiber` — optional JWT/JWKS авторизация.
- `redpanda` — optional Kafka/Redpanda runtime wrapper.
- `runtime` — lifecycle и signal-aware shutdown helpers.
- `testing` — test fixtures для auth/runtime слоев.

Пакеты независимы по смыслу: сервис импортирует только то, что реально использует. Монолит без авторизации и без Kafka может подключить только `errors`, `httpkit`, `logging` или вообще не использовать logging-пакет.

## Установка

```bash
go get github.com/endge-lab/service-kit-go@v0.1.0
```

Пример импорта:

```go
import "github.com/endge-lab/service-kit-go/errors"
```

## Локальная разработка рядом с сервисом

В Go аналог workspace из frontend-монорепозиториев — это `go.work`.

Если `service-kit-go` и сервис лежат рядом:

```text
workspace/
├── service-kit-go/
└── service-template-go/
```

создайте `go.work` в папке `workspace`:

```bash
go work init ./service-kit-go ./service-template-go
```

С этого момента сервис продолжает иметь в `go.mod` опубликованную зависимость:

```go
require github.com/endge-lab/service-kit-go v0.1.0
```

но локально Go будет брать код kit из папки `./service-kit-go`. В CI и production `go.work` обычно отсутствует, поэтому Go скачивает tagged-версию из GitHub.

До публикации первого тега `v0.1.0` команды вроде `go mod tidy` в сервисе-потребителе могут пытаться скачать ещё несуществующую версию из GitHub. Для bootstrap-проверки можно временно добавить в `go.mod` потребителя:

```go
replace github.com/endge-lab/service-kit-go => ../service-kit-go
```

Не коммитьте этот `replace` в публичный сервисный репозиторий.

Проверка kit:

```bash
go test ./...
```

## Публикация версии

Go module публикуется git-тегом:

```bash
git tag v0.1.0
git push origin v0.1.0
```

После этого потребители обновляются явно:

```bash
go get github.com/endge-lab/service-kit-go@v0.1.0
go mod tidy
```

## Правила дизайна

- Не добавляйте доменную бизнес-логику в kit.
- Не делайте auth, telemetry, Redpanda или logging обязательными для всех сервисов.
- Общие helpers должны нормально работать с `nil`/noop logger, если logger не является сутью пакета.
- Пакеты с тяжелыми внешними зависимостями держите отдельно от core-пакетов.
