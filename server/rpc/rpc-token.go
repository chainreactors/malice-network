package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Runas(ctx context.Context, req *implantpb.RunAsRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgExec)

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Rev2Self(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleRev2Self)
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

	go greq.HandlerResponse(ch, types.MsgEmpty)

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Privs(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModulePrivs)
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

	go greq.HandlerResponse(ch, types.MsgResponse)

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) GetSystem(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleGetSystem)
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

	go greq.HandlerResponse(ch, types.MsgResponse)

	return greq.Task.ToProtobuf(), nil
}
