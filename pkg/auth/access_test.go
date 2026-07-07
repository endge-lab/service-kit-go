package auth

import (
	"errors"
	"testing"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
)

func TestHasPermissionChecksContourPermissions(t *testing.T) {
	t.Parallel()

	identity := &Identity{
		Role: "admin",
		Contours: []ContourAccess{
			{
				Contour:     "admin",
				Permissions: []string{"admin.questions.update"},
			},
		},
	}

	if !HasPermission(identity, "admin.questions.update") {
		t.Fatal("expected permission from contour")
	}
	if HasPermission(identity, "admin.questions.delete") {
		t.Fatal("unexpected permission")
	}
}

func TestOwnerBypassesChecks(t *testing.T) {
	t.Parallel()

	identity := &Identity{Role: "owner"}
	if !HasContour(identity, "infra") {
		t.Fatal("owner must bypass contour checks")
	}
	if !HasPermission(identity, "admin.users.set-role") {
		t.Fatal("owner must bypass permission checks")
	}
}

func TestAccessHelpersHandleNilEmptyAndCase(t *testing.T) {
	t.Parallel()

	identity := &Identity{
		Role:        " user ",
		Groups:      []string{" Admins "},
		Permissions: []string{" Todos.Read "},
		Contours: []ContourAccess{{
			Contour:     " Projects ",
			Groups:      []string{" Editors "},
			Permissions: []string{" Projects.Write "},
		}},
	}

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "nil is not owner", got: IsOwner(nil), want: false},
		{name: "empty contour rejected", got: HasContour(identity, " "), want: false},
		{name: "nil contour rejected", got: HasContour(nil, "projects"), want: false},
		{name: "contour trims and ignores case", got: HasContour(identity, "projects"), want: true},
		{name: "flat group trims and ignores case", got: HasGroup(identity, "admins"), want: true},
		{name: "contour group trims and ignores case", got: HasGroup(identity, "editors"), want: true},
		{name: "empty group rejected", got: HasGroup(identity, ""), want: false},
		{name: "flat permission trims and ignores case", got: HasPermission(identity, "todos.read"), want: true},
		{name: "contour permission trims and ignores case", got: HasPermission(identity, "projects.write"), want: true},
		{name: "empty permission rejected", got: HasPermission(identity, " "), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestRequireAccessReturnsForbiddenAppError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "contour", err: RequireContour(&Identity{}, "admin"), code: "auth.contour_forbidden"},
		{name: "permission", err: RequirePermission(&Identity{}, "admin.read"), code: "auth.permission_forbidden"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err == nil {
				t.Fatal("error = nil, want forbidden")
			}
			if !errors.Is(tt.err, serviceerrors.ErrForbidden) {
				t.Fatalf("errors.Is(err, ErrForbidden) = false for %v", tt.err)
			}
			if got := serviceerrors.CodeOf(tt.err); got != tt.code {
				t.Fatalf("CodeOf() = %q, want %q", got, tt.code)
			}
		})
	}

	if err := RequireContour(&Identity{Role: "owner"}, "admin"); err != nil {
		t.Fatalf("owner RequireContour() error = %v, want nil", err)
	}
	if err := RequirePermission(&Identity{Permissions: []string{"admin.read"}}, "admin.read"); err != nil {
		t.Fatalf("RequirePermission() error = %v, want nil", err)
	}
}
