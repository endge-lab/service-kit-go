package grpckit

import "time"

type TLSConfig struct {
	Enabled            bool
	CertFile           string
	KeyFile            string
	CAFile             string
	ServerName         string
	InsecureSkipVerify bool
}

type ServerConfig struct {
	Address             string
	MaxReceiveBytes     int
	MaxSendBytes        int
	GracefulStopTimeout time.Duration
	KeepaliveTime       time.Duration
	KeepaliveTimeout    time.Duration
	TLS                 TLSConfig
}

type ClientConfig struct {
	Target           string
	DefaultTimeout   time.Duration
	MaxReceiveBytes  int
	MaxSendBytes     int
	Compression      bool
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
	TLS              TLSConfig
}
