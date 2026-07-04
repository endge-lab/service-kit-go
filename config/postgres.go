package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ServicePostgresConfigGetter interface {
	GetURI() string
	GetUser() string
	GetPassword() string
	GetDatabase() string
	GetHost() string
	GetPort() int
	GetSSLMode() string
	GetConnTimeout() int
	GetMaxConn() int
	GetMinConnLifeTime() time.Duration
	GetMaxConnIdleTime() time.Duration
	GetSchema() string
	GetMigrationsEnabled() bool
	DSN() string
}

type ServicePostgresConfig struct {
	URI               string        `mapstructure:"uri"`
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	Database          string        `mapstructure:"database"`
	Schema            string        `mapstructure:"schema"`
	SSLMode           string        `mapstructure:"sslmode"`
	ConnTimeout       int           `mapstructure:"conn_timeout"`
	MaxConn           int           `mapstructure:"max_conn"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	MigrationsEnabled bool          `mapstructure:"migrations_enabled"`
}

func (c ServicePostgresConfig) GetURI() string {
	return c.URI
}

func (c ServicePostgresConfig) GetUser() string {
	return c.User
}

func (c ServicePostgresConfig) GetPassword() string {
	return c.Password
}

func (c ServicePostgresConfig) GetHost() string {
	return c.Host
}

func (c ServicePostgresConfig) GetPort() int {
	return c.Port
}

func (c ServicePostgresConfig) GetDatabase() string {
	return c.Database
}

func (c ServicePostgresConfig) GetSchema() string {
	return c.Schema
}

func (c ServicePostgresConfig) GetMigrationsEnabled() bool {
	return c.MigrationsEnabled
}

func (c ServicePostgresConfig) GetSSLMode() string {
	return c.SSLMode
}

func (c ServicePostgresConfig) GetConnTimeout() int {
	return c.ConnTimeout
}

func (c ServicePostgresConfig) GetMaxConn() int {
	return c.MaxConn
}

func (c ServicePostgresConfig) GetMinConnLifeTime() time.Duration {
	return c.MaxConnLifetime
}

func (c ServicePostgresConfig) GetMaxConnIdleTime() time.Duration {
	return c.MaxConnIdleTime
}

func (c ServicePostgresConfig) DSN() string {
	if strings.TrimSpace(c.URI) != "" {
		return strings.TrimSpace(c.URI)
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Database,
	}

	query := dsn.Query()
	query.Set("sslmode", c.SSLMode)
	query.Set("connect_timeout", strconv.Itoa(c.ConnTimeout))
	if c.Schema != "" {
		query.Set("search_path", c.Schema)
	}
	dsn.RawQuery = query.Encode()

	return dsn.String()
}
