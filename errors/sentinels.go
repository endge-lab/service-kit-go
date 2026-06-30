package errors

// Базовые sentinel-ошибки. На них удобно опираться при сборке сервисных ошибок.
var (
	ErrInvalidInput = New("common.invalid_input", "Некорректный запрос", 400)
	ErrUnauthorized = New("common.unauthorized", "Требуется авторизация", 401)
	ErrForbidden    = New("common.forbidden", "Недостаточно прав", 403)
	ErrNotFound     = New("common.not_found", "Ресурс не найден", 404)
	ErrConflict     = New("common.conflict", "Конфликт состояния", 409)
	ErrInternal     = New("common.internal", "Внутренняя ошибка сервиса", 500)
)
