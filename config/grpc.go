package config

import "time"

// GRPCConfigGetter is intentionally separate from ServiceConfigGetter so
// HTTP-only consumers are not forced to implement a new method.
type GRPCConfigGetter interface {
	GetGRPCConfig() ServiceGRPCConfig
}

// ServiceGRPCConfig describes the optional gRPC listener owned by a service.
type ServiceGRPCConfig struct {
	Enabled             bool          `mapstructure:"enabled"`
	Port                int           `mapstructure:"port"`
	MaxReceiveBytes     int           `mapstructure:"max_receive_bytes"`
	MaxSendBytes        int           `mapstructure:"max_send_bytes"`
	GracefulStopTimeout time.Duration `mapstructure:"graceful_stop_timeout"`
	KeepaliveTime       time.Duration `mapstructure:"keepalive_time"`
	KeepaliveTimeout    time.Duration `mapstructure:"keepalive_timeout"`
}
