package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func getTaskContext(sess *core.Session, task *core.Task, index int32) (*clientpb.TaskContext, error) {
	var msg *implantpb.Spite
	var ok bool
	if index == -1 {
		msg, ok = sess.GetLastMessage(int(task.Id))
	} else {
		msg, ok = sess.GetMessage(int(task.Id), int(index))
	}

	if ok {
		return &clientpb.TaskContext{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobuf(),
			Spite:   msg,
		}, nil
	}
	return nil, ErrNotFoundTaskContent
}

func (rpc *Server) GetTasks(ctx context.Context, session *clientpb.Session) (*clientpb.Tasks, error) {
	sess, ok := core.Sessions.Get(session.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}

	return sess.Tasks.ToProtobuf(), nil
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}

	return getTaskContext(sess, task, req.Need)
}

func (rpc *Server) WaitTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, ErrNotFoundTask
	}

	if int(req.Need) > task.Total {
		return nil, ErrTaskIndexExceed
	}

	content, err := getTaskContext(sess, task, req.Need)
	if err == nil {
		return content, nil
	}

	for {
		select {
		case _, ok := <-task.DoneCh:
			if !ok {
				return nil, ErrNotFoundTaskContent
			}
			content, err = getTaskContext(sess, task, req.Need)
			if err != nil {
				continue
			}
			return content, nil
		}
	}
}

func (rpc *Server) WaitTaskFinish(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
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
			return &clientpb.TaskContext{
				Task:    task.ToProtobuf(),
				Session: sess.ToProtobuf(),
				Spite:   msg,
			}, nil
		}
	}
	return nil, ErrNotFoundTaskContent
}

func (rpc *Server) GetAllTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContexts, error) {
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
		return &clientpb.TaskContexts{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobuf(),
			Spites:  msgs,
		}, nil
	}
	return nil, ErrNotFoundTask
}

func (rpc *Server) GetTaskFiles(ctx context.Context, req *clientpb.Session) (*clientpb.Files, error) {
	resp := &clientpb.Files{
		Files: []*clientpb.File{},
	}
	tasks, err := db.GetAllTasks(req.SessionId)
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		resp.Files = append(resp.Files, task.ToFileProtobuf())
	}

	return resp, nil
}
