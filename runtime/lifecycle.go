package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	serviceerrors "github.com/endge-lab/service-kit-go/errors"
	"github.com/endge-lab/service-kit-go/logging"

	"go.uber.org/zap"
)

// Hook описывает старт и остановку runtime-компонента.
type Hook struct {
	Name  string
	Start func(context.Context) error
	Stop  func(context.Context) error
}

// Lifecycle управляет общими runtime hooks сервиса.
type Lifecycle struct {
	hooks  []Hook
	logger *zap.Logger
}

// NewLifecycle создает новый lifecycle manager.
func NewLifecycle(logger *zap.Logger) *Lifecycle {
	return &Lifecycle{
		logger: logging.WithComponent(logger, "runtime_lifecycle"),
	}
}

// Append добавляет hook в lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	l.hooks = append(l.hooks, hook)
}

// Start запускает hooks в прямом порядке.
func (l *Lifecycle) Start(ctx context.Context) error {
	for _, hook := range l.hooks {
		if hook.Start == nil {
			continue
		}
		if l.logger != nil {
			l.logger.Info("starting runtime hook", zap.String("hook", hook.Name))
		}
		if err := hook.Start(ctx); err != nil {
			return serviceerrors.Wrap(err, "runtime.start_failed", "Не удалось запустить runtime hook", 500)
		}
	}

	return nil
}

// Stop останавливает hooks в обратном порядке.
func (l *Lifecycle) Stop(ctx context.Context) error {
	for index := len(l.hooks) - 1; index >= 0; index-- {
		hook := l.hooks[index]
		if hook.Stop == nil {
			continue
		}
		if l.logger != nil {
			l.logger.Info("stopping runtime hook", zap.String("hook", hook.Name))
		}
		if err := hook.Stop(ctx); err != nil {
			return serviceerrors.Wrap(err, "runtime.stop_failed", "Не удалось остановить runtime hook", 500)
		}
	}

	return nil
}

// NotifyContext создает context, который завершится по SIGINT или SIGTERM.
func NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}
