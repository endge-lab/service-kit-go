package httpkit

import (
	"context"
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "no query", raw: "/health", want: "/health"},
		{name: "strips query", raw: "/health?full=true", want: "/health"},
		{name: "query at start", raw: "?token=secret", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SanitizeURL(tt.raw); got != tt.want {
				t.Fatalf("SanitizeURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	ctx := WithRequestID(context.Background(), "req-1")
	if requestID, ok := RequestIDFromContext(ctx); !ok || requestID != "req-1" {
		t.Fatalf("unexpected request id: %q, ok=%v", requestID, ok)
	}
}

func TestContextValuesTrimAndMissing(t *testing.T) {
	t.Parallel()

	if value, ok := RequestIDFromContext(context.Background()); ok || value != "" {
		t.Fatalf("missing request id = %q, %v; want empty false", value, ok)
	}
	if value, ok := UserIDFromContext(context.Background()); ok || value != "" {
		t.Fatalf("missing user id = %q, %v; want empty false", value, ok)
	}
	if value, ok := SessionIDFromContext(context.Background()); ok || value != "" {
		t.Fatalf("missing session id = %q, %v; want empty false", value, ok)
	}

	ctx := WithRequestID(context.Background(), " req-1 ")
	if value, _ := RequestIDFromContext(ctx); value != "req-1" {
		t.Fatalf("trimmed request id = %q, want req-1", value)
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
