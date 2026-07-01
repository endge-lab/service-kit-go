package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadServiceConfigSelectsEnvironmentYAML(t *testing.T) {
	tests := []struct {
		appEnv string
		want   string
	}{
		{appEnv: "development", want: "development-service"},
		{appEnv: "local", want: "local-service"},
		{appEnv: "production", want: "production-service"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.appEnv, func(t *testing.T) {
			clearConfigEnv(t)
			dir := t.TempDir()
			t.Chdir(dir)
			writeServiceConfigFile(t, dir, "development", serviceConfigYAML("development-service", "development"))
			writeServiceConfigFile(t, dir, "local", serviceConfigYAML("local-service", "local"))
			writeServiceConfigFile(t, dir, "production", serviceConfigYAML("production-service", "production"))
			t.Setenv("APP_ENV", tt.appEnv)

			cfg, err := LoadServiceConfig()
			if err != nil {
				t.Fatalf("LoadServiceConfig() error = %v", err)
			}

			if cfg.App.Name != tt.want {
				t.Fatalf("App.Name = %q, want %q", cfg.App.Name, tt.want)
			}
			if cfg.App.Env != NormalizeServiceEnvironment(tt.appEnv) {
				t.Fatalf("App.Env = %q, want %q", cfg.App.Env, NormalizeServiceEnvironment(tt.appEnv))
			}
		})
	}
}

func TestLoadServiceConfigEnvOverridesYAML(t *testing.T) {
	clearConfigEnv(t)
	configPath := writeConfigFile(t, serviceConfigYAML("yaml-service", "development"))
	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("APP_NAME", "env-service")
	t.Setenv("REST_PORT", "9090")
	t.Setenv("LOGGER_LEVEL", "warn")
	t.Setenv("POSTGRES_HOST", "env-postgres")
	t.Setenv("POSTGRES_DATABASE", "env-db")
	t.Setenv("TLS_ENABLED", "true")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")

	cfg, err := LoadServiceConfig()
	if err != nil {
		t.Fatalf("LoadServiceConfig() error = %v", err)
	}

	if cfg.App.Name != "env-service" {
		t.Fatalf("App.Name = %q, want env-service", cfg.App.Name)
	}
	if cfg.HTTP.Port != "9090" {
		t.Fatalf("HTTP.Port = %q, want 9090", cfg.HTTP.Port)
	}
	if cfg.Logger.Level != "warn" {
		t.Fatalf("Logger.Level = %q, want warn", cfg.Logger.Level)
	}
	if cfg.Postgres.Host != "env-postgres" {
		t.Fatalf("Postgres.Host = %q, want env-postgres", cfg.Postgres.Host)
	}
	if cfg.Postgres.Database != "env-db" {
		t.Fatalf("Postgres.Database = %q, want env-db", cfg.Postgres.Database)
	}
	if !cfg.TLS.Enabled || cfg.TLS.CertFile != "/tmp/cert.pem" || cfg.TLS.KeyFile != "/tmp/key.pem" {
		t.Fatalf("TLS override mismatch: %#v", cfg.TLS)
	}
}

