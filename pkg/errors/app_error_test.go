package errors

import (
	stderrors "errors"
	"testing"
)

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

func TestAppErrorMethodsAndWrapping(t *testing.T) {
	t.Parallel()

	cause := stderrors.New("storage failed")
	err := Wrap(cause, "repo.failed", "safe message", 503)

	if got := err.Error(); got != "repo.failed: storage failed" {
		t.Fatalf("Error() = %q, want wrapped text", got)
	}
	if !stderrors.Is(err, cause) {
		t.Fatal("errors.Is(err, cause) = false, want true")
	}
	var appErr *AppError
	if !stderrors.As(err, &appErr) {
		t.Fatal("errors.As(err, *AppError) = false, want true")
	}
	if appErr.Code() != "repo.failed" || appErr.SafeMessage() != "safe message" || appErr.HTTPStatus() != 503 {
		t.Fatalf("unexpected app error fields: code=%q message=%q status=%d", appErr.Code(), appErr.SafeMessage(), appErr.HTTPStatus())
	}
}

func TestAppErrorNilAndFallbacks(t *testing.T) {
	t.Parallel()

	var appErr *AppError
	if appErr.Error() != "" || appErr.Unwrap() != nil || appErr.Code() != "" || appErr.SafeMessage() != "" || appErr.HTTPStatus() != 0 || appErr.Details() != nil {
		t.Fatal("nil AppError methods returned non-zero values")
	}

	plain := stderrors.New("plain")
	if CodeOf(plain) != ErrInternal.Code() {
		t.Fatalf("CodeOf(plain) = %q, want internal", CodeOf(plain))
	}
	if SafeMessageOf(plain) != ErrInternal.SafeMessage() {
		t.Fatalf("SafeMessageOf(plain) = %q, want internal safe message", SafeMessageOf(plain))
	}
	if HTTPStatusOf(plain) != ErrInternal.HTTPStatus() {
		t.Fatalf("HTTPStatusOf(plain) = %d, want internal status", HTTPStatusOf(plain))
	}
	if DetailsOf(plain) != nil {
		t.Fatalf("DetailsOf(plain) = %#v, want nil", DetailsOf(plain))
	}
}

func TestConstructorsMapToSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		target error
		status int
	}{
		{name: "invalid", err: InvalidInput("invalid", "bad"), target: ErrInvalidInput, status: 400},
		{name: "unauthorized", err: Unauthorized("unauthorized", "auth"), target: ErrUnauthorized, status: 401},
		{name: "forbidden", err: Forbidden("forbidden", "deny"), target: ErrForbidden, status: 403},
		{name: "not found", err: NotFound("missing", "missing"), target: ErrNotFound, status: 404},
		{name: "conflict", err: Conflict("conflict", "conflict"), target: ErrConflict, status: 409},
		{name: "internal", err: Internal("internal", "internal"), target: ErrInternal, status: 500},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !stderrors.Is(tt.err, tt.target) {
				t.Fatalf("errors.Is(%v, %v) = false", tt.err, tt.target)
			}
			if got := HTTPStatusOf(tt.err); got != tt.status {
				t.Fatalf("HTTPStatusOf() = %d, want %d", got, tt.status)
			}
		})
	}
}

func TestWithDetailsClonesAndWrapsPlainErrors(t *testing.T) {
	t.Parallel()

	details := map[string]any{"field": "title"}
	err := WithDetails(stderrors.New("boom"), details)
	details["field"] = "mutated"

	if got := CodeOf(err); got != "common.internal" {
		t.Fatalf("CodeOf() = %q, want common.internal", got)
	}
	gotDetails := DetailsOf(err)
	if gotDetails["field"] != "title" {
		t.Fatalf("DetailsOf() = %#v, want cloned title", gotDetails)
	}
	gotDetails["field"] = "caller-mutated"
	if DetailsOf(err)["field"] != "title" {
		t.Fatalf("Details() returned mutable internal map: %#v", DetailsOf(err))
	}

	if unchanged := WithDetails(ErrNotFound, nil); unchanged != ErrNotFound {
		t.Fatal("WithDetails(empty) should return original error")
	}
}
