# Публикация service-kit-go

`service-kit-go` публикуется как обычный публичный Go module через git tag в GitHub.

## Module path

```go
module github.com/endge-lab/service-kit-go
```

## Локальная проверка

```bash
go test ./...
```

Если рядом есть сервис-потребитель, проверяйте его через `go.work`:

```bash
go work init ./service-kit-go ./service-template-go
cd service-template-go
go test ./...
```

## Релиз

```bash
git tag v0.1.0
git push origin v0.1.0
```

После push тега версия становится доступна потребителям:

```bash
go get github.com/endge-lab/service-kit-go@v0.1.0
go mod tidy
```

## Правила

- Используйте semver tags: `vMAJOR.MINOR.PATCH`.
- Не публикуйте breaking changes без повышения major-версии.
- Не добавляйте в kit project-specific доменную логику.
- Не делайте optional-инфраструктуру обязательной для всех потребителей.
