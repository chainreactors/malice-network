package rpc

import (
	"context"
	"errors"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func handlerModule(sess *core.Session) func(spite *implantpb.Spite) {
	return func(spite *implantpb.Spite) {
		if modules := spite.GetModules(); modules != nil {
			sess.Modules = modules.Modules
		}
		sess.PushUpdate("")
	}
}

func (rpc *Server) ListModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleListModule)
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

	go greq.HandlerResponse(ch, types.MsgListModule, handlerModule(greq.Session))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) LoadModule(ctx context.Context, req *implantpb.LoadModule) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgListModule, func(spite *implantpb.Spite) {
		greq.Session.Modules = append(greq.Session.Modules, spite.GetModules().Modules...)
		greq.Session.PushUpdate("")
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) RefreshModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleRefreshModule)
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

	go greq.HandlerResponse(ch, types.MsgListModule, handlerModule(greq.Session))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Clear(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleClear)
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

// ExecuteModule passthrough for fully dynamic module execution.
func (rpc *Server) ExecuteModule(ctx context.Context, req *implantpb.ExecuteModuleRequest) (*clientpb.Task, error) {
	if req == nil || req.Spite == nil {
		return nil, errors.New("spite required")
	}
	if req.Expect == "" {
		return nil, errors.New("expect required")
	}

	return Handler(ctx, rpc, req.Spite, types.MsgName(req.Expect))
}
