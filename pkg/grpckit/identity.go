package grpckit

import (
	"context"
	"errors"
	"strings"

	serviceoidc "github.com/endge-lab/service-kit-go/pkg/oidc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type identityContextKey struct{}

func IdentityFromContext(ctx context.Context) (serviceoidc.Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(serviceoidc.Identity)
	return identity, ok
}

func UnaryServerIdentityInterceptor(verifier serviceoidc.TokenVerifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		identity, enriched, err := authenticateIncoming(ctx, verifier)
		if err != nil {
			return nil, err
		}
		return handler(context.WithValue(enriched, identityContextKey{}, identity), request)
	}
}

func StreamServerIdentityInterceptor(verifier serviceoidc.TokenVerifier) grpc.StreamServerInterceptor {
	return func(server any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		identity, enriched, err := authenticateIncoming(stream.Context(), verifier)
		if err != nil {
			return err
		}
		return handler(server, &contextServerStream{
			ServerStream: stream,
			ctx:          context.WithValue(enriched, identityContextKey{}, identity),
		})
	}
}

func UnaryClientIdentityInterceptor(provider serviceoidc.TokenProvider) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		enriched, err := outgoingContext(ctx, provider)
		if err != nil {
			return status.Error(codes.Unauthenticated, "service identity token is unavailable")
		}
		return invoke(enriched, method, request, reply, connection, options...)
	}
}

func StreamClientIdentityInterceptor(provider serviceoidc.TokenProvider) grpc.StreamClientInterceptor {
	return func(ctx context.Context, description *grpc.StreamDesc, connection *grpc.ClientConn, method string, streamer grpc.Streamer, options ...grpc.CallOption) (grpc.ClientStream, error) {
		enriched, err := outgoingContext(ctx, provider)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "service identity token is unavailable")
		}
		return streamer(enriched, description, connection, method, options...)
	}
}

func authenticateIncoming(ctx context.Context, verifier serviceoidc.TokenVerifier) (serviceoidc.Identity, context.Context, error) {
	if verifier == nil {
		return serviceoidc.Identity{}, ctx, errors.New("service identity verifier is required")
	}
	values, _ := metadata.FromIncomingContext(ctx)
	authorization := firstMetadata(values.Get("authorization"))
	token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if token == authorization || token == "" {
		return serviceoidc.Identity{}, ctx, status.Error(codes.Unauthenticated, "service identity token is required")
	}
	identity, err := verifier.Verify(ctx, token)
	if err != nil {
		return serviceoidc.Identity{}, ctx, status.Error(codes.Unauthenticated, "service identity token is invalid")
	}
	return identity, extractPropagation(ctx, values), nil
}

func outgoingContext(ctx context.Context, provider serviceoidc.TokenProvider) (context.Context, error) {
	if provider == nil {
		return nil, errors.New("service identity token provider is required")
	}
	token, err := provider.Token(ctx)
	if err != nil {
		return nil, err
	}
	values, _ := metadata.FromOutgoingContext(ctx)
	values = values.Copy()
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-request-id", "traceparent", "tracestate"} {
			if values.Get(key) == nil {
				for _, value := range incoming.Get(key) {
					values.Append(key, value)
				}
			}
		}
	}
	values.Set("authorization", "Bearer "+token)
	carrier := metadataCarrier{values: values}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return metadata.NewOutgoingContext(ctx, values), nil
}

func extractPropagation(ctx context.Context, values metadata.MD) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier{values: values})
}

type metadataCarrier struct {
	values metadata.MD
}

func (c metadataCarrier) Get(key string) string { return firstMetadata(c.values.Get(key)) }
func (c metadataCarrier) Set(key, value string) { c.values.Set(key, value) }
func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = metadataCarrier{}

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *contextServerStream) Context() context.Context { return s.ctx }

func firstMetadata(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
