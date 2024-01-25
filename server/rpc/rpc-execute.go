package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) Execute(ctx context.Context, req *pluginpb.ExecRequest) (*clientpb.Task, error) {
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

		err := AssertAsyncResponse(resp, types.MsgExec)
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

func (rpc *Server) ExecuteAssembly(ctx context.Context, req *pluginpb.ExecuteAssembly) (*clientpb.Task, error) {
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
		err := AssertAsyncResponse(resp, types.MsgAssemblyResponse)
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

func (rpc *Server) ExecuteShellcode(ctx context.Context, req *pluginpb.ExecuteShellcode) (*clientpb.Task, error) {
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
		err := AssertAsyncResponse(resp, types.MsgAssemblyResponse)
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

func (rpc *Server) ExecuteBof(ctx context.Context, req *pluginpb.ExecuteBof) (*clientpb.Task, error) {
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
		err := AssertAsyncResponse(resp, types.MsgAssemblyResponse)
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
