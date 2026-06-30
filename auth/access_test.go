package auth

import "testing"

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
