package grpckit

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

type Server struct {
	config ServerConfig
	server *grpc.Server
	health *health.Server
}

func NewServer(config ServerConfig, options ...grpc.ServerOption) (*Server, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("grpc server address is required")
	}
	if config.GracefulStopTimeout <= 0 {
		config.GracefulStopTimeout = 10 * time.Second
	}
	if config.KeepaliveTime <= 0 {
		config.KeepaliveTime = 30 * time.Second
	}
	if config.KeepaliveTimeout <= 0 {
		config.KeepaliveTimeout = 10 * time.Second
	}
	options = append(options, grpc.KeepaliveParams(keepalive.ServerParameters{
		Time: config.KeepaliveTime, Timeout: config.KeepaliveTimeout,
	}))
	if config.MaxReceiveBytes > 0 {
		options = append(options, grpc.MaxRecvMsgSize(config.MaxReceiveBytes))
	}
	if config.MaxSendBytes > 0 {
		options = append(options, grpc.MaxSendMsgSize(config.MaxSendBytes))
	}
	if config.TLS.Enabled {
		certificate, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load grpc server certificate: %w", err)
		}
		options = append(options, grpc.Creds(credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS12,
		})))
	}
	server := grpc.NewServer(options...)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{config: config, server: server, health: healthServer}, nil
}

func (s *Server) GRPC() *grpc.Server     { return s.server }
func (s *Server) Health() *health.Server { return s.health }

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("listen grpc at %s: %w", s.config.Address, err)
	}
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- s.server.Serve(listener) }()
	select {
	case err := <-serveErrors:
		return err
	case <-ctx.Done():
		s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		done := make(chan struct{})
		go func() {
			s.server.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(s.config.GracefulStopTimeout):
			s.server.Stop()
		}
		return nil
	}
}

// FileExists is exported for applications that validate optional TLS files
// before constructing a server or client.
func FileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
