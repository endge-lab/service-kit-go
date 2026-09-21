package config

type ServiceAppConfigGetter interface {
	GetEnv() string
	GetName() string
	GetVersion() string
	GetPublicURL() string
	IsProduction() bool
}

type ServiceAppConfig struct {
	Env       string `mapstructure:"env"`
	Name      string `mapstructure:"name"`
	Version   string `mapstructure:"version"`
	PublicURL string `mapstructure:"public_url"`
}

func (c ServiceAppConfig) GetEnv() string {
	return c.Env
}

func (c ServiceAppConfig) GetName() string {
	return c.Name
}

func (c ServiceAppConfig) GetVersion() string {
	return c.Version
}

func (c ServiceAppConfig) GetPublicURL() string {
	return c.PublicURL
}

func (c ServiceAppConfig) IsProduction() bool {
	return c.Env == "production"
}
