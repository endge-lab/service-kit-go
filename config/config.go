package config

const defaultServiceAppEnv = "development"

// Load is the package-level entrypoint for the current service configuration.
func Load() (*ServiceConfig, error) {
	return LoadServiceConfig()
}
