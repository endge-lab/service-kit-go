package errors

import "testing"

func TestCodeOfReturnsWrappedCode(t *testing.T) {
	err := InvalidInput("todo.invalid_title", "Некорректный title")

	if got := CodeOf(err); got != "todo.invalid_title" {
		t.Fatalf("CodeOf() = %q, want %q", got, "todo.invalid_title")
	}
	if got := HTTPStatusOf(err); got != 400 {
		t.Fatalf("HTTPStatusOf() = %d, want 400", got)
	}
}

func TestWithDetailsPreservesAppError(t *testing.T) {
	err := WithDetails(ErrForbidden, map[string]any{"permission": "admin.users.read"})

	if got := CodeOf(err); got != "common.forbidden" {
		t.Fatalf("CodeOf() = %q, want %q", got, "common.forbidden")
	}

	details := DetailsOf(err)
	if details["permission"] != "admin.users.read" {
		t.Fatalf("unexpected details: %+v", details)
	}
}
