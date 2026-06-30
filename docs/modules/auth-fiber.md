# Auth Fiber

`auth/fiber` подключает `auth.Resolver` к Fiber middleware.

## Что входит

- `Middleware.Authenticate` для обязательной авторизации
- `Middleware.Optional` для опциональной авторизации
- `RequireContour` и `RequirePermission`
- `GetIdentity`, `GetUserID`, `GetSessionID`

## Поведение

- access token читается из `Authorization: Bearer ...`
- опционально можно разрешить `access_token` из query
- после успешной проверки `Identity` кладется в `context` и `fiber.Locals`

## Пример

```go
authMiddleware := fiberauth.NewMiddleware(resolver, logger)

app.Use(authMiddleware.Authenticate(false))
app.Get("/admin/users", fiberauth.RequirePermission("admin.users.read"), handler)
```
