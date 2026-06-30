package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	serviceerrors "github.com/endge-lab/service-kit-go/errors"
	"github.com/endge-lab/service-kit-go/logging"
	"github.com/endge-lab/service-kit-go/telemetry"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var ErrUnauthorized = errors.New("auth unauthorized")

// Config описывает подключение к auth-service для локальной JWT-валидации.
type Config struct {
	JWKSURL          string
	Issuer           string
	AllowedAudiences []string
	CacheTTL         time.Duration
	Timeout          time.Duration
}

// Resolver валидирует access token и возвращает request identity.
type Resolver interface {
	Resolve(ctx context.Context, accessToken string) (*Identity, error)
}

type accessClaims struct {
	Role        string              `json:"role"`
	Scope       string              `json:"scope,omitempty"`
	SID         string              `json:"sid"`
	App         string              `json:"app,omitempty"`
	Platform    string              `json:"platform,omitempty"`
	Login       string              `json:"login,omitempty"`
	Username    string              `json:"username,omitempty"`
	DisplayName string              `json:"display_name,omitempty"`
	Rules       map[string][]string `json:"rules"`
	Contours    []ContourAccess     `json:"contours"`
	Groups      []string            `json:"groups"`
	Permissions []string            `json:"permissions"`
	jwt.RegisteredClaims
}

type jwk struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwksResolver struct {
	client           *http.Client
	jwksURL          string
	issuer           string
	allowedAudiences []string
	cacheTTL         time.Duration
	logger           *zap.Logger
	tracer           trace.Tracer

	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
}

// NewResolver создает стандартный JWKS-based auth resolver.
func NewResolver(cfg Config, tracer trace.Tracer, logger *zap.Logger) Resolver {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}

	return &jwksResolver{
		client: &http.Client{
			Timeout: timeout,
		},
		jwksURL:          strings.TrimSpace(cfg.JWKSURL),
		issuer:           strings.TrimSpace(cfg.Issuer),
		allowedAudiences: normalizeStrings(cfg.AllowedAudiences),
		cacheTTL:         cacheTTL,
		logger:           logging.WithComponent(logger, "auth_resolver"),
		tracer:           tracer,
		keys:             make(map[string]ed25519.PublicKey),
	}
}

// Resolve валидирует access token и нормализует identity claims.
func (r *jwksResolver) Resolve(ctx context.Context, accessToken string) (identity *Identity, err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"auth.resolve_identity",
		attribute.String("auth.jwks_url", r.jwksURL),
	)
	defer func() {
		step.End(err)
	}()

	logger := logging.WithContext(ctx, r.logger)
	tokenValue := strings.TrimSpace(accessToken)
	if tokenValue == "" {
		if logger != nil {
			logger.Warn("auth resolve rejected empty access token")
		}
		return nil, ErrUnauthorized
	}

	claims := &accessClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}

		kid, _ := token.Header["kid"].(string)
		return r.publicKeyFor(ctx, strings.TrimSpace(kid))
	}, jwt.WithIssuer(r.issuer), jwt.WithExpirationRequired())
	if err != nil {
		if logger != nil {
			logger.Warn("jwt validation failed", zap.Error(err))
		}
		return nil, ErrUnauthorized
	}

	if !parsedToken.Valid {
		if logger != nil {
			logger.Warn("jwt validation returned invalid token")
		}
		return nil, ErrUnauthorized
	}

	if len(r.allowedAudiences) > 0 && !hasAllowedAudience(claims.Audience, r.allowedAudiences) {
		if logger != nil {
			logger.Warn("jwt audience is not allowed")
		}
		return nil, ErrUnauthorized
	}

	authUserID := strings.TrimSpace(claims.Subject)
	if authUserID == "" {
		if logger != nil {
			logger.Warn("jwt subject is empty")
		}
		return nil, ErrUnauthorized
	}

	username := firstNonEmpty(claims.Username, claims.Login)
	displayName := strings.TrimSpace(claims.DisplayName)
	primaryAudience := primaryAudience(claims.Audience)
	app := firstNonEmpty(claims.App, primaryAudience)
	platform := firstNonEmpty(claims.Platform, primaryAudience)

	identity = &Identity{
		AuthUserID:  authUserID,
		Username:    username,
		DisplayName: displayName,
		Role:        strings.TrimSpace(claims.Role),
		Rules:       normalizeRules(claims.Rules),
		Contours:    normalizeContours(claims.Contours),
		Groups:      normalizeStrings(claims.Groups),
		Permissions: normalizeStrings(claims.Permissions),
		SessionID:   strings.TrimSpace(claims.SID),
		App:         app,
		Platform:    platform,
		Scope:       splitScope(claims.Scope),
	}

	if claims.ExpiresAt != nil {
		identity.ExpiresAt = claims.ExpiresAt.Time.UTC().Format(time.RFC3339)
	}

	if logger != nil {
		logger.Debug("auth identity resolved",
			zap.String("auth_user_id", identity.AuthUserID),
			zap.String("session_id", identity.SessionID),
		)
	}

	return identity, nil
}

