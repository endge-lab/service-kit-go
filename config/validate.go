package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func (c ServiceConfig) Validate() error {
	if err := c.validateApp(); err != nil {
		return err
	}
	if err := c.validateLogger(); err != nil {
		return err
	}
	if err := c.validateHTTP(); err != nil {
		return err
	}
	if err := c.validateGRPC(); err != nil {
		return err
	}
	if err := c.validateMetrics(); err != nil {
		return err
	}
	if err := c.validateRedis(); err != nil {
		return err
	}
	if err := c.validateAuth(); err != nil {
		return err
	}
	if err := c.validatePostgres(); err != nil {
		return err
	}
	if err := c.validateTelemetry(); err != nil {
		return err
	}
	if err := c.validateRedpanda(); err != nil {
		return err
	}
	if err := c.validateTLS(); err != nil {
		return err
	}
	if err := c.validateServiceIdentity(); err != nil {
		return err
	}

	return nil
}

func (c ServiceConfig) validateGRPC() error {
	if !c.GRPC.Enabled {
		return nil
	}
	if c.GRPC.Port < 1 || c.GRPC.Port > 65535 {
		return errors.New("config.grpc.port must be a valid TCP port when config.grpc.enabled=true")
	}
	if c.GRPC.MaxReceiveBytes <= 0 || c.GRPC.MaxSendBytes <= 0 {
		return errors.New("config.grpc message limits must be positive when config.grpc.enabled=true")
	}
	if c.GRPC.KeepaliveTime <= 0 || c.GRPC.KeepaliveTimeout <= 0 {
		return errors.New("config.grpc keepalive durations must be positive when config.grpc.enabled=true")
	}
	return nil
}

func (c ServiceConfig) validateServiceIdentity() error {
	verifier := c.Identity.Verifier
	if verifier.Enabled {
		switch {
		case strings.TrimSpace(verifier.Issuer) == "":
			return errors.New("config.service_identity.verifier.issuer is required when enabled")
		case strings.TrimSpace(verifier.JWKSURL) == "":
			return errors.New("config.service_identity.verifier.jwks_url is required when enabled")
		case strings.TrimSpace(verifier.Audience) == "":
			return errors.New("config.service_identity.verifier.audience is required when enabled")
		case len(verifier.CallerList()) == 0:
			return errors.New("config.service_identity.verifier.allowed_callers is required when enabled")
		case len(verifier.AlgorithmList()) == 0:
			return errors.New("config.service_identity.verifier.allowed_algorithms is required when enabled")
		}
	}
	client := c.Identity.Client
	if client.Enabled {
		switch {
		case strings.TrimSpace(client.TokenURL) == "":
			return errors.New("config.service_identity.client.token_url is required when enabled")
		case strings.TrimSpace(client.ClientID) == "":
			return errors.New("config.service_identity.client.client_id is required when enabled")
		case strings.TrimSpace(client.ClientSecret) == "":
			return errors.New("config.service_identity.client.client_secret is required when enabled")
		}
	}
	return nil
}

func (c ServiceConfig) validateMetrics() error {
	if !c.Metrics.Enabled {
		return nil
	}

	address := strings.TrimSpace(c.Metrics.BindAddress)
	if address == "" {
		return errors.New("config.metrics.bind_address is required when config.metrics.enabled=true")
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return errors.New("config.metrics.bind_address must be a host:port when config.metrics.enabled=true")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("config.metrics.bind_address must contain a valid port when config.metrics.enabled=true")
	}

	path := strings.TrimSpace(c.Metrics.HandlerPath)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errors.New("config.metrics.handler_path must start with / when config.metrics.enabled=true")
	}

	return nil
}

func (c ServiceConfig) validateApp() error {
	switch {
	case c.App.Name == "":
		return errors.New("config.app.name is required")
	case c.App.PublicURL == "":
		return errors.New("config.app.public_url is required")
	default:
		return nil
	}
}

func (c ServiceConfig) validateHTTP() error {
	switch {
	case c.HTTP.Port == "":
		return errors.New("config.http.port is required")
	case c.HTTP.CORSAllowedOrigins == "":
		return errors.New("config.http.cors_allowed_origins is required")
	default:
		return nil
	}
}

func (c ServiceConfig) validateRedis() error {
	switch {
	case c.Redis.Host == "":
		return errors.New("config.redis.host is required")
	case c.Redis.Port <= 0:
		return errors.New("config.redis.port must be positive")
	default:
		return nil
	}
}

func (c ServiceConfig) validateAuth() error {
	switch {
	case c.Auth.Enabled && c.Auth.ServiceURL == "":
		return errors.New("config.auth.service_url is required when config.auth.enabled=true")
	case c.Auth.Enabled && c.Auth.Issuer == "":
		return errors.New("config.auth.issuer is required when config.auth.enabled=true")
	default:
		return nil
	}
}

func (c ServiceConfig) validatePostgres() error {
	if c.Postgres.Enabled != nil && !*c.Postgres.Enabled {
		return nil
	}
	switch {
	case c.Postgres.Host == "":
		return errors.New("config.postgres.host is required")
	case c.Postgres.Port <= 0:
		return errors.New("config.postgres.port must be positive")
	case c.Postgres.User == "":
		return errors.New("config.postgres.user is required")
	case c.Postgres.Database == "":
		return errors.New("config.postgres.database is required")
	case c.Postgres.SSLMode == "":
		return errors.New("config.postgres.sslmode is required")
	default:
		return nil
	}
}

func (c ServiceConfig) validateTelemetry() error {
	if c.Telemetry.Enabled && c.Telemetry.OTLPEndpoint == "" {
		return errors.New("config.telemetry.otlp_endpoint is required when config.telemetry.enabled=true")
	}

	return nil
}

func (c ServiceConfig) validateLogger() error {
	if !c.Logger.OpenSearch.Enabled {
		return nil
	}

	endpoint := c.Logger.OpenSearch.Endpoint
	if endpoint == "" {
		return errors.New("config.logger.opensearch.endpoint is required when config.logger.opensearch.enabled=true")
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("config.logger.opensearch.endpoint must be an absolute HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("config.logger.opensearch.endpoint must use http or https")
	}
	if c.Logger.OpenSearch.Index == "" {
		return errors.New("config.logger.opensearch.index is required when config.logger.opensearch.enabled=true")
	}

	return nil
}

func (c ServiceConfig) validateRedpanda() error {
	if c.Redpanda.Enabled && len(c.Redpanda.BrokerList()) == 0 {
		return errors.New("config.redpanda.brokers is required when config.redpanda.enabled=true")
	}

	return nil
}

func (c ServiceConfig) validateTLS() error {
	switch {
	case c.TLS.Enabled && c.TLS.CertFile == "":
		return errors.New("config.tls.cert_file is required when config.tls.enabled=true")
	case c.TLS.Enabled && c.TLS.KeyFile == "":
		return errors.New("config.tls.key_file is required when config.tls.enabled=true")
	default:
		return nil
	}
}
