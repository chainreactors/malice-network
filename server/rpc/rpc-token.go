package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Runas(ctx context.Context, req *implantpb.RunAsRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgExec)
}

func (rpc *Server) Rev2Self(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleRev2Self, types.MsgEmpty)
}

func (rpc *Server) Privs(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModulePrivs, types.MsgResponse)
}

func (rpc *Server) GetSystem(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleGetSystem, types.MsgResponse)
}
