package config

import "github.com/spf13/viper"

func setServiceDefaults(v *viper.Viper, appEnv string) {
	setDefaults(v, map[string]any{
		"app.env":                     appEnv,
		"app.name":                    "",
		"app.version":                 "dev",
		"app.public_url":              "",
		"http.port":                   "8080",
		"http.cors_allowed_origins":   "",
		"logger.level":                "debug",
		"metrics.enabled":             false,
		"metrics.bind_address":        ":9090",
		"metrics.handler_path":        "/metrics",
		"grpc.port":                   50051,
		"redis.host":                  "redis",
		"redis.port":                  6379,
		"redis.username":              "redis",
		"redis.password":              "",
		"redis.database":              0,
		"postgres.uri":                "",
		"postgres.host":               "localhost",
		"postgres.port":               5432,
		"postgres.user":               "postgres",
		"postgres.password":           "",
		"postgres.database":           "",
		"postgres.schema":             "public",
		"postgres.sslmode":            "disable",
		"postgres.conn_timeout":       5,
		"postgres.max_conn":           100,
		"postgres.max_conn_lifetime":  "25m",
		"postgres.max_conn_idle_time": "5m",
		"postgres.migrations_enabled": false,
		"auth.enabled":                false,
		"auth.service_url":            "",
		"auth.issuer":                 "",
		"auth.allowed_audiences":      "",
		"auth.jwks_path":              "/.well-known/jwks.json",
		"auth.jwks_cache_ttl":         "5m",
		"auth.timeout":                "5s",
		"telemetry.enabled":           false,
		"telemetry.otlp_endpoint":     "",
		"telemetry.otlp_insecure":     false,
		"redpanda.enabled":            false,
		"redpanda.brokers":            "",
		"redpanda.client_id":          "",
		"redpanda.dial_timeout":       "5s",
		"redpanda.read_batch_timeout": "2s",
		"redpanda.write_timeout":      "10s",
		"tls.enabled":                 false,
		"tls.cert_file":               "",
		"tls.key_file":                "",
		"tls.ca_file":                 "",
		"tls.insecure_skip_verify":    false,
		"rate_limiter.rate_limit":     1,
		"rate_limiter.burst_limit":    2,
	})
}

func setDefaults(v *viper.Viper, defaults map[string]any) {
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}
