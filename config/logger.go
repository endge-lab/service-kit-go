package config

type ServiceLoggerConfigGetter interface {
	GetLoggerConfig() string
}

type ServiceLoggerConfig struct {
	Level string `mapstructure:"level"`
}

func (c ServiceLoggerConfig) GetLoggerConfig() string {
	return c.Level
}
