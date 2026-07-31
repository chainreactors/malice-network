package listener

import (
	"context"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func withListenerID(ctx context.Context, listenerID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	if md == nil {
		md = metadata.MD{}
	}
	md.Set("listener_id", listenerID)
	return metadata.NewOutgoingContext(ctx, md)
}

func listenerIdentityUnaryClientInterceptor(listenerID string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(withListenerID(ctx, listenerID), method, req, reply, conn, opts...)
	}
}

func listenerIdentityStreamClientInterceptor(listenerID string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(withListenerID(ctx, listenerID), desc, conn, method, opts...)
	}
}

type pipelineRPCClient interface {
	core.ForwardClient
	GetArtifact(ctx context.Context, in *clientpb.Artifact, opts ...grpc.CallOption) (*clientpb.Artifact, error)
}

type reversePipelineRPC struct {
	listenerrpc.ListenerRPCClient
}

func (r *reversePipelineRPC) OpenForwardStream(ctx context.Context, pipeline core.Pipeline) (core.ForwardStream, error) {
	return core.NewReverseForwardClient(r.ListenerRPCClient).OpenForwardStream(ctx, pipeline)
}

func (r *reversePipelineRPC) Register(ctx context.Context, in *clientpb.RegisterSession, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.ListenerRPCClient.Register(ctx, in, opts...)
}

func (r *reversePipelineRPC) Checkin(ctx context.Context, in *implantpb.Ping, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.ListenerRPCClient.Checkin(ctx, in, opts...)
}
