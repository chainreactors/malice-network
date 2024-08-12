package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) GetSessions(ctx context.Context, _ *clientpb.Empty) (*clientpb.Sessions, error) {
	sessions, err := db.FindAllSessions()
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (rpc *Server) BasicSessionOP(ctx context.Context, req *clientpb.BasicUpdateSession) (*clientpb.Empty, error) {
	if req.IsDelete {
		err := db.DeleteSession(req.SessionId)
		if err != nil {
			return nil, err
		}
	} else {
		err := db.UpdateSession(req.SessionId, req.Note, req.GroupName)
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Info(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerAsyncResponse(ch, types.MsgSysInfo)
	return greq.Task.ToProtobuf(), nil
}
