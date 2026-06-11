package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) Netstat(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleNetstat, types.MsgNetstat)
}

func (rpc *Server) Curl(ctx context.Context, req *implantpb.CurlRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}
