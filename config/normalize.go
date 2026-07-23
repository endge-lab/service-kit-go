package config

import (
	"strings"
	"time"
)

func (c *ServiceConfig) Normalize() {
	c.normalizeApp()
	c.normalizeHTTP()
	c.normalizeLogger()
	c.normalizeMetrics()
	c.normalizeRedis()
	c.normalizeAuth()
	c.normalizePostgres()
	c.normalizeTelemetry()
	c.normalizeRedpanda()
	c.normalizeTLS()
}

func (c *ServiceConfig) normalizeApp() {
	c.App.Env = NormalizeServiceEnvironment(c.App.Env)
	c.App.Name = strings.TrimSpace(c.App.Name)
	c.App.Version = strings.TrimSpace(c.App.Version)
	c.App.PublicURL = strings.TrimRight(strings.TrimSpace(c.App.PublicURL), "/")
}

func (c *ServiceConfig) normalizeHTTP() {
	c.HTTP.Port = strings.TrimSpace(c.HTTP.Port)
	c.HTTP.CORSAllowedOrigins = strings.TrimSpace(c.HTTP.CORSAllowedOrigins)
}

func (c *ServiceConfig) normalizeLogger() {
	c.Logger.Level = strings.TrimSpace(c.Logger.Level)
	c.Logger.OpenSearch.Endpoint = strings.TrimRight(strings.TrimSpace(c.Logger.OpenSearch.Endpoint), "/")
	c.Logger.OpenSearch.Index = strings.TrimSpace(c.Logger.OpenSearch.Index)
	c.Logger.OpenSearch.Username = strings.TrimSpace(c.Logger.OpenSearch.Username)
	c.Logger.OpenSearch.Password = strings.TrimSpace(c.Logger.OpenSearch.Password)
	if c.Logger.OpenSearch.FlushInterval <= 0 {
		c.Logger.OpenSearch.FlushInterval = time.Second
	}
	if c.Logger.OpenSearch.BatchSize <= 0 {
		c.Logger.OpenSearch.BatchSize = 100
	}
	if c.Logger.OpenSearch.QueueSize <= 0 {
		c.Logger.OpenSearch.QueueSize = 1000
	}
	if c.Logger.OpenSearch.RequestTimeout <= 0 {
		c.Logger.OpenSearch.RequestTimeout = 5 * time.Second
	}
}

func (c *ServiceConfig) normalizeMetrics() {
	c.Metrics.BindAddress = strings.TrimSpace(c.Metrics.BindAddress)
	c.Metrics.HandlerPath = ensureLeadingSlash(strings.TrimSpace(c.Metrics.HandlerPath))
}

func (c *ServiceConfig) normalizeRedis() {
	c.Redis.Host = strings.TrimSpace(c.Redis.Host)
	c.Redis.Username = strings.TrimSpace(c.Redis.Username)
	c.Redis.Password = strings.TrimSpace(c.Redis.Password)
	if c.Redis.Port <= 0 {
		c.Redis.Port = 6379
	}
}

func (c *ServiceConfig) normalizeAuth() {
	c.Auth.ServiceURL = strings.TrimRight(strings.TrimSpace(c.Auth.ServiceURL), "/")
	c.Auth.Issuer = strings.TrimRight(strings.TrimSpace(c.Auth.Issuer), "/")
	c.Auth.JWKSPath = ensureLeadingSlash(strings.TrimSpace(c.Auth.JWKSPath))
	c.Auth.AllowedAudiences = strings.TrimSpace(c.Auth.AllowedAudiences)
	if c.Auth.Timeout <= 0 {
		c.Auth.Timeout = 5 * time.Second
	}
	if c.Auth.JWKSCacheTTL <= 0 {
		c.Auth.JWKSCacheTTL = 5 * time.Minute
	}
}

func (c *ServiceConfig) normalizePostgres() {
	c.Postgres.Host = strings.TrimSpace(c.Postgres.Host)
	c.Postgres.User = strings.TrimSpace(c.Postgres.User)
	c.Postgres.Password = strings.TrimSpace(c.Postgres.Password)
	c.Postgres.Database = strings.TrimSpace(c.Postgres.Database)
	c.Postgres.Schema = strings.TrimSpace(c.Postgres.Schema)
	c.Postgres.SSLMode = strings.TrimSpace(c.Postgres.SSLMode)
	if c.Postgres.Port <= 0 {
		c.Postgres.Port = 5432
	}
	if c.Postgres.ConnTimeout <= 0 {
		c.Postgres.ConnTimeout = 5
	}
	if c.Postgres.MaxConn <= 0 {
		c.Postgres.MaxConn = 100
	}
	if c.Postgres.MaxConnLifetime <= 0 {
		c.Postgres.MaxConnLifetime = 25 * time.Minute
	}
	if c.Postgres.MaxConnIdleTime <= 0 {
		c.Postgres.MaxConnIdleTime = 5 * time.Minute
	}
}

func (c *ServiceConfig) normalizeTelemetry() {
	c.Telemetry.OTLPEndpoint = strings.TrimSpace(c.Telemetry.OTLPEndpoint)
}

func (c *ServiceConfig) normalizeRedpanda() {
	c.Redpanda.Brokers = strings.TrimSpace(c.Redpanda.Brokers)
	c.Redpanda.ClientID = strings.TrimSpace(c.Redpanda.ClientID)
	if c.Redpanda.DialTimeout <= 0 {
		c.Redpanda.DialTimeout = 5 * time.Second
	}
	if c.Redpanda.ReadBatchTimeout <= 0 {
		c.Redpanda.ReadBatchTimeout = 2 * time.Second
	}
	if c.Redpanda.WriteTimeout <= 0 {
		c.Redpanda.WriteTimeout = 10 * time.Second
	}
	if c.Redpanda.ClientID == "" {
		c.Redpanda.ClientID = c.App.Name
	}
}

func (c *ServiceConfig) normalizeTLS() {
	c.TLS.CertFile = strings.TrimSpace(c.TLS.CertFile)
	c.TLS.KeyFile = strings.TrimSpace(c.TLS.KeyFile)
	c.TLS.CAFile = strings.TrimSpace(c.TLS.CAFile)
}

func NormalizeServiceEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "dev", "development":
		return "development"
	case "local", "dev.local", "development.local", "local.development":
		return "local"
	case "prod", "production":
		return "production"
	default:
		return strings.TrimSpace(value)
	}
}

func ensureLeadingSlash(value string) string {
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}
