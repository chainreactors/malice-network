package listener

import (
	"context"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

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
