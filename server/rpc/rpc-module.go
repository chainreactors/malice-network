package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) ListModules(ctx context.Context, _ *commonpb.Empty) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, &commonpb.Request{Name: types.MsgModules.String()})
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go func() {
		resp := <-ch

		err := AssertResponse(resp, types.MsgModules)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		greq.SetCallback(func() {
			greq.Task.Spite = resp
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventTaskCallback,
				Task:      greq.Task,
			})
		})
		greq.Task.Done()
	}()
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) LoadModule(ctx context.Context, req *pluginpb.LoadModule) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go func() {
		resp := <-ch

		err := AssertStatus(resp)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		greq.SetCallback(func() {
			greq.Task.Spite = resp
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventTaskCallback,
				Task:      greq.Task,
			})
		})
		greq.Task.Done()
	}()
	return greq.Task.ToProtobuf(), nil
}
