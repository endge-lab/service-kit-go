package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientCredentialsConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Audience     string
	Scope        string
	Timeout      time.Duration
}

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type clientCredentialsProvider struct {
	config ClientCredentialsConfig
	client *http.Client
	mu     sync.Mutex
	token  string
	expiry time.Time
}

func NewClientCredentialsProvider(config ClientCredentialsConfig) (TokenProvider, error) {
	config.TokenURL = strings.TrimSpace(config.TokenURL)
	config.ClientID = strings.TrimSpace(config.ClientID)
	if config.TokenURL == "" || config.ClientID == "" || config.ClientSecret == "" {
		return nil, errors.New("client credentials token url, client id and client secret are required")
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	return &clientCredentialsProvider{config: config, client: &http.Client{Timeout: config.Timeout}}, nil
}

func (p *clientCredentialsProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Until(p.expiry) > 30*time.Second {
		return p.token, nil
	}
	values := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
	}
	if audience := strings.TrimSpace(p.config.Audience); audience != "" {
		values.Set("audience", audience)
	}
	if scope := strings.TrimSpace(p.config.Scope); scope != "" {
		values.Set("scope", scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("build client credentials request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request client credentials token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request client credentials token: unexpected status %d", resp.StatusCode)
	}
	var payload struct {
		AccessToken string          `json:"access_token"`
		TokenType   string          `json:"token_type"`
		ExpiresIn   json.RawMessage `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode client credentials token: %w", err)
	}
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	if payload.AccessToken == "" {
		return "", errors.New("client credentials response does not contain access_token")
	}
	expiresIn := parseExpiresIn(payload.ExpiresIn)
	p.token = payload.AccessToken
	p.expiry = time.Now().Add(expiresIn)
	return p.token, nil
}

func parseExpiresIn(raw json.RawMessage) time.Duration {
	if len(raw) == 0 {
		return 5 * time.Minute
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil && number > 0 {
		return time.Duration(number * float64(time.Second))
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if seconds, err := strconv.Atoi(text); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return 5 * time.Minute
}
