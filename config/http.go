package config

import "fmt"

type HTTPConfigGetter interface {
	GetPort() string
	GetCORSAllowedOrigins() string
	BindAddr() string
}

type ServiceHTTPConfig struct {
	Port               string `mapstructure:"port"`
	CORSAllowedOrigins string `mapstructure:"cors_allowed_origins"`
}

func (c ServiceHTTPConfig) GetPort() string {
	return c.Port
}

func (c ServiceHTTPConfig) GetCORSAllowedOrigins() string {
	return c.CORSAllowedOrigins
}

func (c ServiceHTTPConfig) BindAddr() string {
	return fmt.Sprintf(":%s", c.Port)
}
