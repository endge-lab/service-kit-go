package testing

import "testing"

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
}
