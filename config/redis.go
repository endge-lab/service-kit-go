package config

import "fmt"

type ServiceRedisConfigGetter interface {
	GetHost() string
	GetPort() int
	GetUsername() string
	GetPassword() string
	GetDatabase() int
	Addr() string
}

type ServiceRedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

func (c ServiceRedisConfig) GetHost() string {
	return c.Host
}

func (c ServiceRedisConfig) GetPort() int {
	return c.Port
}

func (c ServiceRedisConfig) GetUsername() string {
	return c.Username
}

func (c ServiceRedisConfig) GetPassword() string {
	return c.Password
}

func (c ServiceRedisConfig) GetDatabase() int {
	return c.Database
}

func (c ServiceRedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
