package config

import (
	"strings"
	"time"
)

type AuthConfigGetter interface {
	GetEnabled() bool
	GetServiceURL() string
	GetIssuer() string
	GetAllowedAudiences() string
	GetJWKSPath() string
	GetJWKSCacheTTL() time.Duration
	GetTimeout() time.Duration
}

type ServiceAuthConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	ServiceURL       string        `mapstructure:"service_url"`
	Issuer           string        `mapstructure:"issuer"`
	AllowedAudiences string        `mapstructure:"allowed_audiences"`
	JWKSPath         string        `mapstructure:"jwks_path"`
	JWKSCacheTTL     time.Duration `mapstructure:"jwks_cache_ttl"`
	Timeout          time.Duration `mapstructure:"timeout"`
}

func (c ServiceAuthConfig) GetEnabled() bool {
	return c.Enabled
}

func (c ServiceAuthConfig) GetServiceURL() string {
	return c.ServiceURL
}

func (c ServiceAuthConfig) GetIssuer() string {
	return c.Issuer
}

func (c ServiceAuthConfig) GetAllowedAudiences() string {
	return c.AllowedAudiences
}

func (c ServiceAuthConfig) GetJWKSPath() string {
	return c.JWKSPath
}

func (c ServiceAuthConfig) GetJWKSCacheTTL() time.Duration {
	return c.JWKSCacheTTL
}

func (c ServiceAuthConfig) GetTimeout() time.Duration {
	return c.Timeout
}

func (c ServiceAuthConfig) JWKSURL() string {
	return strings.TrimRight(c.ServiceURL, "/") + ensureLeadingSlash(c.JWKSPath)
}
