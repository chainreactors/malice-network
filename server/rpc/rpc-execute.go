package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) Execute(ctx context.Context, req *pluginpb.ExecRequest) (*pluginpb.ExecResponse, error) {
	resp, err := rpc.genericHandler(ctx, newGenericRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.(*pluginpb.ExecResponse), nil
}

func (rpc *Server) ExecuteAssembly(ctx context.Context, req *pluginpb.ExecuteLoadAssembly) (*clientpb.Task, error) {
	greq := newGenericRequest(req)
	stat, ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	if stat.Status == 0 {
		return nil, fmt.Errorf("execute %s error, %s", req.Name, stat.Error)
	}
	go func() {
		greq.SetCallback(func() {
			data := <-ch
			resp := data.GetAssemblyResponse()
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventTaskCallback,
				Data:      resp.Data,
				Err:       resp.Err,
			})
		})
	}()
	return greq.Task.ToProtobuf(), nil
}
