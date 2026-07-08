package httpkit

import (
	"context"
	"strings"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
	sessionIDKey contextKey = "session_id"
)

// ErrorResponse задает единый JSON-формат ошибок.
type ErrorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// WithRequestID кладет request id в context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, strings.TrimSpace(requestID))
}

// RequestIDFromContext возвращает request id из context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(requestIDKey).(string)
	return value, ok
}

// WithUserID кладет user id в context для middleware, логов и метрик.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, strings.TrimSpace(userID))
}

// UserIDFromContext возвращает user id из context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(userIDKey).(string)
	return value, ok
}

// WithSessionID кладет session id в context для middleware, логов и метрик.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, strings.TrimSpace(sessionID))
}

// SessionIDFromContext возвращает session id из context.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(sessionIDKey).(string)
	return value, ok
}

// SanitizeURL убирает query string из URL перед логированием и метриками.
func SanitizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	for index, char := range rawURL {
		if char == '?' {
			return rawURL[:index]
		}
	}
	return rawURL
}
