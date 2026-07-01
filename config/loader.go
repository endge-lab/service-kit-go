package config

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func LoadServiceConfig() (*ServiceConfig, error) {
	preloadServiceEnvFiles()

	appEnv := NormalizeServiceEnvironment(detectServiceAppEnv())
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setServiceDefaults(v, appEnv)
	if err := bindServiceEnv(v); err != nil {
		return nil, err
	}
	setServiceConfigFile(v, appEnv)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg ServiceConfig
	if err := v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc())); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
