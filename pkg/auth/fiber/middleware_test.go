package fiber

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	serviceauth "github.com/endge-lab/service-kit-go/pkg/auth"
	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/endge-lab/service-kit-go/pkg/httpkit"
	gofiber "github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type fakeResolver struct {
	identity *serviceauth.Identity
	err      error
	token    string
	calls    int
}

func (r *fakeResolver) Resolve(ctx context.Context, token string) (*serviceauth.Identity, error) {
	r.calls++
	r.token = token
	return r.identity, r.err
}

func TestMiddlewareAuthenticateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		header     string
		query      string
		allowQuery bool
		resolver   *fakeResolver
		wantErr    string
		wantToken  string
	}{
		{
			name:      "missing token",
			resolver:  &fakeResolver{},
			wantErr:   "auth.access_token_required",
			wantToken: "",
		},
		{
			name:      "invalid bearer prefix",
			header:    "Basic abc",
			resolver:  &fakeResolver{},
			wantErr:   "auth.access_token_required",
			wantToken: "",
		},
		{
			name:      "resolver unauthorized",
			header:    "Bearer bad",
			resolver:  &fakeResolver{err: serviceauth.ErrUnauthorized},
			wantErr:   "auth.invalid_access_token",
			wantToken: "bad",
		},
		{
			name:      "resolver unexpected error",
			header:    "Bearer token",
			resolver:  &fakeResolver{err: stderrors.New("auth unavailable")},
			wantErr:   "auth.service_unavailable",
			wantToken: "token",
		},
		{
			name:      "identity missing subject",
			header:    "Bearer token",
			resolver:  &fakeResolver{identity: &serviceauth.Identity{}},
			wantErr:   "auth.identity_missing",
			wantToken: "token",
		},
		{
			name:       "query token wins when allowed",
			header:     "Bearer header-token",
			query:      "query-token",
			allowQuery: true,
			resolver: &fakeResolver{identity: &serviceauth.Identity{
				AuthUserID: "user-1",
				SessionID:  "session-1",
			}},
			wantToken: "query-token",
		},
		{
			name:   "header token success",
			header: "Bearer header-token",
			resolver: &fakeResolver{identity: &serviceauth.Identity{
				AuthUserID: " user-1 ",
				SessionID:  " session-1 ",
			}},
			wantToken: "header-token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handlerCalled bool
			app := gofiber.New(gofiber.Config{ErrorHandler: func(c *gofiber.Ctx, err error) error {
				return c.Status(serviceerrors.HTTPStatusOf(err)).JSON(httpkit.ErrorResponse{Code: serviceerrors.CodeOf(err)})
			}})
			middleware := NewMiddleware(tt.resolver, zap.NewNop())
			app.Get("/test", middleware.Authenticate(tt.allowQuery), func(c *gofiber.Ctx) error {
				handlerCalled = true
				if userID, ok := GetUserID(c.UserContext()); !ok || userID != "user-1" {
					t.Fatalf("GetUserID() = %q/%v, want user-1/true", userID, ok)
				}
				if userID, ok := httpkit.UserIDFromContext(c.UserContext()); !ok || userID != "user-1" {
					t.Fatalf("httpkit.UserIDFromContext() = %q/%v, want user-1/true", userID, ok)
				}
				if sessionID, ok := GetSessionID(c.UserContext()); !ok || sessionID != "session-1" {
					t.Fatalf("GetSessionID() = %q/%v, want session-1/true", sessionID, ok)
				}
				if identity, ok := GetIdentity(c.UserContext()); !ok || identity != tt.resolver.identity {
					t.Fatalf("GetIdentity() = %#v/%v, want resolver identity", identity, ok)
				}
				return c.SendStatus(gofiber.StatusNoContent)
			})

			target := "/test"
			if tt.query != "" {
				target += "?access_token=" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			if tt.header != "" {
				req.Header.Set(gofiber.HeaderAuthorization, tt.header)
			}
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if tt.wantErr != "" {
				if resp.StatusCode < 400 {
					t.Fatalf("status = %d, want error for %s", resp.StatusCode, tt.wantErr)
				}
				if tt.resolver.token != tt.wantToken {
					t.Fatalf("resolver token = %q, want %q", tt.resolver.token, tt.wantToken)
				}
				if handlerCalled {
					t.Fatal("handler was called on auth error")
				}
				return
			}
			if resp.StatusCode != gofiber.StatusNoContent {
				t.Fatalf("status = %d, want 204", resp.StatusCode)
			}
			if tt.resolver.token != tt.wantToken {
				t.Fatalf("resolver token = %q, want %q", tt.resolver.token, tt.wantToken)
			}
			if !handlerCalled {
				t.Fatal("handler was not called on successful auth")
			}
		})
	}
}

