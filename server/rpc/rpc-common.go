package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
)

// ModuleCommon dispatches CommonBody based modules by name.
func (rpc *Server) ModuleCommon(ctx context.Context, req *implantpb.CommonRequest) (*clientpb.Task, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("common body required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("module name required")
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go func() {
		resp := <-ch
		if err := types.HandleMaleficError(resp); err != nil {
			logs.Log.Debug(err)
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}
		respType := types.MessageType(resp)
		if respType != types.MsgName(req.Name) && respType != types.MsgEmpty {
			err := fmt.Errorf("%w, expect %s or empty, got %s", types.ErrAssertFailure, req.Name, respType)
			logs.Log.Debug(err)
			greq.Task.Panic(buildErrorEvent(greq.Task, err))
			return
		}

		if err := greq.HandlerSpite(resp); err != nil {
			logs.Log.Errorf("handler spite error, %s", err.Error())
			return
		}
		greq.Task.Finish(resp, "")
	}()
	return greq.Task.ToProtobuf(), nil
}
