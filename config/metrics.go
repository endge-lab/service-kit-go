package config

type ServiceMetricsConfigGetter interface {
	GetEnabled() bool
	GetBindAddress() string
	GetHandlerPath() string
}

type ServiceMetricsConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	BindAddress string `mapstructure:"bind_address"`
	HandlerPath string `mapstructure:"handler_path"`
}

func (c ServiceMetricsConfig) GetEnabled() bool {
	return c.Enabled
}

func (c ServiceMetricsConfig) GetBindAddress() string {
	return c.BindAddress
}

func (c ServiceMetricsConfig) GetHandlerPath() string {
	return c.HandlerPath
}
