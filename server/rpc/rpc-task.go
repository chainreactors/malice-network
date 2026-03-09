package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var taskFileNamePattern = regexp.MustCompile(`^(\d+)_(\d+)$`)

type taskSpiteEntry struct {
	Index int
	Spite *implantpb.Spite
}

func readTaskSpitesFromDisk(sessionID string, taskID uint32) ([]taskSpiteEntry, error) {
	taskDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sessionID, consts.TaskPath))
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	entries := make([]taskSpiteEntry, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := taskFileNamePattern.FindStringSubmatch(file.Name())
		if matches == nil {
			continue
		}

		fileTaskID, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil || uint32(fileTaskID) != taskID {
			continue
		}

		index, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		taskPath := filepath.Join(taskDir, file.Name())
		content, err := os.ReadFile(taskPath)
		if err != nil {
			logs.Log.Warnf("failed to read task file %s: %v", taskPath, err)
			continue
		}

		spite := &implantpb.Spite{}
		if err = proto.Unmarshal(content, spite); err != nil {
			logs.Log.Warnf("failed to unmarshal task file %s: %v", taskPath, err)
			continue
		}

		entries = append(entries, taskSpiteEntry{Index: index, Spite: spite})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Index < entries[j].Index
	})

	return entries, nil
}

func getTaskContextFromDisk(sess *core.Session, task *core.Task, index int32) (*clientpb.TaskContext, error) {
	entries, err := readTaskSpitesFromDisk(sess.ID, task.Id)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}

	var spite *implantpb.Spite
	if index == -1 {
		spite = entries[len(entries)-1].Spite
	} else {
		for _, entry := range entries {
			if entry.Index == int(index) {
				spite = entry.Spite
				break
			}
		}
	}

	if spite == nil {
		return nil, types.ErrNotFoundTaskContent
	}

	return &clientpb.TaskContext{
		Task:    task.ToProtobuf(),
		Session: sess.ToProtobufLite(),
		Spite:   spite,
	}, nil
}

func getAllTaskContextsFromDisk(sess *core.Session, task *core.Task) (*clientpb.TaskContexts, error) {
	entries, err := readTaskSpitesFromDisk(sess.ID, task.Id)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, types.ErrNotFoundTaskContent
	}

	spites := make([]*implantpb.Spite, 0, len(entries))
	for _, entry := range entries {
		spites = append(spites, entry.Spite)
	}

	return &clientpb.TaskContexts{
		Task:    task.ToProtobuf(),
		Session: sess.ToProtobufLite(),
		Spites:  spites,
	}, nil
}

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
			Session: sess.ToProtobufLite(),
			Spite:   msg,
		}, nil
	}
	return nil, types.ErrNotFoundTaskContent
}

func (rpc *Server) GetTasks(ctx context.Context, req *clientpb.TaskRequest) (*clientpb.Tasks, error) {
	if req.All {
		modelTasks, err := db.ListTasksBySession(req.SessionId)
		if err != nil {
			return nil, err
		}
		return modelTasks.ToProtobuf(), nil
	} else {
		sess, err := core.Sessions.Get(req.SessionId)
		if err != nil {
			return nil, types.ErrNotFoundSession
		}
		return sess.Tasks.ToProtobuf(), nil
	}
}

func (rpc *Server) GetTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, types.ErrNotFoundSession
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	content, err := getTaskContext(sess, task, req.Need)
	if err == nil {
		return content, nil
	}
	if !errors.Is(err, types.ErrNotFoundTaskContent) {
		return nil, err
	}

	content, err = getTaskContextFromDisk(sess, task, req.Need)
	if err == nil {
		return content, nil
	}
	if !errors.Is(err, types.ErrNotFoundTaskContent) {
		return nil, err
	}

	return nil, types.ErrNotFoundTaskContent
}

func (rpc *Server) WaitTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContext, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, types.ErrNotFoundSession
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	if int(req.Need) > task.Total {
		return nil, types.ErrTaskIndexExceed
	}

	content, err := getTaskContext(sess, task, req.Need)
	if err == nil {
		return content, nil
	}

	for {
		select {
		case _, ok := <-task.DoneCh:
			if !ok {
				return nil, types.ErrNotFoundTaskContent
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
		return nil, types.ErrNotFoundSession
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	select {
	case <-task.Ctx.Done():
		msg, ok := sess.GetLastMessage(int(task.Id))
		if ok {
			return &clientpb.TaskContext{
				Task:    task.ToProtobuf(),
				Session: sess.ToProtobufLite(),
				Spite:   msg,
			}, nil
		}

		content, err := getTaskContextFromDisk(sess, task, -1)
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, types.ErrNotFoundTaskContent) {
			return nil, err
		}
	}
	return nil, types.ErrNotFoundTaskContent
}

func (rpc *Server) GetAllTaskContent(ctx context.Context, req *clientpb.Task) (*clientpb.TaskContexts, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, types.ErrNotFoundSession
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}
	msgs, ok := sess.GetMessages(int(task.Id))
	if ok {
		return &clientpb.TaskContexts{
			Task:    task.ToProtobuf(),
			Session: sess.ToProtobufLite(),
			Spites:  msgs,
		}, nil
	}

	contexts, err := getAllTaskContextsFromDisk(sess, task)
	if err == nil {
		return contexts, nil
	}
	if errors.Is(err, types.ErrNotFoundTaskContent) {
		return nil, types.ErrNotFoundTask
	}
	return nil, fmt.Errorf("load task content from disk: %w", err)
}

func (rpc *Server) GetFiles(ctx context.Context, req *clientpb.Session) (*clientpb.Files, error) {
	files, err := db.GetDownloadFiles(req.SessionId)
	if err != nil {
		return nil, err
	}
	return &clientpb.Files{
		Files: files,
	}, nil
}

func (rpc *Server) CancelTask(ctx context.Context, req *implantpb.TaskCtrl) (*clientpb.Task, error) {
	sess, err := getSession(ctx)
	if err != nil {
		return nil, err
	}
	task := sess.Tasks.GetOrRecover(sess, req.TaskId)
	if task == nil {
		return nil, types.ErrNotFoundTask
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty, func(spite *implantpb.Spite) {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventTask,
			Op:        consts.CtrlTaskCancel,
			Task:      task.ToProtobuf(),
		})
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ListTasks(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleListTask)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgTasks)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) QueryTask(ctx context.Context, req *implantpb.TaskCtrl) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgTask)
	return greq.Task.ToProtobuf(), nil
}
