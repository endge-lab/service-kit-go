package config

import (
	"strings"
	"time"
)

type ServiceIdentityConfigGetter interface {
	GetServiceIdentityConfig() ServiceIdentityConfig
}

// ServiceIdentityConfig contains independent inbound verifier and outbound
// client-credentials settings. A service may enable either side or both.
type ServiceIdentityConfig struct {
	Verifier ServiceIdentityVerifierConfig `mapstructure:"verifier"`
	Client   ServiceIdentityClientConfig   `mapstructure:"client"`
}

type ServiceIdentityVerifierConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Issuer            string        `mapstructure:"issuer"`
	JWKSURL           string        `mapstructure:"jwks_url"`
	Audience          string        `mapstructure:"audience"`
	AllowedCallers    string        `mapstructure:"allowed_callers"`
	AllowedAlgorithms string        `mapstructure:"allowed_algorithms"`
	JWKSCacheTTL      time.Duration `mapstructure:"jwks_cache_ttl"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

func (c ServiceIdentityVerifierConfig) CallerList() []string {
	return splitCommaSeparated(c.AllowedCallers)
}

func (c ServiceIdentityVerifierConfig) AlgorithmList() []string {
	return splitCommaSeparated(c.AllowedAlgorithms)
}

type ServiceIdentityClientConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	TokenURL     string        `mapstructure:"token_url"`
	ClientID     string        `mapstructure:"client_id"`
	ClientSecret string        `mapstructure:"client_secret"`
	Audience     string        `mapstructure:"audience"`
	Scope        string        `mapstructure:"scope"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
