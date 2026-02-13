package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Kill(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleKill)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Ps(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModulePs)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgPs)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Env(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleEnv)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) SetEnv(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleSetEnv)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) UnsetEnv(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleUnsetEnv)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Whoami(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleWhoami)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Bypass(ctx context.Context, req *implantpb.BypassRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}
