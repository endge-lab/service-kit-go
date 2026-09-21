package grpckit

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

func NewClient(config ClientConfig, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	if config.Target == "" {
		return nil, errors.New("grpc client target is required")
	}
	transportCredentials, err := clientTransportCredentials(config.TLS)
	if err != nil {
		return nil, err
	}
	options = append(options, grpc.WithTransportCredentials(transportCredentials))
	keepaliveTime := config.KeepaliveTime
	if keepaliveTime <= 0 {
		keepaliveTime = 30 * time.Second
	}
	keepaliveTimeout := config.KeepaliveTimeout
	if keepaliveTimeout <= 0 {
		keepaliveTimeout = 10 * time.Second
	}
	options = append(options, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time: keepaliveTime, Timeout: keepaliveTimeout, PermitWithoutStream: true,
	}))
	callOptions := make([]grpc.CallOption, 0, 3)
	if config.MaxReceiveBytes > 0 {
		callOptions = append(callOptions, grpc.MaxCallRecvMsgSize(config.MaxReceiveBytes))
	}
	if config.MaxSendBytes > 0 {
		callOptions = append(callOptions, grpc.MaxCallSendMsgSize(config.MaxSendBytes))
	}
	if config.Compression {
		callOptions = append(callOptions, grpc.UseCompressor(gzip.Name))
	}
	if len(callOptions) > 0 {
		options = append(options, grpc.WithDefaultCallOptions(callOptions...))
	}
	return grpc.NewClient(config.Target, options...)
}

func ContextWithDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline || timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func clientTransportCredentials(config TLSConfig) (credentials.TransportCredentials, error) {
	if !config.Enabled {
		return insecure.NewCredentials(), nil
	}
	tlsConfig := &tls.Config{
		ServerName:         config.ServerName,
		InsecureSkipVerify: config.InsecureSkipVerify, //nolint:gosec // explicit development option
		MinVersion:         tls.VersionTLS12,
	}
	if config.CAFile != "" {
		contents, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read grpc ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(contents) {
			return nil, errors.New("grpc ca file contains no certificates")
		}
		tlsConfig.RootCAs = pool
	}
	if config.CertFile != "" || config.KeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load grpc client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return credentials.NewTLS(tlsConfig), nil
}
