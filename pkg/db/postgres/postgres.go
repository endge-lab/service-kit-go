package postgres

import (
	"context"
	_ "database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/endge-lab/service-kit-go/pkg/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const dsn = "user=%s password=%s dbname=%s host=%s port=%d sslmode=%s connect_timeout=%d pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s"

type options struct {
	breaker      *gobreaker.CircuitBreaker
	fs           embed.FS
	runMigration bool
}

type Option func(*options)

// WithBreaker — connect CircuitBreaker
func WithBreaker(b *gobreaker.CircuitBreaker) Option {
	return func(o *options) {
		o.breaker = b
	}
}

// WithEmbedFS — connect embed.FS
func WithEmbedFS(fs embed.FS) Option {
	return func(o *options) {
		o.fs = fs
	}
}

// WithRunMigration — run migrations
func WithRunMigration(enabled bool) Option {
	return func(o *options) {
		o.runMigration = enabled
	}
}

func NewPostgresClient(
	ctx context.Context,
	lc fx.Lifecycle,
	conf Config,
	logger *zap.Logger,
	opts ...Option,
) (*pgxpool.Pool, error) {
	dsnStr := fmt.Sprintf(dsn,
		conf.GetUser(),
		conf.GetPassword(),
		conf.GetDatabase(),
		conf.GetHost(),
		conf.GetPort(),
		conf.GetSSLMode(),
		conf.GetConnTimeout(),
		conf.GetMaxConn(),
		conf.GetMinConnLifeTime(),
		conf.GetMaxConnIdleTime(),
	)

	o := &options{
		runMigration: false,
	}

	for _, apply := range opts {
		apply(o)
	}

	connect := func() (*pgxpool.Pool, error) {
		cfg, err := pgxpool.ParseConfig(dsnStr)
		if err != nil {
			logger.Error("parse config failed", zap.Error(err))
			return nil, err
		}

		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", conf.GetSchema()))
			if err != nil {
				logger.Error("set search path failed", zap.Error(err))
			}
			return err
		}

		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			logger.Error("create pgx pool failed", zap.Error(err))
			return nil, err
		}

		if err = pool.Ping(ctx); err != nil {
			pool.Close()
			logger.Error("ping postgres pool failed", zap.Error(err))
			return nil, err
		}

		return pool, nil
	}

	var pool *pgxpool.Pool
	var err error

	if o.breaker != nil {
		result, execErr := o.breaker.Execute(func() (interface{}, error) {
			return connect()
		})
		if execErr != nil {
			return nil, fmt.Errorf("postgres breaker error: %w", execErr)
		}

		var ok bool
		pool, ok = result.(*pgxpool.Pool)
		if !ok {
			return nil, errors.New("invalid breaker result type")
		}
	} else {
		pool, err = connect()
		if err != nil {
			return nil, err
		}
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing postgres pool", zap.String("host", conf.GetHost()))
			pool.Close()
			return nil
		},
	})

	if o.runMigration || conf.GetMigrationsEnabled() {
		if err = migration(pool, logger, o.fs); err != nil {
			_, err = pool.Exec(context.Background(), fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", conf.GetSchema()))
			if err != nil {
				pool.Close()
				return nil, fmt.Errorf("failed to create schema: %w", err)
			}

			if err = migration(pool, logger, o.fs); err != nil {
				pool.Close()
				return nil, fmt.Errorf("failed to migrate: %w", err)
			}
		}
	}

	return pool, nil
}

func migration(pool *pgxpool.Pool, log *zap.Logger, fs embed.FS) error {
	return migrator.NewMigrator(stdlib.OpenDBFromPool(pool), fs, log).Up()
}
