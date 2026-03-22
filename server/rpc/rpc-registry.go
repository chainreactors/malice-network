package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) RegQuery(ctx context.Context, req *implantpb.RegistryRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}

func (rpc *Server) RegAdd(ctx context.Context, req *implantpb.RegistryWriteRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) RegDelete(ctx context.Context, req *implantpb.RegistryRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgEmpty)
}

func (rpc *Server) RegListKey(ctx context.Context, req *implantpb.RegistryRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}

func (rpc *Server) RegListValue(ctx context.Context, req *implantpb.RegistryRequest) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgResponse)
}
