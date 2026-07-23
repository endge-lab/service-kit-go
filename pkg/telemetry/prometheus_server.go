package telemetry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// PrometheusServerConfig configures the dedicated Prometheus scrape endpoint.
type PrometheusServerConfig struct {
	BindAddress string
	HandlerPath string
}

// PrometheusServer serves a scrape handler independently from the application
// HTTP server, so it is not affected by API routing or authentication.
type PrometheusServer struct {
	config  PrometheusServerConfig
	handler http.Handler
	logger  *zap.Logger

	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
}

func NewPrometheusServer(cfg PrometheusServerConfig, handler http.Handler, logger *zap.Logger) (*PrometheusServer, error) {
	if handler == nil {
		return nil, errors.New("prometheus handler is required")
	}

	cfg.BindAddress = strings.TrimSpace(cfg.BindAddress)
	cfg.HandlerPath = strings.TrimSpace(cfg.HandlerPath)
	if cfg.BindAddress == "" {
		return nil, errors.New("prometheus bind address is required")
	}
	if cfg.HandlerPath == "" {
		return nil, errors.New("prometheus handler path is required")
	}
	if !strings.HasPrefix(cfg.HandlerPath, "/") {
		cfg.HandlerPath = "/" + cfg.HandlerPath
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &PrometheusServer{config: cfg, handler: handler, logger: logger}, nil
}

// Start begins serving the configured scrape endpoint.
func (s *PrometheusServer) Start() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return nil
	}

	listener, err := net.Listen("tcp", s.config.BindAddress)
	if err != nil {
		return err
	}

	s.listener = listener
	s.server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != s.config.HandlerPath {
				http.NotFound(w, r)
				return
			}
			s.handler.ServeHTTP(w, r)
		}),
	}

	s.logger.Info("prometheus metrics server started", zap.String("address", listener.Addr().String()), zap.String("path", s.config.HandlerPath))
	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("prometheus metrics server stopped unexpectedly", zap.Error(err))
		}
	}()

	return nil
}

// Address returns the bound address after Start. It is mainly useful in tests.
func (s *PrometheusServer) Address() string {
	if s == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}

	return s.listener.Addr().String()
}

// Shutdown stops accepting Prometheus scrapes.
func (s *PrometheusServer) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	server := s.server
	s.mu.Unlock()
	if server == nil {
		return nil
	}

	return server.Shutdown(ctx)
}
