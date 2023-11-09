package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
)

func (rpc *Server) Execute(ctx context.Context, req *pluginpb.ExecRequest) (*pluginpb.ExecResponse, error) {
	resp, err := rpc.GenericHandler(ctx, newGenericRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.(*pluginpb.ExecResponse), nil
}
