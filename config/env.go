package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

func bindServiceEnv(v *viper.Viper) error {
	return bindEnv(v, map[string][]string{
		"app.env":                                      {"APP_ENV"},
		"app.name":                                     {"APP_NAME"},
		"app.version":                                  {"APP_VERSION"},
		"app.public_url":                               {"PUBLIC_URL"},
		"http.port":                                    {"REST_PORT", "HTTP_PORT"},
		"http.cors_allowed_origins":                    {"CORS_ALLOWED_ORIGINS"},
		"logger.level":                                 {"LOGGER_LEVEL"},
		"logger.opensearch.enabled":                    {"LOGGER_OPENSEARCH_ENABLED"},
		"logger.opensearch.endpoint":                   {"LOGGER_OPENSEARCH_ENDPOINT"},
		"logger.opensearch.index":                      {"LOGGER_OPENSEARCH_INDEX"},
		"logger.opensearch.username":                   {"LOGGER_OPENSEARCH_USERNAME"},
		"logger.opensearch.password":                   {"LOGGER_OPENSEARCH_PASSWORD"},
		"logger.opensearch.insecure_skip_verify":       {"LOGGER_OPENSEARCH_INSECURE_SKIP_VERIFY"},
		"logger.opensearch.flush_interval":             {"LOGGER_OPENSEARCH_FLUSH_INTERVAL"},
		"logger.opensearch.batch_size":                 {"LOGGER_OPENSEARCH_BATCH_SIZE"},
		"logger.opensearch.queue_size":                 {"LOGGER_OPENSEARCH_QUEUE_SIZE"},
		"logger.opensearch.request_timeout":            {"LOGGER_OPENSEARCH_REQUEST_TIMEOUT"},
		"metrics.enabled":                              {"METRICS_ENABLED"},
		"metrics.bind_address":                         {"METRICS_BIND_ADDRESS"},
		"metrics.handler_path":                         {"METRICS_HANDLER_PATH"},
		"grpc.enabled":                                 {"GRPC_SERVER_ENABLED"},
		"grpc.port":                                    {"GRPC_SERVER_PORT"},
		"grpc.max_receive_bytes":                       {"GRPC_SERVER_MAX_RECEIVE_BYTES"},
		"grpc.max_send_bytes":                          {"GRPC_SERVER_MAX_SEND_BYTES"},
		"grpc.graceful_stop_timeout":                   {"GRPC_SERVER_GRACEFUL_STOP_TIMEOUT"},
		"grpc.keepalive_time":                          {"GRPC_KEEPALIVE_TIME"},
		"grpc.keepalive_timeout":                       {"GRPC_KEEPALIVE_TIMEOUT"},
		"redis.host":                                   {"REDIS_HOST"},
		"redis.port":                                   {"REDIS_PORT"},
		"redis.username":                               {"REDIS_USERNAME"},
		"redis.password":                               {"REDIS_PASSWORD"},
		"redis.database":                               {"REDIS_DATABASE"},
		"postgres.enabled":                             {"POSTGRES_ENABLED"},
		"postgres.host":                                {"POSTGRES_HOST"},
		"postgres.port":                                {"POSTGRES_PORT"},
		"postgres.user":                                {"POSTGRES_USER"},
		"postgres.password":                            {"POSTGRES_PASSWORD"},
		"postgres.database":                            {"POSTGRES_DATABASE", "POSTGRES_DB"},
		"postgres.schema":                              {"POSTGRES_SCHEMA"},
		"postgres.sslmode":                             {"POSTGRES_SSLMODE"},
		"postgres.conn_timeout":                        {"POSTGRES_CONNTIMEOUT", "POSTGRES_CONN_TIMEOUT"},
		"postgres.max_conn":                            {"POSTGRES_MAXCONN", "POSTGRES_MAX_CONN"},
		"postgres.max_conn_lifetime":                   {"POSTGRES_MAXCONN_LIFETIME", "POSTGRES_MAX_CONN_LIFETIME"},
		"postgres.max_conn_idle_time":                  {"POSTGRES_MAXCONN_IDLETIME", "POSTGRES_MAX_CONN_IDLE_TIME"},
		"postgres.migrations_enabled":                  {"POSTGRES_MIGRATIONS_ENABLED", "MIGRATIONS_ENABLED"},
		"auth.enabled":                                 {"AUTH_ENABLED"},
		"auth.service_url":                             {"AUTH_SERVICE_URL"},
		"auth.issuer":                                  {"AUTH_ISSUER"},
		"auth.allowed_audiences":                       {"AUTH_ALLOWED_AUDIENCES"},
		"auth.jwks_path":                               {"AUTH_JWKS_PATH"},
		"auth.jwks_cache_ttl":                          {"AUTH_JWKS_CACHE_TTL"},
		"auth.timeout":                                 {"AUTH_SERVICE_TIMEOUT", "AUTH_TIMEOUT"},
		"telemetry.enabled":                            {"TELEMETRY_ENABLED"},
		"telemetry.otlp_endpoint":                      {"OTEL_EXPORTER_OTLP_ENDPOINT", "OTLP_ENDPOINT"},
		"telemetry.otlp_insecure":                      {"OTEL_EXPORTER_OTLP_INSECURE", "OTLP_INSECURE"},
		"redpanda.enabled":                             {"REDPANDA_ENABLED"},
		"redpanda.brokers":                             {"REDPANDA_BROKERS"},
		"redpanda.client_id":                           {"REDPANDA_CLIENT_ID"},
		"redpanda.dial_timeout":                        {"REDPANDA_DIAL_TIMEOUT"},
		"redpanda.read_batch_timeout":                  {"REDPANDA_READ_BATCH_TIMEOUT"},
		"redpanda.write_timeout":                       {"REDPANDA_WRITE_TIMEOUT"},
		"tls.enabled":                                  {"TLS_ENABLED"},
		"tls.cert_file":                                {"TLS_CERT_FILE", "CERT_FILE"},
		"tls.key_file":                                 {"TLS_KEY_FILE", "KEY_FILE"},
		"tls.ca_file":                                  {"TLS_CA_FILE", "CA_FILE"},
		"tls.insecure_skip_verify":                     {"TLS_INSECURE_SKIP_VERIFY"},
		"service_identity.verifier.enabled":            {"SERVICE_AUTH_VERIFIER_ENABLED"},
		"service_identity.verifier.issuer":             {"SERVICE_AUTH_ISSUER"},
		"service_identity.verifier.jwks_url":           {"SERVICE_AUTH_JWKS_URL"},
		"service_identity.verifier.audience":           {"SERVICE_AUTH_AUDIENCE"},
		"service_identity.verifier.allowed_callers":    {"SERVICE_AUTH_ALLOWED_CALLERS"},
		"service_identity.verifier.allowed_algorithms": {"SERVICE_AUTH_ALLOWED_ALGORITHMS"},
		"service_identity.verifier.jwks_cache_ttl":     {"SERVICE_AUTH_JWKS_CACHE_TTL"},
		"service_identity.verifier.timeout":            {"SERVICE_AUTH_TIMEOUT"},
		"service_identity.client.enabled":              {"SERVICE_AUTH_CLIENT_ENABLED"},
		"service_identity.client.token_url":            {"SERVICE_AUTH_TOKEN_URL"},
		"service_identity.client.client_id":            {"SERVICE_AUTH_CLIENT_ID"},
		"service_identity.client.client_secret":        {"SERVICE_AUTH_CLIENT_SECRET"},
		"service_identity.client.audience":             {"SERVICE_AUTH_CLIENT_AUDIENCE"},
		"service_identity.client.scope":                {"SERVICE_AUTH_CLIENT_SCOPE"},
		"service_identity.client.timeout":              {"SERVICE_AUTH_CLIENT_TIMEOUT"},
		"rate_limiter.rate_limit":                      {"RATE_LIMIT"},
		"rate_limiter.burst_limit":                     {"BURST_LIMIT"},
	})
}