func TestServiceConfigNormalize(t *testing.T) {
	cfg := validServiceConfig()
	cfg.App.Env = " dev.local "
	cfg.App.Name = " service "
	cfg.App.PublicURL = " https://example.test/// "
	cfg.HTTP.CORSAllowedOrigins = " https://app.example.test "
	cfg.Auth.ServiceURL = " https://auth.example.test/ "
	cfg.Auth.Issuer = " https://issuer.example.test/ "
	cfg.Auth.JWKSPath = "jwks.json"
	cfg.Auth.Timeout = 0
	cfg.Auth.JWKSCacheTTL = 0
	cfg.Postgres.Port = 0
	cfg.Postgres.ConnTimeout = 0
	cfg.Postgres.MaxConn = 0
	cfg.Postgres.MaxConnLifetime = 0
	cfg.Postgres.MaxConnIdleTime = 0
	cfg.Redpanda.ClientID = ""
	cfg.Redpanda.DialTimeout = 0
	cfg.Redpanda.ReadBatchTimeout = 0
	cfg.Redpanda.WriteTimeout = 0
	cfg.TLS.CertFile = " cert.pem "

	cfg.Normalize()

	if cfg.App.Env != "local" {
		t.Fatalf("App.Env = %q, want local", cfg.App.Env)
	}
	if cfg.App.PublicURL != "https://example.test" {
		t.Fatalf("App.PublicURL = %q, want https://example.test", cfg.App.PublicURL)
	}
	if cfg.Auth.JWKSPath != "/jwks.json" {
		t.Fatalf("Auth.JWKSPath = %q, want /jwks.json", cfg.Auth.JWKSPath)
	}
	if cfg.Auth.Timeout != 5*time.Second || cfg.Auth.JWKSCacheTTL != 5*time.Minute {
		t.Fatalf("Auth defaults mismatch: timeout=%s ttl=%s", cfg.Auth.Timeout, cfg.Auth.JWKSCacheTTL)
	}
	if cfg.Postgres.Port != 5432 || cfg.Postgres.ConnTimeout != 5 || cfg.Postgres.MaxConn != 100 {
		t.Fatalf("Postgres defaults mismatch: %#v", cfg.Postgres)
	}
	if cfg.Redpanda.ClientID != "service" {
		t.Fatalf("Redpanda.ClientID = %q, want service", cfg.Redpanda.ClientID)
	}
	if cfg.TLS.CertFile != "cert.pem" {
		t.Fatalf("TLS.CertFile = %q, want cert.pem", cfg.TLS.CertFile)
	}
}

func TestServiceConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ServiceConfig)
		wantErr string
	}{
		{
			name: "missing app name",
			mutate: func(c *ServiceConfig) {
				c.App.Name = ""
			},
			wantErr: "config.app.name is required",
		},
		{
			name: "missing auth issuer",
			mutate: func(c *ServiceConfig) {
				c.Auth.Enabled = true
				c.Auth.ServiceURL = "https://auth.example.test"
				c.Auth.Issuer = ""
			},
			wantErr: "config.auth.issuer is required",
		},
		{
			name: "missing postgres database",
			mutate: func(c *ServiceConfig) {
				c.Postgres.URI = ""
				c.Postgres.Database = ""
			},
			wantErr: "config.postgres.database is required",
		},
		{
			name: "missing redis host",
			mutate: func(c *ServiceConfig) {
				c.Redis.Host = ""
			},
			wantErr: "config.redis.host is required",
		},
		{
			name: "missing telemetry endpoint",
			mutate: func(c *ServiceConfig) {
				c.Telemetry.Enabled = true
			},
			wantErr: "config.telemetry.otlp_endpoint is required",
		},
		{
			name: "missing redpanda brokers",
			mutate: func(c *ServiceConfig) {
				c.Redpanda.Enabled = true
			},
			wantErr: "config.redpanda.brokers is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := validServiceConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestServicePostgresDSNUsesURIWhenProvided(t *testing.T) {
	cfg := ServicePostgresConfig{
		URI:      "postgres://uri-user:uri-pass@uri-host:5432/uri-db?sslmode=require",
		Host:     "ignored",
		Port:     9999,
		User:     "ignored",
		Password: "ignored",
		Database: "ignored",
	}

	if got := cfg.DSN(); got != cfg.URI {
		t.Fatalf("DSN() = %q, want URI %q", got, cfg.URI)
	}
}

func TestServicePostgresDSNBuildsFromFields(t *testing.T) {
	cfg := ServicePostgresConfig{
		Host:        "postgres",
		Port:        5433,
		User:        "user",
		Password:    "password",
		Database:    "service_db",
		Schema:      "custom",
		SSLMode:     "require",
		ConnTimeout: 7,
	}

	want := "postgres://user:password@postgres:5433/service_db?connect_timeout=7&search_path=custom&sslmode=require"
	if got := cfg.DSN(); got != want {
		t.Fatalf("DSN() = %q, want %q", got, want)
	}
}

func TestServiceRedisAddr(t *testing.T) {
	cfg := ServiceRedisConfig{
		Host: "redis",
		Port: 6380,
	}

	if got := cfg.Addr(); got != "redis:6380" {
		t.Fatalf("Addr() = %q, want redis:6380", got)
	}
}

func TestServiceHTTPBindAddr(t *testing.T) {
	cfg := ServiceHTTPConfig{Port: "9090"}

	if got := cfg.BindAddr(); got != ":9090" {
		t.Fatalf("BindAddr() = %q, want :9090", got)
	}
}

func TestServiceTLSValidation(t *testing.T) {
	cfg := validServiceConfig()
	cfg.TLS.Enabled = true
	cfg.TLS.CertFile = ""
	cfg.TLS.KeyFile = "key.pem"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "config.tls.cert_file is required") {
		t.Fatalf("Validate() error = %v, want TLS_CERT_FILE error", err)
	}

	cfg.TLS.CertFile = "cert.pem"
	cfg.TLS.KeyFile = ""
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "config.tls.key_file is required") {
		t.Fatalf("Validate() error = %v, want TLS_KEY_FILE error", err)
	}

	cfg.TLS.KeyFile = "key.pem"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestServiceConfigGetters(t *testing.T) {
	cfg := validServiceConfig()
	cfg.TLS.Enabled = true
	cfg.TLS.CertFile = "cert.pem"
	cfg.TLS.KeyFile = "key.pem"

	var getter ServiceConfigGetter = cfg
	if getter.GetAppConfig().Name != cfg.App.Name {
		t.Fatalf("GetAppConfig() mismatch")
	}
	if getter.GetHTTPConfig().Port != cfg.HTTP.Port {
		t.Fatalf("GetHTTPConfig() mismatch")
	}
	if getter.GetPostgresConfig().Database != cfg.Postgres.Database {
		t.Fatalf("GetPostgresConfig() mismatch")
	}
	if getter.GetRedisConfig().Addr() != cfg.Redis.Addr() {
		t.Fatalf("GetRedisConfig() mismatch")
	}
	if !getter.GetTLSConfig().Enabled {
		t.Fatalf("GetTLSConfig().Enabled = false, want true")
	}
}

func TestLoadServiceConfigReturnsErrorWithoutFatal(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("CONFIG_PATH", writeConfigFile(t, `
app:
  name: ""
`))

	cfg, err := LoadServiceConfig()
	if err == nil {
		t.Fatalf("LoadServiceConfig() error = nil, want error; cfg = %#v", cfg)
	}
}

func validServiceConfig() ServiceConfig {
	cfg := ServiceConfig{
		App: ServiceAppConfig{
			Env:       "development",
			Name:      "service",
			Version:   "dev",
			PublicURL: "http://localhost:8080",
		},
		HTTP: ServiceHTTPConfig{
			Port:               "8080",
			CORSAllowedOrigins: "http://localhost:5173",
		},
		Logger: ServiceLoggerConfig{Level: "debug"},
		Redis: ServiceRedisConfig{
			Host:     "redis",
			Port:     6379,
			Username: "redis",
		},
		Postgres: ServicePostgresConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "postgres",
			Database:        "service",
			Schema:          "public",
			SSLMode:         "disable",
			ConnTimeout:     5,
			MaxConn:         100,
			MaxConnLifetime: 25 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		Auth: ServiceAuthConfig{
			JWKSPath:     "/.well-known/jwks.json",
			JWKSCacheTTL: 5 * time.Minute,
			Timeout:      5 * time.Second,
		},
		Redpanda: ServiceRedpandaConfig{
			DialTimeout:      5 * time.Second,
			ReadBatchTimeout: 2 * time.Second,
			WriteTimeout:     10 * time.Second,
		},
	}
	cfg.Normalize()
	return cfg
}

