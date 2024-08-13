package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
)

func (rpc *Server) ListExtensions(ctx context.Context, _ *implantpb.Empty) (*clientpb.Task, error) {
	return nil, nil
}

func (rpc *Server) LoadExtension(ctx context.Context, req *implantpb.LoadExtension) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerAsyncResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteExtension(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerAsyncResponse(ch, types.MsgAssemblyResponse)
	return greq.Task.ToProtobuf(), nil
}
