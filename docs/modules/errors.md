# Errors

Пакет: `errors`

## Назначение

`errors` задает единый machine-readable error contract для всех Go-сервисов Endge.

Он нужен, чтобы:

- use case и adapters возвращали один и тот же формат ошибок;
- HTTP transport не держал ручной mapping статусов на каждом endpoint;
- логи и traces читали стабильный `error.code`, а не парсили `err.Error()`.

## Что внутри

- `AppError`
- sentinel-ошибки `ErrInvalidInput`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrConflict`, `ErrInternal`
- конструкторы `InvalidInput`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `Internal`
- helpers `CodeOf`, `SafeMessageOf`, `HTTPStatusOf`, `DetailsOf`, `WithDetails`

## Как использовать

```go
return errors.Forbidden("admin.permission_denied", "Недостаточно прав")
```

Или с деталями:

```go
return errors.WithDetails(
    errors.Forbidden("admin.permission_denied", "Недостаточно прав"),
    map[string]any{"permission": "admin.users.read"},
)
```

## Что не делать

- не использовать `errors` как место для бизнесовых правил сервиса;
- не хранить тут transport DTO;
- не подменять `AppError` строковыми `fmt.Errorf(...)` на boundary-слое.
