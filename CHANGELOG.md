# CHANGELOG

Краткий changelog для `service-kit-go`.

## 0.1.0

- Подготовлен публичный Go module `github.com/endge-lab/service-kit-go`.
- Убран старый приватный module path и локальный `replace` из `go.mod`.
- Добавлен GitHub Actions CI для `go test ./...`.
- Документация переписана под публичную публикацию через GitHub tag и локальную разработку через `go.work`.
- `httpkit/fiber` больше не зависит напрямую от `auth/fiber` для чтения `user_id` и `session_id`; эти значения теперь доступны через нейтральные helpers в `httpkit`.
