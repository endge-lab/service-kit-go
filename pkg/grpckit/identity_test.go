package grpckit

import (
	"context"
	"testing"

	serviceoidc "github.com/endge-lab/service-kit-go/pkg/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type staticTokenProvider struct{ token string }

func (p staticTokenProvider) Token(context.Context) (string, error) { return p.token, nil }

type staticVerifier struct{ identity serviceoidc.Identity }

func (v staticVerifier) Verify(_ context.Context, token string) (serviceoidc.Identity, error) {
	if token != "service-token" {
		return serviceoidc.Identity{}, serviceoidc.ErrUnauthorized
	}
	return v.identity, nil
}

func TestIdentityInterceptorsTransferServiceIdentity(t *testing.T) {
	t.Parallel()
	client := UnaryClientIdentityInterceptor(staticTokenProvider{token: "service-token"})
	server := UnaryServerIdentityInterceptor(staticVerifier{identity: serviceoidc.Identity{Subject: "backend", ClientID: "backend"}})
	var outgoing metadata.MD
	err := client(context.Background(), "/test.Service/Call", struct{}{}, &struct{}{}, nil,
		func(ctx context.Context, method string, request, reply any, connection *grpc.ClientConn, options ...grpc.CallOption) error {
			outgoing, _ = metadata.FromOutgoingContext(ctx)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	incoming := metadata.NewIncomingContext(context.Background(), outgoing)
	_, err = server(incoming, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"},
		func(ctx context.Context, request any) (any, error) {
			identity, ok := IdentityFromContext(ctx)
			if !ok || identity.ClientID != "backend" {
				t.Fatalf("unexpected identity: %#v %v", identity, ok)
			}
			return struct{}{}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}
