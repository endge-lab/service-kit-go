package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifierAcceptsRS256ServiceToken(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "https://issuer.example"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "rsa-1", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	verifier, err := NewVerifier(VerifierConfig{
		Issuer: issuer, JWKSURL: server.URL, Audience: "workbench",
		AllowedCallers: []string{"backend"}, AllowedAlgorithms: []string{"RS256"},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := claims{
		AuthorizedParty: "backend",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: "service-account-backend", Audience: jwt.ClaimStrings{"workbench"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "rsa-1"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := verifier.Verify(context.Background(), signed)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ClientID != "backend" || identity.Subject != "service-account-backend" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestVerifierRejectsUnexpectedCaller(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "https://issuer.example"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "rsa-1", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	verifier, err := NewVerifier(VerifierConfig{
		Issuer: issuer, JWKSURL: server.URL, Audience: "workbench",
		AllowedCallers: []string{"backend"}, AllowedAlgorithms: []string{"RS256"},
	})
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{
		AuthorizedParty: "hostile-service",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: "hostile", Audience: jwt.ClaimStrings{"workbench"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	token.Header["kid"] = "rsa-1"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(context.Background(), signed); err == nil {
		t.Fatal("expected unexpected caller to be rejected")
	}
}
