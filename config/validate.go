package config

import "errors"

func (c ServiceConfig) Validate() error {
	if err := c.validateApp(); err != nil {
		return err
	}
	if err := c.validateHTTP(); err != nil {
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
