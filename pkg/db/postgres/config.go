package postgres

import "time"

type Config interface {
	GetUser() string
	GetPassword() string
	GetDatabase() string
	GetHost() string
	GetPort() int
	GetSSLMode() string
	GetConnTimeout() int
	GetMaxConn() int
	GetMinConnLifeTime() time.Duration
	GetMaxConnIdleTime() time.Duration
	GetSchema() string
	GetMigrationsEnabled() bool
}
