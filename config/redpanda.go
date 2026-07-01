package config

import (
	"strings"
	"time"
)

type RedpandaConfigGetter interface {
	GetEnabled() bool
	GetBrokers() string
	GetClientID() string
	GetDialTimeout() time.Duration
	GetReadBatchTimeout() time.Duration
	GetWriteTimeout() time.Duration
}

type ServiceRedpandaConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	Brokers          string        `mapstructure:"brokers"`
	ClientID         string        `mapstructure:"client_id"`
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`
	ReadBatchTimeout time.Duration `mapstructure:"read_batch_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
}

func (c ServiceRedpandaConfig) GetEnabled() bool {
	return c.Enabled
}

func (c ServiceRedpandaConfig) GetBrokers() string {
	return c.Brokers
}

func (c ServiceRedpandaConfig) GetClientID() string {
	return c.ClientID
}

func (c ServiceRedpandaConfig) GetDialTimeout() time.Duration {
	return c.DialTimeout
}

func (c ServiceRedpandaConfig) GetReadBatchTimeout() time.Duration {
	return c.ReadBatchTimeout
}

func (c ServiceRedpandaConfig) GetWriteTimeout() time.Duration {
	return c.WriteTimeout
}

func (c ServiceRedpandaConfig) BrokerList() []string {
	if strings.TrimSpace(c.Brokers) == "" {
		return nil
	}

	parts := strings.Split(c.Brokers, ",")
	brokers := make([]string, 0, len(parts))

	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker == "" {
			continue
		}
		brokers = append(brokers, broker)
	}

	return brokers
}
