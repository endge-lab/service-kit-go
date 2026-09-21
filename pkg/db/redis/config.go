package redis

type Config interface {
	GetHost() string
	GetPort() int
	GetUsername() string
	GetPassword() string
	GetDatabase() int
}
