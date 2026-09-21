# Auth

`auth` дает общий runtime-контракт для всех Go-сервисов Endge:

- локальная проверка JWT через `JWKS`
- нормализованная `Identity`
- helpers для проверки `contour`, `group`, `permission`

## Что входит

- `Config` для подключения к `auth-service`
- `Resolver` и `NewResolver`
- `Identity` и `ContourAccess`
- `HasContour`, `HasGroup`, `HasPermission`
- `RequireContour`, `RequirePermission`

## Принцип работы

- сервис валидирует JWT локально, без remote introspection на каждый запрос
- публичные ключи берутся из `JWKS` и кешируются
- `owner` рассматривается как глобальный bypass

## Пример

```go
resolver := auth.NewResolver(auth.Config{
    JWKSURL:          "https://auth.example.com/.well-known/jwks.json",
    Issuer:           "https://auth.example.com",
    AllowedAudiences: []string{"example-platform"},
}, providers.Tracer("auth"), logger)

identity, err := resolver.Resolve(ctx, token)
if err != nil {
    return err
}

if err := auth.RequirePermission(identity, "admin.users.set-role"); err != nil {
    return err
}
```
