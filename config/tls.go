package config

// ServiceTLSConfig is shared by server and client modules. When Enabled is true and
// the config is used for server TLS, CertFile and KeyFile are required.
type ServiceTLSConfigGetter interface {
	GetEnabled() bool
	GetCertFile() string
	GetKeyFile() string
	GetCaFile() string
	GetInsecureSkipVerify() bool
}

type ServiceTLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	CAFile             string `mapstructure:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

func (c ServiceTLSConfig) GetEnabled() bool {
	return c.Enabled
}

func (c ServiceTLSConfig) GetCertFile() string {
	return c.CertFile
}

func (c ServiceTLSConfig) GetKeyFile() string {
	return c.KeyFile
}

func (c ServiceTLSConfig) GetCaFile() string {
	return c.CAFile
}

func (c ServiceTLSConfig) GetInsecureSkipVerify() bool {
	return c.InsecureSkipVerify
}
