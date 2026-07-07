package testing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueToken(t *testing.T) {
	t.Parallel()

	fixture, err := NewAuthFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}

	token, err := fixture.IssueToken(TokenOptions{
		Subject: "user-1",
		Role:    "user",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}

	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if parsed.Header["kid"] != fixture.KeyID {
		t.Fatalf("kid = %v, want %s", parsed.Header["kid"], fixture.KeyID)
	}
}

func TestAuthFixtureConfigTokenDefaultsAndJWKSHandler(t *testing.T) {
	t.Parallel()

	fixture, err := NewAuthFixture(" https://auth.example.com ", "web")
	if err != nil {
		t.Fatalf("NewAuthFixture() error = %v", err)
	}
	if fixture.Issuer != "https://auth.example.com" {
		t.Fatalf("Issuer = %q, want trimmed issuer", fixture.Issuer)
	}

	cfg := fixture.ResolverConfig(" https://auth.example.com/jwks ")
	if cfg.JWKSURL != "https://auth.example.com/jwks" || cfg.Issuer != fixture.Issuer || len(cfg.AllowedAudiences) != 1 || cfg.AllowedAudiences[0] != "web" {
		t.Fatalf("ResolverConfig() mismatch: %#v", cfg)
	}
	if cfg.CacheTTL <= 0 || cfg.Timeout <= 0 {
		t.Fatalf("ResolverConfig() non-positive durations: %#v", cfg)
	}

	token, err := fixture.IssueToken(TokenOptions{
		Subject:     " user-1 ",
		Username:    " alice ",
		DisplayName: " Alice ",
		SessionID:   " session-1 ",
		ExpiresIn:   0,
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		t.Fatalf("ParseUnverified() error = %v", err)
	}
	if claims["sub"] != "user-1" || claims["username"] != "alice" || claims["sid"] != "session-1" {
		t.Fatalf("claims were not trimmed/defaulted as expected: %#v", claims)
	}

	recorder := httptest.NewRecorder()
	fixture.JWKSHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/jwks", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("JWKS status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var payload struct {
		Keys []struct {
			KTY string `json:"kty"`
			CRV string `json:"crv"`
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode JWKS: %v", err)
	}
	if len(payload.Keys) != 1 || payload.Keys[0].KTY != "OKP" || payload.Keys[0].CRV != "Ed25519" || payload.Keys[0].Kid != fixture.KeyID {
		t.Fatalf("JWKS payload mismatch: %#v", payload)
	}
}
