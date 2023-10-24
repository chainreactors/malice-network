package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) RegisterListener(ctx context.Context, req *lispb.RegisterListener) (*commonpb.Empty, error) {
	fmt.Println(req)
	return &commonpb.Empty{}, nil
}

func (rpc *Server) SpiteStream(stream listenerrpc.ListenerRPC_SpiteStreamServer) error {
	listenerID, err := rpc.listenerID(stream.Context())
	if err != nil {
		logs.Log.Error(err.Error())
		return err
	}
	listenersCh[listenerID] = stream
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		sess := core.Sessions.Get(msg.SessionId)
		if ch, ok := sess.GetTask(msg.TaskId); ok {
			ch <- msg.Spite
		}
	}
}
