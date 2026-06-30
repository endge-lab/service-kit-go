package httpkit

import (
	"context"
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	t.Parallel()

	if got := SanitizeURL("/health?full=true"); got != "/health" {
		t.Fatalf("unexpected sanitized url: %s", got)
	}
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	ctx := WithRequestID(context.Background(), "req-1")
	if requestID, ok := RequestIDFromContext(ctx); !ok || requestID != "req-1" {
		t.Fatalf("unexpected request id: %q, ok=%v", requestID, ok)
	}
}

func TestWithUserAndSessionID(t *testing.T) {
	t.Parallel()

	ctx := WithUserID(context.Background(), " user-1 ")
	ctx = WithSessionID(ctx, " session-1 ")

	if userID, ok := UserIDFromContext(ctx); !ok || userID != "user-1" {
		t.Fatalf("unexpected user id: %q, ok=%v", userID, ok)
	}
	if sessionID, ok := SessionIDFromContext(ctx); !ok || sessionID != "session-1" {
		t.Fatalf("unexpected session id: %q, ok=%v", sessionID, ok)
	}
}
