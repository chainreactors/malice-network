package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) GetTasks(ctx context.Context, session *clientpb.Session) (*clientpb.Tasks, error) {
	resp := &clientpb.Tasks{
		Tasks: []*clientpb.Task{},
	}

	sess, ok := core.Sessions.Get(session.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	for _, task := range sess.Tasks.All() {
		resp.Tasks = append(resp.Tasks, task.ToProtobuf())
	}

	return resp, nil
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*commonpb.Spite, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}

	return task.Spite, nil
}
