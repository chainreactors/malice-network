package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) ListDevice(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleFFmpeg, types.MsgListModule, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyModulesResponse(greq.Session, spite, false)
	})
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
	greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}
