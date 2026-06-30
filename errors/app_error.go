package errors

import (
	stderrors "errors"
	"fmt"
)

// Code — стабильный machine-readable код ошибки.
type Code string

// AppError — канонический error contract для Endge-сервисов.
//
// Идея простая:
// - use case и adapters возвращают доменные или application-level ошибки;
// - HTTP, логи и traces читают код/статус/безопасное сообщение из этой структуры;
// - transport-слой не придумывает свои отдельные маппинги на каждый endpoint.
type AppError struct {
	code       Code
	message    string
	httpStatus int
	details    map[string]any
	cause      error
}

// New создает новую ошибку без вложенной причины.
func New(code Code, message string, httpStatus int) *AppError {
	return &AppError{
		code:       code,
		message:    message,
		httpStatus: httpStatus,
	}
}

// Wrap создает новую ошибку поверх причины.
func Wrap(cause error, code Code, message string, httpStatus int) *AppError {
	return &AppError{
		code:       code,
		message:    message,
		httpStatus: httpStatus,
		cause:      cause,
	}
}

// InvalidInput создает ошибку 400.
func InvalidInput(code Code, message string) *AppError {
	return Wrap(ErrInvalidInput, code, message, 400)
}

// Unauthorized создает ошибку 401.
func Unauthorized(code Code, message string) *AppError {
	return Wrap(ErrUnauthorized, code, message, 401)
}

// Forbidden создает ошибку 403.
func Forbidden(code Code, message string) *AppError {
	return Wrap(ErrForbidden, code, message, 403)
}

// NotFound создает ошибку 404.
func NotFound(code Code, message string) *AppError {
	return Wrap(ErrNotFound, code, message, 404)
}

// Conflict создает ошибку 409.
func Conflict(code Code, message string) *AppError {
	return Wrap(ErrConflict, code, message, 409)
}

// Internal создает ошибку 500.
func Internal(code Code, message string) *AppError {
	return Wrap(ErrInternal, code, message, 500)
}

// WithDetails добавляет machine-readable details к ошибке.
func WithDetails(err error, details map[string]any) error {
	if len(details) == 0 {
		return err
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		cloned := *appErr
		cloned.details = cloneDetails(details)
		return &cloned
	}

	internalErr := Internal("common.internal", ErrInternal.SafeMessage())
	internalErr.cause = err
	internalErr.details = cloneDetails(details)
	return internalErr
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.code, e.cause)
	}
	return string(e.code)
}

// Unwrap возвращает вложенную ошибку.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Code возвращает machine-readable код ошибки.
func (e *AppError) Code() string {
	if e == nil {
		return ""
	}
	return string(e.code)
}

// SafeMessage возвращает сообщение, безопасное для внешнего клиента.
func (e *AppError) SafeMessage() string {
	if e == nil {
		return ""
	}
	return e.message
}

// HTTPStatus возвращает HTTP-статус для transport boundary.
func (e *AppError) HTTPStatus() int {
	if e == nil {
		return 0
	}
	return e.httpStatus
}

// Details возвращает копию details, чтобы вызывающий код не мутировал исходную ошибку.
func (e *AppError) Details() map[string]any {
	if e == nil {
		return nil
	}
	return cloneDetails(e.details)
}

// Is позволяет сравнивать ошибки по стабильному коду.
func (e *AppError) Is(target error) bool {
	other, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.code != "" && e.code == other.code
}

// CodeOf читает код ошибки с graceful fallback на common.internal.
func CodeOf(err error) string {
	var appErr interface{ Code() string }
	if stderrors.As(err, &appErr) {
		return appErr.Code()
	}
	return ErrInternal.Code()
}

// SafeMessageOf читает безопасное сообщение с graceful fallback.
func SafeMessageOf(err error) string {
	var appErr interface{ SafeMessage() string }
	if stderrors.As(err, &appErr) {
		return appErr.SafeMessage()
	}
	return ErrInternal.SafeMessage()
}

// HTTPStatusOf читает HTTP-статус с graceful fallback.
func HTTPStatusOf(err error) int {
	var appErr interface{ HTTPStatus() int }
	if stderrors.As(err, &appErr) && appErr.HTTPStatus() > 0 {
		return appErr.HTTPStatus()
	}
	return ErrInternal.HTTPStatus()
}

// DetailsOf читает details, если ошибка их поддерживает.
func DetailsOf(err error) map[string]any {
	var appErr interface{ Details() map[string]any }
	if stderrors.As(err, &appErr) {
		return appErr.Details()
	}
	return nil
}

func cloneDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[key] = value
	}

	return cloned
}
