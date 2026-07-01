package config

type ServiceConfig struct {
	App       ServiceAppConfig       `mapstructure:"app"`
	HTTP      ServiceHTTPConfig      `mapstructure:"http"`
	Logger    ServiceLoggerConfig    `mapstructure:"logger"`
	Metrics   ServiceMetricsConfig   `mapstructure:"metrics"`
	Redis     ServiceRedisConfig     `mapstructure:"redis"`
	Postgres  ServicePostgresConfig  `mapstructure:"postgres"`
	Auth      ServiceAuthConfig      `mapstructure:"auth"`
	Telemetry ServiceTelemetryConfig `mapstructure:"telemetry"`
	Redpanda  ServiceRedpandaConfig  `mapstructure:"redpanda"`
	TLS       ServiceTLSConfig       `mapstructure:"tls"`
}

type ServiceConfigGetter interface {
	GetAppConfig() ServiceAppConfig
	GetHTTPConfig() ServiceHTTPConfig
	GetLoggerConfig() ServiceLoggerConfig
	GetMetricsConfig() ServiceMetricsConfig
	GetRedisConfig() ServiceRedisConfig
	GetPostgresConfig() ServicePostgresConfig
	GetAuthConfig() ServiceAuthConfig
	GetTelemetryConfig() ServiceTelemetryConfig
	GetRedpandaConfig() ServiceRedpandaConfig
	GetTLSConfig() ServiceTLSConfig
}

func (c ServiceConfig) GetAppConfig() ServiceAppConfig {
	return c.App
}

func (c ServiceConfig) GetHTTPConfig() ServiceHTTPConfig {
	return c.HTTP
}

func (c ServiceConfig) GetLoggerConfig() ServiceLoggerConfig {
	return c.Logger
}

func (c ServiceConfig) GetMetricsConfig() ServiceMetricsConfig {
	return c.Metrics
}

func (c ServiceConfig) GetRedisConfig() ServiceRedisConfig {
	return c.Redis
}

func (c ServiceConfig) GetPostgresConfig() ServicePostgresConfig {
	return c.Postgres
}

func (c ServiceConfig) GetAuthConfig() ServiceAuthConfig {
	return c.Auth
}

func (c ServiceConfig) GetTelemetryConfig() ServiceTelemetryConfig {
	return c.Telemetry
}

func (c ServiceConfig) GetRedpandaConfig() ServiceRedpandaConfig {
	return c.Redpanda
}

func (c ServiceConfig) GetTLSConfig() ServiceTLSConfig {
	return c.TLS
}
