package config

const defaultServiceAppEnv = "local"

// Load is the package-level entrypoint for the current service configuration.
func Load() (*ServiceConfig, error) {
	return LoadServiceConfig()
}
