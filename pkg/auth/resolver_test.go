package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResolverResolvesIdentity(t *testing.T) {
	t.Parallel()

	fixture, err := newResolverFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}

	token, err := fixture.issueToken(tokenOptions{
		Subject:     "user-1",
		Username:    "alice",
		DisplayName: "Alice",
		Role:        "admin",
		SessionID:   "session-1",
		Rules: map[string][]string{
			" todos ": {" read ", "READ", ""},
			" ":       {"ignored"},
		},
		Contours: []ContourAccess{
			{Contour: "admin", Permissions: []string{"admin.users.read"}},
			{Contour: " Admin ", Permissions: []string{"duplicate"}},
			{Contour: " ", Permissions: []string{"ignored"}},
		},
		Permissions: []string{"admin.users.read"},
		Scope:       "read write read",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	resolver := newTestResolver(t, fixture, http.StatusOK, "", nil)
	identity, err := resolver.Resolve(context.Background(), token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if identity.AuthUserID != "user-1" {
		t.Fatalf("unexpected auth user id: %s", identity.AuthUserID)
	}
	if identity.Username != "alice" || identity.DisplayName != "Alice" || identity.SessionID != "session-1" {
		t.Fatalf("unexpected identity fields: %#v", identity)
	}
	if !HasPermission(identity, "admin.users.read") {
		t.Fatal("expected admin.users.read permission")
	}
	if len(identity.Contours) != 1 || identity.Contours[0].Contour != "admin" {
		t.Fatalf("Contours = %#v, want one normalized admin contour", identity.Contours)
	}
	if got := identity.Rules["todos"]; len(got) != 1 || got[0] != "read" {
		t.Fatalf("Rules[todos] = %#v, want normalized read", got)
	}
	if got := identity.Scope; len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("Scope = %#v, want normalized read/write", got)
	}
	if identity.ExpiresAt == "" {
		t.Fatal("ExpiresAt = empty, want RFC3339 value")
	}
}

func TestResolverUsesAudienceAsAppAndPlatformFallback(t *testing.T) {
	t.Parallel()

	fixture, err := newResolverFixture("https://auth.example.com", "web")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}
	token, err := fixture.issueToken(tokenOptions{Subject: "user-1", Audience: []string{" mobile "}})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	resolver := newTestResolver(t, fixture, http.StatusOK, "", nil)
	resolver.allowedAudiences = nil

	identity, err := resolver.Resolve(context.Background(), token)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if identity.App != "mobile" || identity.Platform != "mobile" {
		t.Fatalf("App/Platform = %q/%q, want mobile/mobile", identity.App, identity.Platform)
	}
}

func TestResolverRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	fixture, err := newResolverFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}
	validToken, err := fixture.issueToken(tokenOptions{
		Subject:  "user-1",
		Audience: []string{"other-platform"},
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	expiredToken, err := fixture.issueToken(tokenOptions{
		Subject:   "user-1",
		ExpiresIn: -time.Hour,
	})
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: "  "},
		{name: "malformed", token: "not-a-jwt"},
		{name: "expired", token: expiredToken},
		{name: "audience not allowed", token: validToken},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestResolver(t, fixture, http.StatusOK, "", nil)
			identity, err := resolver.Resolve(context.Background(), tt.token)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Resolve() error = %v, want ErrUnauthorized", err)
			}
			if identity != nil {
				t.Fatalf("Resolve() identity = %#v, want nil", identity)
			}
		})
	}
}

