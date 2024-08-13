package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) GetTasks(ctx context.Context, session *clientpb.Session) (*clientpb.Tasks, error) {
	sess, ok := core.Sessions.Get(session.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}

	return sess.Tasks.ToProtobuf(), nil
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*implantpb.Spite, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}

	if task.Cur == 0 {
		msg, ok := sess.GetLastMessage(int(task.Id))
		if ok {
			return msg, nil
		} else if task.Status != nil {
			return task.Status, nil
		}
		return nil, ErrNotFoundTaskContent
	} else {
		msg, ok := sess.GetMessage(int(task.Id), int(task.Cur))
		if ok {
			return msg, nil
		} else if task.Status != nil {
			return task.Status, nil
		}
		return nil, ErrNotFoundTaskContent
	}
}

func (rpc *Server) WaitTaskContent(ctx context.Context, req *clientpb.Task) (*implantpb.Spite, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}
	select {
	case <-task.Ctx.Done():
		msg, ok := sess.GetLastMessage(int(task.Id))
		if ok {
			return msg, nil
		} else if task.Status != nil {
			return task.Status, nil
		}
	}
	return nil, ErrNotFoundTaskContent
}

func (rpc *Server) GetAllTaskContent(ctx context.Context, req *clientpb.Task) ([]*implantpb.Spite, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}
	msgs, ok := sess.GetMessages(int(task.Id))
	if ok {
		return msgs, nil
	}
	return nil, ErrNotFoundTask
}

func (rpc *Server) GetTaskDescs(ctx context.Context, req *clientpb.Session) (*clientpb.TaskDescs, error) {
	resp := &clientpb.TaskDescs{
		Tasks: []*clientpb.TaskDesc{},
	}
	tasks, err := db.GetAllTasks(req.SessionId)
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		resp.Tasks = append(resp.Tasks, task.ToDescProtobuf())
	}

	return resp, nil
}
