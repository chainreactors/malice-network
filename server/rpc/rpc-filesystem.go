package rpc

import (
	"context"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Pwd(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModulePwd, types.MsgResponse)
}

func (rpc *Server) Ls(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleLs, types.MsgLs)
}

func (rpc *Server) Cd(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleCd, types.MsgResponse, func(greq *GenericRequest, spite *implantpb.Spite) {
		if output := spite.GetResponse().GetOutput(); output != "" {
			greq.Session.WorkDir = output
			_ = greq.Session.SaveAndNotify("")
		}
	})
}

func (rpc *Server) Mkdir(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleMkdir, types.MsgEmpty)
}

func (rpc *Server) Touch(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleTouch, types.MsgEmpty)
}

func (rpc *Server) Rm(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleRm, types.MsgEmpty)
}

func (rpc *Server) Cat(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleCat, types.MsgBinaryResponse)
}

func (rpc *Server) Mv(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleMv, types.MsgEmpty)
}

func (rpc *Server) Cp(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleCp, types.MsgEmpty)
}

func (rpc *Server) Chmod(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleChmod, types.MsgEmpty)
}

func (rpc *Server) Chown(ctx context.Context, req *implantpb.ChownRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}

func (rpc *Server) EnumDrivers(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleEnumDrivers, types.MsgEnumDrivers)
}
