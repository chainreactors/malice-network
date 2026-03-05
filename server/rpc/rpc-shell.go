package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"strings"
)

func (rpc *Server) PtyRequest(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	action := req.GetType()

	switch action {
	case consts.ModulePtyStart:
		greq.Count = -1
		_, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		core.SafeGoWithTask(greq.Task, func() {
			for {
				resp := <-out
				err := types.AssertSpite(resp, types.MsgPtyResponse)
				if err != nil {
					greq.Task.Panic(buildErrorEvent(greq.Task, err))
					return
				}
				err = greq.HandlerSpite(resp)
				if err != nil {
					return
				}
				greq.Task.Finish(resp, "")

				moduleResp := resp.GetResponse()
				if moduleResp != nil && moduleResp.GetError() != "" &&
					(strings.Contains(moduleResp.GetError(), "session") &&
						strings.Contains(moduleResp.GetError(), "closed")) {
					greq.Task.Finish(resp, "Shell session ended")
					break
				}
			}
		}, greq.Task.Close)

	default:
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgPtyResponse)
	}

	return greq.Task.ToProtobuf(), nil
}
