package oidc

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = errors.New("service identity unauthorized")

type VerifierConfig struct {
	Issuer            string
	JWKSURL           string
	Audience          string
	AllowedCallers    []string
	AllowedAlgorithms []string
	CacheTTL          time.Duration
	Timeout           time.Duration
}

type Identity struct {
	Subject  string
	ClientID string
}

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

type claims struct {
	AuthorizedParty string `json:"azp"`
	ClientID        string `json:"client_id"`
	jwt.RegisteredClaims
}

type verifier struct {
	config  VerifierConfig
	client  *http.Client
	callers map[string]struct{}
	methods map[string]struct{}

	mu        sync.RWMutex
	keys      map[string]any
	fetchedAt time.Time
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KTY string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewVerifier(config VerifierConfig) (TokenVerifier, error) {
	config.Issuer = strings.TrimSpace(config.Issuer)
	config.JWKSURL = strings.TrimSpace(config.JWKSURL)
	config.Audience = strings.TrimSpace(config.Audience)
	if config.Issuer == "" || config.JWKSURL == "" || config.Audience == "" {
		return nil, errors.New("oidc verifier issuer, jwks url and audience are required")
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	callers := stringSet(config.AllowedCallers)
	if len(callers) == 0 {
		return nil, errors.New("oidc verifier allowed callers are required")
	}
	methods := stringSet(config.AllowedAlgorithms)
	if len(methods) == 0 {
		return nil, errors.New("oidc verifier allowed algorithms are required")
	}
	return &verifier{
		config:  config,
		client:  &http.Client{Timeout: config.Timeout},
		callers: callers,
		methods: methods,
		keys:    make(map[string]any),
	}, nil
}

func (v *verifier) Verify(ctx context.Context, tokenValue string) (Identity, error) {
	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return Identity{}, ErrUnauthorized
	}
	parsedClaims := &claims{}
	parsed, err := jwt.ParseWithClaims(tokenValue, parsedClaims, func(token *jwt.Token) (any, error) {
		algorithm := token.Method.Alg()
		if _, allowed := v.methods[algorithm]; !allowed {
			return nil, ErrUnauthorized
		}
		kid, _ := token.Header["kid"].(string)
		return v.key(ctx, strings.TrimSpace(kid), algorithm)
	}, jwt.WithIssuer(v.config.Issuer), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid || !hasAudience(parsedClaims.Audience, v.config.Audience) {
		return Identity{}, ErrUnauthorized
	}
	caller := firstNonEmpty(parsedClaims.AuthorizedParty, parsedClaims.ClientID, parsedClaims.Subject)
	if _, allowed := v.callers[caller]; !allowed {
		return Identity{}, ErrUnauthorized
	}
	if strings.TrimSpace(parsedClaims.Subject) == "" {
		return Identity{}, ErrUnauthorized
	}
	return Identity{Subject: parsedClaims.Subject, ClientID: caller}, nil
}

func (v *verifier) key(ctx context.Context, kid, algorithm string) (any, error) {
	if kid == "" {
		return nil, ErrUnauthorized
	}
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < v.config.CacheTTL
	v.mu.RUnlock()
	if ok && fresh {
		return keyForAlgorithm(key, algorithm)
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, ErrUnauthorized
	}
	return keyForAlgorithm(key, algorithm)
}

func (v *verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.config.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("build jwks request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}
	var document jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]any, len(document.Keys))
	for _, item := range document.Keys {
		if item.Kid == "" {
			continue
		}
		key, err := parseJWK(item)
		if err != nil {
			continue
		}
		keys[item.Kid] = key
	}
	if len(keys) == 0 {
		return errors.New("jwks contains no supported keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func parseJWK(item jwk) (any, error) {
	switch item.KTY {
	case "RSA":
		modulus, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil || len(modulus) == 0 {
			return nil, errors.New("invalid rsa modulus")
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil || len(exponentBytes) == 0 {
			return nil, errors.New("invalid rsa exponent")
		}
		exponent := 0
		for _, current := range exponentBytes {
			exponent = exponent<<8 + int(current)
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}, nil
	case "OKP":
		if item.CRV != "Ed25519" {
			return nil, errors.New("unsupported okp curve")
		}
		publicKey, err := base64.RawURLEncoding.DecodeString(item.X)
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			return nil, errors.New("invalid ed25519 key")
		}
		return ed25519.PublicKey(publicKey), nil
	default:
		return nil, errors.New("unsupported jwk type")
	}
}

func keyForAlgorithm(key any, algorithm string) (any, error) {
	switch {
	case strings.HasPrefix(algorithm, "RS") || strings.HasPrefix(algorithm, "PS"):
		if _, ok := key.(*rsa.PublicKey); !ok {
			return nil, ErrUnauthorized
		}
	case algorithm == "EdDSA":
		if _, ok := key.(ed25519.PublicKey); !ok {
			return nil, ErrUnauthorized
		}
	default:
		return nil, ErrUnauthorized
	}
	return key, nil
}

func hasAudience(values jwt.ClaimStrings, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result[trimmed] = struct{}{}
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
