package config

type TelemetryConfigGetter interface {
	GetEnabled() bool
	GetOTLPEndpoint() string
	GetOTLPInsecure() bool
}

type ServiceTelemetryConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	OTLPInsecure bool   `mapstructure:"otlp_insecure"`
}

func (c ServiceTelemetryConfig) GetEnabled() bool {
	return c.Enabled
}

func (c ServiceTelemetryConfig) GetOTLPEndpoint() string {
	return c.OTLPEndpoint
}

func (c ServiceTelemetryConfig) GetOTLPInsecure() bool {
	return c.OTLPInsecure
}
