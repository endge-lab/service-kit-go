package fiber

import (
	"context"
	"testing"

	gofiber "github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fakeLifecycle struct {
	hooks []fx.Hook
}

func (l *fakeLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestRegisterLifecycleRegistersHooksAndStopsApp(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeLifecycle{}
	app := gofiber.New()

	RegisterLifecycle(lifecycle, app, "127.0.0.1:0", zap.NewNop())

	if len(lifecycle.hooks) != 1 {
		t.Fatalf("hooks len = %d, want 1", len(lifecycle.hooks))
	}
	if lifecycle.hooks[0].OnStart == nil {
		t.Fatal("OnStart hook = nil")
	}
	if lifecycle.hooks[0].OnStop == nil {
		t.Fatal("OnStop hook = nil")
	}
	if err := lifecycle.hooks[0].OnStop(context.Background()); err != nil {
		t.Fatalf("OnStop() error = %v, want nil", err)
	}
}
