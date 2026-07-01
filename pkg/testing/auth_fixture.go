package testing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	serviceauth "github.com/endge-lab/service-kit-go/pkg/auth"

	"github.com/golang-jwt/jwt/v5"
)

// TokenOptions задает claims для тестового access token.
type TokenOptions struct {
	Subject     string
	Username    string
	DisplayName string
	Role        string
	Rules       map[string][]string
	Contours    []serviceauth.ContourAccess
	Groups      []string
	Permissions []string
	SessionID   string
	App         string
	Platform    string
	Scope       string
	Audience    []string
	ExpiresIn   time.Duration
}

// AuthFixture поднимает тестовый Ed25519 ключ и выдает JWT/JWKS.
type AuthFixture struct {
	KeyID      string
	Issuer     string
	Audience   []string
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

type testClaims struct {
	Role        string                      `json:"role"`
	Scope       string                      `json:"scope,omitempty"`
	SID         string                      `json:"sid"`
	App         string                      `json:"app,omitempty"`
	Platform    string                      `json:"platform,omitempty"`
	Username    string                      `json:"username,omitempty"`
	DisplayName string                      `json:"display_name,omitempty"`
	Rules       map[string][]string         `json:"rules,omitempty"`
	Contours    []serviceauth.ContourAccess `json:"contours,omitempty"`
	Groups      []string                    `json:"groups,omitempty"`
	Permissions []string                    `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// NewAuthFixture создает тестовый auth fixture.
func NewAuthFixture(issuer string, audience ...string) (*AuthFixture, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &AuthFixture{
		KeyID:      "test-key",
		Issuer:     strings.TrimSpace(issuer),
		Audience:   audience,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// ResolverConfig возвращает стандартную конфигурацию для auth.Resolver.
func (f *AuthFixture) ResolverConfig(jwksURL string) serviceauth.Config {
	return serviceauth.Config{
		JWKSURL:          strings.TrimSpace(jwksURL),
		Issuer:           f.Issuer,
		AllowedAudiences: f.Audience,
		CacheTTL:         time.Minute,
		Timeout:          time.Second,
	}
}

// IssueToken подписывает тестовый JWT.
func (f *AuthFixture) IssueToken(options TokenOptions) (string, error) {
	expiresIn := options.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}

	audience := options.Audience
	if len(audience) == 0 {
		audience = f.Audience
	}

	claims := testClaims{
		Role:        strings.TrimSpace(options.Role),
		Scope:       strings.TrimSpace(options.Scope),
		SID:         strings.TrimSpace(options.SessionID),
		App:         strings.TrimSpace(options.App),
		Platform:    strings.TrimSpace(options.Platform),
		Username:    strings.TrimSpace(options.Username),
		DisplayName: strings.TrimSpace(options.DisplayName),
		Rules:       options.Rules,
		Contours:    options.Contours,
		Groups:      options.Groups,
		Permissions: options.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    f.Issuer,
			Subject:   strings.TrimSpace(options.Subject),
			Audience:  jwt.ClaimStrings(audience),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = f.KeyID
	return token.SignedString(f.PrivateKey)
}

// JWKSHandler возвращает http handler с публичным ключом для тестов.
func (f *AuthFixture) JWKSHandler() http.Handler {
	type jwk struct {
		KTY string `json:"kty"`
		CRV string `json:"crv"`
		X   string `json:"x"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	type response struct {
		Keys []jwk `json:"keys"`
	}

	payload := response{
		Keys: []jwk{
			{
				KTY: "OKP",
				CRV: "Ed25519",
				X:   base64.RawURLEncoding.EncodeToString(f.PublicKey),
				Use: "sig",
				Alg: "EdDSA",
				Kid: f.KeyID,
			},
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
}