func (r *jwksResolver) publicKeyFor(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	logger := logging.WithContext(ctx, r.logger)
	if kid == "" {
		if logger != nil {
			logger.Warn("jwt header is missing kid")
		}
		return nil, ErrUnauthorized
	}

	if key, ok := r.cachedKey(kid); ok {
		return key, nil
	}

	if err := r.refreshKeys(ctx); err != nil {
		return nil, err
	}

	if key, ok := r.cachedKey(kid); ok {
		return key, nil
	}

	return nil, ErrUnauthorized
}

func (r *jwksResolver) cachedKey(kid string) (ed25519.PublicKey, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if time.Since(r.fetchedAt) > r.cacheTTL {
		return nil, false
	}

	key, ok := r.keys[kid]
	return key, ok
}

func (r *jwksResolver) refreshKeys(ctx context.Context) (err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"auth.refresh_jwks",
		attribute.String("auth.jwks_url", r.jwksURL),
	)
	defer func() {
		step.End(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.jwksURL, nil)
	if err != nil {
		return serviceerrors.Wrap(err, "auth.jwks_request_failed", "Не удалось собрать запрос к JWKS", 500)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := r.client.Do(req)
	if err != nil {
		return serviceerrors.Wrap(err, "auth.jwks_call_failed", "Не удалось получить JWKS", 500)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return serviceerrors.Internal("auth.jwks_unexpected_status", "JWKS endpoint вернул неожиданный статус")
	}

	var payload jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return serviceerrors.Wrap(err, "auth.jwks_decode_failed", "Не удалось декодировать JWKS", 500)
	}

	keys := make(map[string]ed25519.PublicKey, len(payload.Keys))
	for _, item := range payload.Keys {
		if !strings.EqualFold(item.KTY, "OKP") || !strings.EqualFold(item.CRV, "Ed25519") {
			continue
		}

		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(item.X))
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			continue
		}

		kid := strings.TrimSpace(item.Kid)
		if kid == "" {
			continue
		}

		keys[kid] = ed25519.PublicKey(decoded)
	}
	if len(keys) == 0 {
		return serviceerrors.Internal("auth.jwks_no_supported_keys", "JWKS не содержит поддерживаемых ключей")
	}

	r.mu.Lock()
	r.keys = keys
	r.fetchedAt = time.Now()
	r.mu.Unlock()

	if r.logger != nil {
		r.logger.Debug("jwks cache refreshed", zap.Int("keys", len(keys)))
	}
	return nil
}

func splitScope(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return normalizeStrings(strings.Fields(value))
}

func hasAllowedAudience(actual jwt.ClaimStrings, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}

	for _, actualValue := range actual {
		for _, allowedValue := range allowed {
			if strings.EqualFold(strings.TrimSpace(actualValue), strings.TrimSpace(allowedValue)) {
				return true
			}
		}
	}

	return false
}

func primaryAudience(audience jwt.ClaimStrings) string {
	if len(audience) == 0 {
		return ""
	}

	return strings.TrimSpace(audience[0])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
