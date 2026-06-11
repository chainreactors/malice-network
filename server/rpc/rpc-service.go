package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) ServiceList(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleServiceList, types.MsgServicesResponse)
}

func (rpc *Server) ServiceStart(ctx context.Context, req *implantpb.ServiceRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) ServiceStop(ctx context.Context, req *implantpb.ServiceRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) ServiceQuery(ctx context.Context, req *implantpb.ServiceRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgServiceResponse)
}

func (rpc *Server) ServiceCreate(ctx context.Context, req *implantpb.ServiceRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) ServiceDelete(ctx context.Context, req *implantpb.ServiceRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}
