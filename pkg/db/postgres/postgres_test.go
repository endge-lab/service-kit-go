package postgres

import (
	"context"
	"embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fakeConfig struct {
	user              string
	password          string
	database          string
	host              string
	port              int
	sslMode           string
	connTimeout       int
	maxConn           int
	maxConnLifetime   time.Duration
	maxConnIdleTime   time.Duration
	schema            string
	migrationsEnabled bool
}

func (c fakeConfig) GetUser() string                   { return c.user }
func (c fakeConfig) GetPassword() string               { return c.password }
func (c fakeConfig) GetDatabase() string               { return c.database }
func (c fakeConfig) GetHost() string                   { return c.host }
func (c fakeConfig) GetPort() int                      { return c.port }
func (c fakeConfig) GetSSLMode() string                { return c.sslMode }
func (c fakeConfig) GetConnTimeout() int               { return c.connTimeout }
func (c fakeConfig) GetMaxConn() int                   { return c.maxConn }
func (c fakeConfig) GetMinConnLifeTime() time.Duration { return c.maxConnLifetime }
func (c fakeConfig) GetMaxConnIdleTime() time.Duration { return c.maxConnIdleTime }
func (c fakeConfig) GetSchema() string                 { return c.schema }
func (c fakeConfig) GetMigrationsEnabled() bool        { return c.migrationsEnabled }

type fakeLifecycle struct {
	hooks []fx.Hook
}

func (l *fakeLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestOptionsApply(t *testing.T) {
	t.Parallel()

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "postgres"})
	var fs embed.FS
	opts := &options{}
	WithBreaker(breaker)(opts)
	WithEmbedFS(fs)(opts)
	WithRunMigration(true)(opts)

	if opts.breaker != breaker {
		t.Fatal("WithBreaker() did not set breaker")
	}
	if !opts.runMigration {
		t.Fatal("WithRunMigration(true) did not enable migrations")
	}
}

func TestNewPostgresClientReturnsParseErrorWithoutConnecting(t *testing.T) {
	t.Parallel()

	pool, err := NewPostgresClient(context.Background(), &fakeLifecycle{}, fakeConfig{
		user:            "postgres",
		password:        "secret",
		database:        "service",
		host:            "localhost",
		port:            -1,
		sslMode:         "disable",
		connTimeout:     1,
		maxConn:         1,
		maxConnLifetime: time.Minute,
		maxConnIdleTime: time.Minute,
		schema:          "public",
	}, zap.NewNop())
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("NewPostgresClient() error = nil, want parse error")
	}
	if pool != nil {
		t.Fatal("NewPostgresClient() pool != nil, want nil")
	}
	if !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("NewPostgresClient() error = %q, want invalid port parse error", err.Error())
	}
}

func TestNewPostgresClientWrapsBreakerError(t *testing.T) {
	t.Parallel()

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name: "postgres",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})
	_, _ = breaker.Execute(func() (interface{}, error) {
		return nil, errors.New("trip breaker")
	})

	pool, err := NewPostgresClient(context.Background(), &fakeLifecycle{}, fakeConfig{
		user:            "postgres",
		password:        "secret",
		database:        "service",
		host:            "localhost",
		port:            5432,
		sslMode:         "disable",
		connTimeout:     1,
		maxConn:         1,
		maxConnLifetime: time.Minute,
		maxConnIdleTime: time.Minute,
		schema:          "public",
	}, zap.NewNop(), WithBreaker(breaker))
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("NewPostgresClient() error = nil, want breaker error")
	}
	if pool != nil {
		t.Fatal("NewPostgresClient() pool != nil, want nil")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("NewPostgresClient() error = %v, want ErrOpenState", err)
	}
}