func TestResolverWrapsJWKSFailures(t *testing.T) {
	t.Parallel()

	fixture, err := newResolverFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}
	token, err := fixture.issueToken(tokenOptions{Subject: "user-1"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	tests := []struct {
		name      string
		status    int
		body      string
		transport error
	}{
		{name: "transport error", transport: errors.New("dial failed")},
		{name: "unexpected status", status: http.StatusServiceUnavailable, body: `{"keys":[]}`},
		{name: "invalid json", status: http.StatusOK, body: `{`},
		{name: "no supported keys", status: http.StatusOK, body: `{"keys":[{"kty":"RSA","kid":"rsa"}]}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestResolver(t, fixture, tt.status, tt.body, tt.transport)
			identity, err := resolver.Resolve(context.Background(), token)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Resolve() error = %v, want ErrUnauthorized", err)
			}
			if identity != nil {
				t.Fatalf("Resolve() identity = %#v, want nil", identity)
			}
		})
	}
}

func TestResolverCachesJWKSAndIsRaceSafe(t *testing.T) {
	t.Parallel()

	fixture, err := newResolverFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}
	token, err := fixture.issueToken(tokenOptions{Subject: "user-1"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	var calls atomic.Int64
	resolver := newTestResolverWithTransport(fixture, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return jwksResponseFromFixture(fixture, http.StatusOK, "")
	}))

	if _, err := resolver.Resolve(context.Background(), token); err != nil {
		t.Fatalf("warm Resolve() error = %v, want nil", err)
	}

	const workers = 16
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			identity, err := resolver.Resolve(context.Background(), token)
			if err != nil {
				t.Errorf("Resolve() error = %v, want nil", err)
				return
			}
			if identity.AuthUserID != "user-1" {
				t.Errorf("AuthUserID = %q, want user-1", identity.AuthUserID)
			}
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("JWKS calls = %d, want cached reuse after warm fetch", got)
	}
}

func newTestResolver(t *testing.T, fixture *resolverFixture, status int, body string, transportErr error) *jwksResolver {
	t.Helper()

	return newTestResolverWithTransport(fixture, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("JWKS method = %s, want GET", req.Method)
		}
		if transportErr != nil {
			return nil, transportErr
		}
		return jwksResponseFromFixture(fixture, status, body)
	}))
}

func newTestResolverWithTransport(fixture *resolverFixture, transport http.RoundTripper) *jwksResolver {
	resolver := NewResolver(fixture.resolverConfig("https://auth.example.com/.well-known/jwks.json"), noop.NewTracerProvider().Tracer("test"), zap.NewNop()).(*jwksResolver)
	resolver.client.Transport = transport
	return resolver
}

func jwksResponseFromFixture(fixture *resolverFixture, status int, body string) (*http.Response, error) {
	if status == 0 {
		status = http.StatusOK
	}
	if body == "" && status == http.StatusOK {
		body = fixture.jwksBody()
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

type resolverFixture struct {
	keyID      string
	issuer     string
	audience   []string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

type tokenOptions struct {
	Subject     string
	Username    string
	DisplayName string
	Role        string
	Rules       map[string][]string
	Contours    []ContourAccess
	Groups      []string
	Permissions []string
	SessionID   string
	App         string
	Platform    string
	Scope       string
	Audience    []string
	ExpiresIn   time.Duration
}

func newResolverFixture(issuer string, audience ...string) (*resolverFixture, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &resolverFixture{
		keyID:      "test-key",
		issuer:     issuer,
		audience:   audience,
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

func (f *resolverFixture) resolverConfig(jwksURL string) Config {
	return Config{
		JWKSURL:          jwksURL,
		Issuer:           f.issuer,
		AllowedAudiences: f.audience,
		CacheTTL:         time.Minute,
		Timeout:          time.Second,
	}
}

func (f *resolverFixture) issueToken(options tokenOptions) (string, error) {
	expiresIn := options.ExpiresIn
	if expiresIn == 0 {
		expiresIn = time.Hour
	}
	audience := options.Audience
	if len(audience) == 0 {
		audience = f.audience
	}

	claims := accessClaims{
		Role:        options.Role,
		Scope:       options.Scope,
		SID:         options.SessionID,
		App:         options.App,
		Platform:    options.Platform,
		Username:    options.Username,
		DisplayName: options.DisplayName,
		Rules:       options.Rules,
		Contours:    options.Contours,
		Groups:      options.Groups,
		Permissions: options.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    f.issuer,
			Subject:   options.Subject,
			Audience:  jwt.ClaimStrings(audience),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = f.keyID
	return token.SignedString(f.privateKey)
}

func (f *resolverFixture) jwksBody() string {
	payload := struct {
		Keys []jwk `json:"keys"`
	}{
		Keys: []jwk{{
			KTY: "OKP",
			CRV: "Ed25519",
			X:   base64.RawURLEncoding.EncodeToString(f.publicKey),
			Use: "sig",
			Alg: "EdDSA",
			Kid: f.keyID,
		}},
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}