func bindEnv(v *viper.Viper, bindings map[string][]string) error {
	for key, envs := range bindings {
		args := append([]string{key}, envs...)
		if err := v.BindEnv(args...); err != nil {
			return fmt.Errorf("bind env %s to %s: %w", strings.Join(envs, ","), key, err)
		}
	}

	return nil
}

func setServiceConfigFile(v *viper.Viper, appEnv string) {
	if configPath := strings.TrimSpace(os.Getenv("CONFIG_PATH")); configPath != "" {
		v.SetConfigFile(configPath)
		return
	}

	configName := NormalizeServiceEnvironment(appEnv)
	v.SetConfigName(configName)
	v.AddConfigPath("configs")
	v.AddConfigPath(".")

	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		v.AddConfigPath(filepath.Join(executableDir, "configs"))
		v.AddConfigPath(executableDir)
	}
}

func detectServiceAppEnv() string {
	for _, key := range []string{"APP_ENV", "GO_ENV", "ENVIRONMENT", "NODE_ENV"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return defaultServiceAppEnv
}

func preloadServiceEnvFiles() {
	appEnv := detectServiceAppEnv()
	searchDirs := []string{"."}
	seen := map[string]struct{}{}

	if executable, err := os.Executable(); err == nil {
		searchDirs = append(searchDirs, filepath.Dir(executable))
	}

	for _, dir := range searchDirs {
		for _, fileName := range serviceEnvFileCandidates(appEnv) {
			filePath := filepath.Join(dir, fileName)
			if _, exists := seen[filePath]; exists {
				continue
			}
			seen[filePath] = struct{}{}

			if _, err := os.Stat(filePath); err == nil {
				_ = gotenv.Load(filePath)
			}
		}
	}
}

func serviceEnvFileCandidates(appEnv string) []string {
	trimmedAppEnv := strings.TrimSpace(appEnv)

	switch NormalizeServiceEnvironment(appEnv) {
	case "development":
		return []string{".env.development.local", ".env.local", ".env.development", ".env"}
	case "local":
		return []string{".env.local", ".env.development.local", ".env.development", ".env"}
	case "production":
		return []string{".env.production", ".env.local", ".env"}
	default:
		if trimmedAppEnv == "" {
			return []string{".env.development.local", ".env.local", ".env.development", ".env"}
		}
		return []string{".env." + trimmedAppEnv + ".local", ".env.local", ".env." + trimmedAppEnv, ".env"}
	}
}
