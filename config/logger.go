package config

import "time"

type ServiceLoggerConfigGetter interface {
	GetLoggerConfig() string
}

// ServiceLoggerOpenSearchConfig задает прямую выгрузку структурированных логов
// в OpenSearch Bulk API. Пустой endpoint отключает exporter.
type ServiceLoggerOpenSearchConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	Endpoint           string        `mapstructure:"endpoint"`
	Index              string        `mapstructure:"index"`
	Username           string        `mapstructure:"username"`
	Password           string        `mapstructure:"password"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"`
	FlushInterval      time.Duration `mapstructure:"flush_interval"`
	BatchSize          int           `mapstructure:"batch_size"`
	QueueSize          int           `mapstructure:"queue_size"`
	RequestTimeout     time.Duration `mapstructure:"request_timeout"`
}

type ServiceLoggerConfig struct {
	Level      string                        `mapstructure:"level"`
	OpenSearch ServiceLoggerOpenSearchConfig `mapstructure:"opensearch"`
}

func (c ServiceLoggerConfig) GetLoggerConfig() string {
	return c.Level
}
