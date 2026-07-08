package redis

import (
	"context"
	"errors"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fakeConfig struct {
	host     string
	port     int
	username string
	password string
	database int
}

func (c fakeConfig) GetHost() string {
	return c.host
}

func (c fakeConfig) GetPort() int {
	return c.port
}

func (c fakeConfig) GetUsername() string {
	return c.username
}

func (c fakeConfig) GetPassword() string {
	return c.password
}

func (c fakeConfig) GetDatabase() int {
	return c.database
}

type fakeLifecycle struct {
	hooks []fx.Hook
}

func (l *fakeLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestNewRedisClientMapsConfig(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeLifecycle{}
	client, err := NewRedisClient(lifecycle, fakeConfig{
		host:     "redis",
		port:     6380,
		username: "user",
		password: "secret",
		database: 2,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil && !errors.Is(err, goredis.ErrClosed) {
			t.Fatalf("client.Close() error = %v", err)
		}
	})

	opts := client.Options()
	if opts.Addr != "redis:6380" {
		t.Fatalf("Addr = %q, want %q", opts.Addr, "redis:6380")
	}
	if opts.Username != "user" {
		t.Fatalf("Username = %q, want %q", opts.Username, "user")
	}
	if opts.Password != "secret" {
		t.Fatalf("Password = %q, want %q", opts.Password, "secret")
	}
	if opts.DB != 2 {
		t.Fatalf("DB = %d, want 2", opts.DB)
	}
	if len(lifecycle.hooks) != 1 {
		t.Fatalf("hooks len = %d, want 1", len(lifecycle.hooks))
	}
	if lifecycle.hooks[0].OnStart == nil {
		t.Fatal("OnStart hook = nil")
	}
	if lifecycle.hooks[0].OnStop == nil {
		t.Fatal("OnStop hook = nil")
	}
}

func TestNewRedisClientLeavesEmptyCredentialsUnset(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeLifecycle{}
	client, err := NewRedisClient(lifecycle, fakeConfig{
		host:     "redis",
		port:     6379,
		database: 0,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil && !errors.Is(err, goredis.ErrClosed) {
			t.Fatalf("client.Close() error = %v", err)
		}
	})

	opts := client.Options()
	if opts.Username != "" {
		t.Fatalf("Username = %q, want empty", opts.Username)
	}
	if opts.Password != "" {
		t.Fatalf("Password = %q, want empty", opts.Password)
	}
}

func TestNewRedisClientLifecycleHooks(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeLifecycle{}
	client, err := NewRedisClient(lifecycle, fakeConfig{
		host: "redis",
		port: 6379,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v, want nil", err)
	}

	if len(lifecycle.hooks) != 1 {
		t.Fatalf("hooks len = %d, want 1", len(lifecycle.hooks))
	}
	if err := lifecycle.hooks[0].OnStart(context.Background()); err != nil {
		t.Fatalf("OnStart() error = %v, want nil", err)
	}
	if err := lifecycle.hooks[0].OnStop(context.Background()); err != nil {
		t.Fatalf("OnStop() error = %v, want nil", err)
	}
	if err := client.Close(); err != nil && !errors.Is(err, goredis.ErrClosed) {
		t.Fatalf("client.Close() after OnStop() error = %v", err)
	}
}

func TestNewRedisClientWithBreaker(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeLifecycle{}
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "redis"})

	client, err := NewRedisClient(lifecycle, fakeConfig{
		host: "redis",
		port: 6379,
	}, zap.NewNop(), WithBreaker(breaker))
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil && !errors.Is(err, goredis.ErrClosed) {
			t.Fatalf("client.Close() error = %v", err)
		}
	})

	if client.Options().Addr != "redis:6379" {
		t.Fatalf("Addr = %q, want %q", client.Options().Addr, "redis:6379")
	}
}

func TestNewRedisClientReturnsBreakerError(t *testing.T) {
	t.Parallel()

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name: "redis",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})
	_, _ = breaker.Execute(func() (interface{}, error) {
		return nil, errors.New("trip breaker")
	})

	client, err := NewRedisClient(&fakeLifecycle{}, fakeConfig{
		host: "redis",
		port: 6379,
	}, zap.NewNop(), WithBreaker(breaker))
	if err == nil {
		if client != nil {
			_ = client.Close()
		}
		t.Fatal("NewRedisClient() error = nil, want breaker error")
	}
	if client != nil {
		t.Fatal("NewRedisClient() client != nil, want nil")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("NewRedisClient() error = %v, want ErrOpenState", err)
	}
}
