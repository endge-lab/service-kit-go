package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestClientCredentialsProviderCachesTokenAcrossConcurrentCalls(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		if err := request.ParseForm(); err != nil {
			t.Error(err)
		}
		if request.Form.Get("grant_type") != "client_credentials" || request.Form.Get("audience") != "workbench" {
			t.Errorf("unexpected form: %v", request.Form)
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"access_token": "token", "expires_in": 300})
	}))
	defer server.Close()
	provider, err := NewClientCredentialsProvider(ClientCredentialsConfig{
		TokenURL: server.URL, ClientID: "backend", ClientSecret: "development-fixture", Audience: "workbench",
	})
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for range 12 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if token, err := provider.Token(context.Background()); err != nil || token != "token" {
				t.Errorf("unexpected token result: %q %v", token, err)
			}
		}()
	}
	wait.Wait()
	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("expected one token request, got %d", requests)
	}
}