func serviceConfigYAML(name string, env string) string {
	return `
app:
  env: ` + env + `
  name: ` + name + `
  version: 0.1.0
  public_url: http://localhost:8080/
http:
  port: "8080"
  cors_allowed_origins: http://localhost:5173
logger:
  level: debug
metrics:
  enabled: false
  bind_address: ":9090"
  handler_path: /metrics
redis:
  host: redis
  port: 6379
  username: redis
  password: ""
  database: 0
postgres:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: service
  schema: public
  sslmode: disable
  conn_timeout: 5
  max_conn: 100
  max_conn_lifetime: 25m
  max_conn_idle_time: 5m
auth:
  enabled: false
  jwks_path: /.well-known/jwks.json
  jwks_cache_ttl: 5m
  timeout: 5s
telemetry:
  enabled: false
redpanda:
  enabled: false
  dial_timeout: 5s
  read_batch_timeout: 2s
  write_timeout: 10s
tls:
  enabled: false
`
}

func writeServiceConfigFile(t *testing.T, dir string, env string, body string) {
	t.Helper()

	configsDir := filepath.Join(dir, "configs")
	if err := os.MkdirAll(configsDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	path := filepath.Join(configsDir, env+".yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write service config file: %v", err)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, env := range []string{
		"CONFIG_PATH",
		"APP_ENV",
		"APP_NAME",
		"APP_VERSION",
		"PUBLIC_URL",
		"REST_PORT",
		"HTTP_PORT",
		"CORS_ALLOWED_ORIGINS",
		"LOGGER_LEVEL",
		"METRICS_ENABLED",
		"METRICS_BIND_ADDRESS",
		"METRICS_HANDLER_PATH",
		"GRPC_SERVER_PORT",
		"REDIS_HOST",
		"REDIS_PORT",
		"REDIS_USERNAME",
		"REDIS_PASSWORD",
		"REDIS_DATABASE",
		"DATABASE_URI",
		"POSTGRES_URI",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DATABASE",
		"POSTGRES_DB",
		"POSTGRES_SCHEMA",
		"POSTGRES_SSLMODE",
		"POSTGRES_CONNTIMEOUT",
		"POSTGRES_CONN_TIMEOUT",
		"POSTGRES_MAXCONN",
		"POSTGRES_MAX_CONN",
		"POSTGRES_MAXCONN_LIFETIME",
		"POSTGRES_MAX_CONN_LIFETIME",
		"POSTGRES_MAXCONN_IDLETIME",
		"POSTGRES_MAX_CONN_IDLE_TIME",
		"AUTH_ENABLED",
		"AUTH_SERVICE_URL",
		"AUTH_ISSUER",
		"AUTH_ALLOWED_AUDIENCES",
		"AUTH_JWKS_PATH",
		"AUTH_JWKS_CACHE_TTL",
		"AUTH_SERVICE_TIMEOUT",
		"AUTH_TIMEOUT",
		"TELEMETRY_ENABLED",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTLP_INSECURE",
		"REDPANDA_ENABLED",
		"REDPANDA_BROKERS",
		"REDPANDA_CLIENT_ID",
		"REDPANDA_DIAL_TIMEOUT",
		"REDPANDA_READ_BATCH_TIMEOUT",
		"REDPANDA_WRITE_TIMEOUT",
		"TLS_ENABLED",
		"TLS_CERT_FILE",
		"TLS_KEY_FILE",
		"TLS_CA_FILE",
		"TLS_INSECURE_SKIP_VERIFY",
		"CERT_FILE",
		"KEY_FILE",
		"CA_FILE",
		"RATE_LIMIT",
		"BURST_LIMIT",
	} {
		t.Setenv(env, "")
	}
}

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}
