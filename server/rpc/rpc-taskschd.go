package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) TaskSchdList(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleTaskSchdList, types.MsgTaskSchdsResponse)
}

func (rpc *Server) TaskSchdCreate(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) TaskSchdStart(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) TaskSchdStop(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) TaskSchdDelete(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) TaskSchdQuery(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgTaskSchdResponse)
}

func (rpc *Server) TaskSchdRun(ctx context.Context, req *implantpb.TaskScheduleRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}
