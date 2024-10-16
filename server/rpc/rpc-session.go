package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
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

func (rpc *Server) GetSessions(ctx context.Context, _ *clientpb.Empty) (*clientpb.Sessions, error) {
	sessions := &clientpb.Sessions{}
	for _, session := range core.Sessions.All() {
		sessions.Sessions = append(sessions.Sessions, session.ToProtobuf())
	}
	return sessions, nil
}

func (rpc *Server) GetAlivedSessions(ctx context.Context, _ *clientpb.Empty) (*clientpb.Sessions, error) {
	var sessions []*clientpb.Session
	for _, session := range core.Sessions.All() {
		sessionProto := session.ToProtobuf()
		if sessionProto.IsAlive {
			sessions = append(sessions, session.ToProtobuf())
		}
	}
	return &clientpb.Sessions{Sessions: sessions}, nil
}

func (rpc *Server) GetSession(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Session, error) {
	session, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		return nil, ErrNotFoundSession
	}
	return session.ToProtobuf(), nil
}

func (rpc *Server) BasicSessionOP(ctx context.Context, req *clientpb.BasicUpdateSession) (*clientpb.Empty, error) {
	switch req.Op {
	case "delete":
		core.Sessions.Remove(req.SessionId)
		err := db.DeleteSession(req.SessionId)
		if err != nil {
			return nil, err
		}
	case "note":
		session, ok := core.Sessions.Get(req.SessionId)
		if !ok {
			return nil, ErrNotFoundSession
		}
		session.Name = req.Arg
		err := db.UpdateSession(req.SessionId, req.Arg, "")
		if err != nil {
			return nil, err
		}
	case "group":
		session, ok := core.Sessions.Get(req.SessionId)
		if !ok {
			return nil, ErrNotFoundSession
		}
		session.Group = req.Arg
		err := db.UpdateSession(req.SessionId, "", req.Arg)
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
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgSysInfo)
	return greq.Task.ToProtobuf(), nil
}

type taskFile struct {
	TaskID int
	Cur    int
	Name   string
}

func (rpc *Server) GetSessionLog(ctx context.Context, req *clientpb.SessionLog) (*clientpb.TasksContext, error) {
	var taskFiles []taskFile
	contexts := &clientpb.TasksContext{
		Contexts: make([]*clientpb.TaskContext, 0),
	}
	taskDir := filepath.Join(configs.LogPath, req.SessionId)
	re := regexp.MustCompile(`^(\d+)_(\d+)$`)

	files, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			matches := re.FindStringSubmatch(file.Name())
			if matches != nil {
				taskid, _ := strconv.Atoi(matches[1])
				cur, _ := strconv.Atoi(matches[2])
				taskFiles = append(taskFiles, taskFile{
					TaskID: taskid,
					Cur:    cur,
					Name:   file.Name(),
				})
			}
		}
	}

	if len(taskFiles) == 0 {
		return nil, errors.New("no log files found")
	}

	sort.Slice(taskFiles, func(i, j int) bool {
		if taskFiles[i].TaskID == taskFiles[j].TaskID {
			return taskFiles[i].Cur < taskFiles[j].Cur
		}
		return taskFiles[i].TaskID > taskFiles[j].TaskID
	})

	taskIDCount := 0
	lastTaskID := -1
	for _, f := range taskFiles {
		if f.TaskID != lastTaskID {
			taskIDCount++
			lastTaskID = f.TaskID
		}

		if taskIDCount > int(req.Limit) {
			break
		}

		taskPath := filepath.Join(taskDir, f.Name)
		content, err := os.ReadFile(taskPath)
		if err != nil {
			logs.Log.Errorf("Error reading file: %s", err)
			continue
		}
		spite := &implantpb.Spite{}
		err = proto.Unmarshal(content, spite)
		if err != nil {
			logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
			continue
		}

		taskID := req.SessionId + "-" + strconv.Itoa(f.TaskID)
		task, err := db.GetTaskPB(taskID)
		if err != nil {
			return nil, err
		}
		session, ok := core.Sessions.Get(req.SessionId)
		if !ok {
			return nil, ErrNotFoundSession
		}
		contexts.Contexts = append(contexts.Contexts, &clientpb.TaskContext{
			Task:    task,
			Session: session.ToProtobuf(),
			Spite:   spite,
		})
	}
	return contexts, nil
}
