package listener

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestListenerIdentityUnaryClientInterceptorSetsListenerID(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"listener_id", "spoofed-listener",
		"trace_id", "trace-1",
	))

	err := listenerIdentityUnaryClientInterceptor("listener-2")(
		ctx,
		"/listenerrpc.ListenerRPC/GetArtifact",
		nil,
		nil,
		nil,
		func(ctx context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("outgoing metadata is missing")
			}
			if got := md.Get("listener_id"); len(got) != 1 || got[0] != "listener-2" {
				t.Fatalf("listener_id metadata = %v, want [listener-2]", got)
			}
			if got := md.Get("trace_id"); len(got) != 1 || got[0] != "trace-1" {
				t.Fatalf("trace_id metadata = %v, want [trace-1]", got)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unary interceptor returned error: %v", err)
	}
}

func TestListenerIdentityStreamClientInterceptorSetsListenerID(t *testing.T) {
	_, err := listenerIdentityStreamClientInterceptor("listener-2")(
		context.Background(),
		&grpc.StreamDesc{},
		nil,
		"/listenerrpc.ListenerRPC/JobStream",
		func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatal("outgoing metadata is missing")
			}
			if got := md.Get("listener_id"); len(got) != 1 || got[0] != "listener-2" {
				t.Fatalf("listener_id metadata = %v, want [listener-2]", got)
			}
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("stream interceptor returned error: %v", err)
	}
}
