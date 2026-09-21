package fiber

import (
	"context"
	"errors"
	"strings"

	serviceauth "github.com/endge-lab/service-kit-go/pkg/auth"
	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/endge-lab/service-kit-go/pkg/httpkit"
	"github.com/endge-lab/service-kit-go/pkg/logging"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	sessionIDKey contextKey = "session_id"
	identityKey  contextKey = "identity"
)

// Middleware переиспользует auth.Resolver внутри Fiber runtime.
type Middleware struct {
	resolver serviceauth.Resolver
	logger   *zap.Logger
}

// NewMiddleware создает auth middleware для Fiber.
func NewMiddleware(resolver serviceauth.Resolver, logger *zap.Logger) *Middleware {
	return &Middleware{
		resolver: resolver,
		logger:   logging.WithComponent(logger, "auth_middleware"),
	}
}

// Authenticate требует обязательный access token.
func (m *Middleware) Authenticate(allowQueryToken bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := m.AuthenticateRequest(c, allowQueryToken); err != nil {
			return err
		}
		return c.Next()
	}
}

// Optional пытается авторизовать пользователя, но пропускает запрос без токена.
func (m *Middleware) Optional(allowQueryToken bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenValue := accessTokenFromRequest(c, allowQueryToken)
		if tokenValue == "" {
			return c.Next()
		}

		if err := m.AuthenticateRequest(c, allowQueryToken); err != nil {
			return err
		}
		return c.Next()
	}
}

// AuthenticateRequest валидирует токен и кладет identity в request context без вызова next handler.
func (m *Middleware) AuthenticateRequest(c *fiber.Ctx, allowQueryToken bool) error {
	return m.authenticate(c, allowQueryToken)
}

func (m *Middleware) authenticate(c *fiber.Ctx, allowQueryToken bool) error {
	tokenValue := accessTokenFromRequest(c, allowQueryToken)
	if tokenValue == "" {
		return serviceerrors.Unauthorized("auth.access_token_required", "Требуется access token")
	}

	identity, err := m.resolver.Resolve(c.UserContext(), tokenValue)
	if err != nil {
		logger := logging.WithContext(c.UserContext(), m.logger)
		if errors.Is(err, serviceauth.ErrUnauthorized) {
			if logger != nil {
				logger.Warn("invalid access token", zap.Error(err))
			}
			return serviceerrors.Unauthorized("auth.invalid_access_token", "Access token недействителен или просрочен")
		}

		if logger != nil {
			logger.Error("auth validation failed", zap.Error(err))
		}
		return serviceerrors.Internal("auth.service_unavailable", "Сервис авторизации временно недоступен")
	}

	userID := strings.TrimSpace(identity.AuthUserID)
	if userID == "" {
		return serviceerrors.Unauthorized("auth.identity_missing", "В токене отсутствует идентификатор пользователя")
	}

	ctx := httpkit.WithUserID(c.UserContext(), userID)
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, identityKey, identity)
	if sessionID := strings.TrimSpace(identity.SessionID); sessionID != "" {
		ctx = httpkit.WithSessionID(ctx, sessionID)
		ctx = context.WithValue(ctx, sessionIDKey, sessionID)
		c.Locals(string(sessionIDKey), sessionID)
	}

	c.SetUserContext(ctx)
	c.Locals(string(userIDKey), userID)
	c.Locals(string(identityKey), identity)

	logger := logging.WithContext(ctx, m.logger)
	if logger != nil {
		logger.Debug("request identity resolved",
			zap.String("auth_user_id", userID),
			zap.String("session_id", strings.TrimSpace(identity.SessionID)),
		)
	}

	return nil
}

// RequireContour проверяет контур уже после аутентификации.
func RequireContour(contour string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity, ok := GetIdentity(c.UserContext())
		if !ok {
			return serviceerrors.Unauthorized("auth.identity_missing", "Пользователь не аутентифицирован")
		}
		if err := serviceauth.RequireContour(identity, contour); err != nil {
			return err
		}
		return c.Next()
	}
}

// RequirePermission проверяет permission уже после аутентификации.
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity, ok := GetIdentity(c.UserContext())
		if !ok {
			return serviceerrors.Unauthorized("auth.identity_missing", "Пользователь не аутентифицирован")
		}
		if err := serviceauth.RequirePermission(identity, permission); err != nil {
			return err
		}
		return c.Next()
	}
}

// GetIdentity возвращает identity из context.
func GetIdentity(ctx context.Context) (*serviceauth.Identity, bool) {
	identity, ok := ctx.Value(identityKey).(*serviceauth.Identity)
	return identity, ok
}

// GetUserID возвращает auth user id из context.
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// GetSessionID возвращает session id из context.
func GetSessionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey).(string)
	return id, ok
}

func accessTokenFromRequest(c *fiber.Ctx, allowQueryToken bool) string {
	if allowQueryToken {
		if queryToken := strings.TrimSpace(c.Query("access_token")); queryToken != "" {
			return queryToken
		}
	}

	authHeader := c.Get(fiber.HeaderAuthorization)
	if authHeader == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}

	return strings.TrimSpace(authHeader[len("Bearer "):])
}
