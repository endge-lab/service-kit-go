package fiber

import (
	"context"
	"sync/atomic"

	gofiber "github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegisterLifecycle starts Fiber on fx start and shuts it down on fx stop.
func RegisterLifecycle(lc fx.Lifecycle, app *gofiber.App, addr string, logger *zap.Logger) {
	var stopping atomic.Bool

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(addr); err != nil {
					if stopping.Load() {
						if logger != nil {
							logger.Debug("HTTP server stopped", zap.Error(err))
						}
						return
					}
					if logger != nil {
						logger.Error("HTTP server failed", zap.Error(err))
					}
				}
			}()
			if logger != nil {
				logger.Info("HTTP server started", zap.String("addr", addr))
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopping.Store(true)
			if logger != nil {
				logger.Info("shutting down HTTP server")
			}
			return app.ShutdownWithContext(ctx)
		},
	})
}
