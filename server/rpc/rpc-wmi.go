package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) WmiQuery(ctx context.Context, req *implantpb.WmiQueryRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}

func (rpc *Server) WmiExecute(ctx context.Context, req *implantpb.WmiMethodRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}
