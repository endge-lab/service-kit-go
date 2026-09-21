package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type options struct {
	breaker *gobreaker.CircuitBreaker
}

type Option func(*options)

func WithBreaker(b *gobreaker.CircuitBreaker) Option {
	return func(o *options) {
		o.breaker = b
	}
}

func NewRedisClient(
	lc fx.Lifecycle,
	conf Config,
	logg *zap.Logger,
	opts ...Option,
) (*redis.Client, error) {
	o := &options{}
	for _, apply := range opts {
		apply(o)
	}

	createClient := func() (*redis.Client, error) {
		rdbOpts := &redis.Options{
			Addr: conf.GetHost() + ":" + strconv.Itoa(conf.GetPort()),
			DB:   conf.GetDatabase(),
		}

		if conf.GetPassword() != "" {
			rdbOpts.Password = conf.GetPassword()
		}

		if conf.GetUsername() != "" {
			rdbOpts.Username = conf.GetUsername()
		}

		client := redis.NewClient(rdbOpts)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logg.Info("redis started")

				return nil
			},
			OnStop: func(ctx context.Context) error {
				logg.Info("redis stopped")

				return client.Close()
			},
		})

		return client, nil
	}

	if o.breaker != nil {
		result, err := o.breaker.Execute(func() (interface{}, error) {
			return createClient()
		})
		if err != nil {
			return nil, fmt.Errorf("redis circuit breaker error: %w", err)
		}

		return result.(*redis.Client), nil
	}

	client, err := createClient()
	if err != nil {
		return nil, fmt.Errorf("redis init error: %w", err)
	}

	return client, nil
}
