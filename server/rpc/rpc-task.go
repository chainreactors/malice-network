package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
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
	return nil, errs.ErrNotFoundTaskContent
}

func (rpc *Server) GetTasks(ctx context.Context, req *clientpb.TaskRequest) (*clientpb.Tasks, error) {
	if req.All {
		tasks, err := db.GetTasksByID(req.SessionId)
		if err != nil {
			return nil, err
		}
		return tasks, err
	} else {
		sess, err := core.Sessions.Get(req.SessionId)
		if err != nil {
			return nil, errs.ErrNotFoundSession
		}
		return sess.Tasks.ToProtobuf(), nil
	}
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, errs.ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}

	return getTaskContext(sess, task, req.Need)
}

func (rpc *Server) WaitTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, errs.ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}

	if int(req.Need) > task.Total {
		return nil, errs.ErrTaskIndexExceed
	}

	content, err := getTaskContext(sess, task, req.Need)
	if err == nil {
		return content, nil
	}

	for {
		select {
		case _, ok := <-task.DoneCh:
			if !ok {
				return nil, errs.ErrNotFoundTaskContent
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
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, errs.ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
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
	return nil, errs.ErrNotFoundTaskContent
}

func (rpc *Server) GetAllTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContexts, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, errs.ErrNotFoundSession
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}
	msgs, ok := sess.GetMessages(int(task.Id))
	if ok {
		return &clientpb.TaskContexts{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobuf(),
			Spites:  msgs,
		}, nil
	}
	return nil, errs.ErrNotFoundTask
}

func (rpc *Server) GetTaskFiles(ctx context.Context, req *clientpb.Session) (*clientpb.Files, error) {
	resp := &clientpb.Files{
		Files: []*clientpb.File{},
	}
	files, err := db.GetFilesBySessionID(req.SessionId)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		resp.Files = append(resp.Files, file.ToFileProtobuf())
	}

	return resp, nil
}

func (rpc *Server) GetAllDownloadFiles(ctx context.Context, req *clientpb.Empty) (*clientpb.Files, error) {
	resp := &clientpb.Files{
		Files: []*clientpb.File{},
	}
	files, err := db.GetAllDownloadFiles()
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		resp.Files = append(resp.Files, file.ToFileProtobuf())
	}

	return resp, nil
}

func (rpc *Server) CancelTask(ctx context.Context, req *implantpb.ImplantTask) (*clientpb.Task, error) {
	sess, err := getSession(ctx)
	if err != nil {
		return nil, err
	}
	task := sess.Tasks.Get(req.TaskId)
	if task == nil {
		return nil, errs.ErrNotFoundTask
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgEmpty, func(spite *implantpb.Spite) {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventTask,
			Op:        consts.CtrlTaskCancel,
			Task:      task.ToProtobuf(),
		})
	})

	return greq.Task.ToProtobuf(), nil
}
