package auth_test

import (
	"context"
	"net/http/httptest"
	"testing"

	serviceauth "github.com/endge-lab/service-kit-go/pkg/auth"
	servicetesting "github.com/endge-lab/service-kit-go/pkg/testing"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

func TestResolverResolvesIdentity(t *testing.T) {
	t.Parallel()

	fixture, err := servicetesting.NewAuthFixture("https://auth.example.com", "example-platform")
	if err != nil {
		t.Fatalf("new fixture: %v", err)
	}

	server := httptest.NewServer(fixture.JWKSHandler())
	defer server.Close()

	token, err := fixture.IssueToken(servicetesting.TokenOptions{
		Subject:     "user-1",
		Username:    "alice",
		DisplayName: "Alice",
		Role:        "admin",
		SessionID:   "session-1",
		Contours: []serviceauth.ContourAccess{
			{Contour: "admin", Permissions: []string{"admin.users.read"}},
		},
		Permissions: []string{"admin.users.read"},
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	resolver := serviceauth.NewResolver(fixture.ResolverConfig(server.URL), noop.NewTracerProvider().Tracer("test"), zap.NewNop())
	identity, err := resolver.Resolve(context.Background(), token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if identity.AuthUserID != "user-1" {
		t.Fatalf("unexpected auth user id: %s", identity.AuthUserID)
	}
	if !serviceauth.HasPermission(identity, "admin.users.read") {
		t.Fatal("expected admin.users.read permission")
	}
}