func TestMiddlewareHandlersAndRequirements(t *testing.T) {
	t.Parallel()

	app := gofiber.New(gofiber.Config{ErrorHandler: func(c *gofiber.Ctx, err error) error {
		return c.Status(serviceerrors.HTTPStatusOf(err)).JSON(httpkit.ErrorResponse{Code: serviceerrors.CodeOf(err)})
	}})
	resolver := &fakeResolver{identity: &serviceauth.Identity{
		AuthUserID:  "user-1",
		Permissions: []string{"todos.read"},
		Contours:    []serviceauth.ContourAccess{{Contour: "admin"}},
		Groups:      []string{"users"},
		SessionID:   "session-1",
		DisplayName: "Alice",
		Role:        "user",
		Username:    "alice",
		Rules:       map[string][]string{"todos": {"read"}},
	}}
	middleware := NewMiddleware(resolver, zap.NewNop())
	app.Get("/private", middleware.Authenticate(false), RequirePermission("todos.read"), func(c *gofiber.Ctx) error {
		return c.SendStatus(gofiber.StatusNoContent)
	})
	app.Get("/optional", middleware.Optional(false), func(c *gofiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/contour", middleware.Authenticate(false), RequireContour("admin"), func(c *gofiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/denied", middleware.Authenticate(false), RequirePermission("todos.write"), func(c *gofiber.Ctx) error {
		return c.SendString("unreachable")
	})

	privateReq := httptest.NewRequest(http.MethodGet, "/private", nil)
	privateReq.Header.Set(gofiber.HeaderAuthorization, "Bearer token")
	privateResp, err := app.Test(privateReq, -1)
	if err != nil {
		t.Fatalf("private app.Test() error = %v", err)
	}
	if privateResp.StatusCode != gofiber.StatusNoContent {
		t.Fatalf("private status = %d, want 204", privateResp.StatusCode)
	}

	optionalResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/optional", nil), -1)
	if err != nil {
		t.Fatalf("optional app.Test() error = %v", err)
	}
	if optionalResp.StatusCode != gofiber.StatusOK {
		t.Fatalf("optional status = %d, want 200", optionalResp.StatusCode)
	}

	contourReq := httptest.NewRequest(http.MethodGet, "/contour", nil)
	contourReq.Header.Set(gofiber.HeaderAuthorization, "Bearer token")
	contourResp, err := app.Test(contourReq, -1)
	if err != nil {
		t.Fatalf("contour app.Test() error = %v", err)
	}
	if contourResp.StatusCode != gofiber.StatusOK {
		t.Fatalf("contour status = %d, want 200", contourResp.StatusCode)
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/denied", nil)
	deniedReq.Header.Set(gofiber.HeaderAuthorization, "Bearer token")
	deniedResp, err := app.Test(deniedReq, -1)
	if err != nil {
		t.Fatalf("denied app.Test() error = %v", err)
	}
	if deniedResp.StatusCode != gofiber.StatusForbidden {
		t.Fatalf("denied status = %d, want 403", deniedResp.StatusCode)
	}
}

func TestRequireHandlersWithoutIdentity(t *testing.T) {
	t.Parallel()

	app := gofiber.New(gofiber.Config{ErrorHandler: func(c *gofiber.Ctx, err error) error {
		return c.Status(serviceerrors.HTTPStatusOf(err)).JSON(httpkit.ErrorResponse{Code: serviceerrors.CodeOf(err)})
	}})
	app.Get("/contour", RequireContour("admin"), func(c *gofiber.Ctx) error { return c.SendString("unreachable") })
	app.Get("/permission", RequirePermission("admin.read"), func(c *gofiber.Ctx) error { return c.SendString("unreachable") })

	for _, path := range []string{"/contour", "/permission"} {
		path := path
		t.Run(path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil), -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if resp.StatusCode != gofiber.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}
