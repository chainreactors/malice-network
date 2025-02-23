package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
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
	"strconv"
)

func (rpc *Server) GetSessions(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Sessions, error) {
	var sessions *clientpb.Sessions
	var err error
	if req.All {
		sessions, err = db.FindAllSessions()
		if err != nil {
			return nil, err
		}
	} else {
		sessions = &clientpb.Sessions{
			Sessions: make([]*clientpb.Session, 0),
		}
		for _, session := range core.Sessions.All() {
			sessions.Sessions = append(sessions.Sessions, session.ToProtobuf())
		}
	}

	return sessions, nil
}

func (rpc *Server) GetSession(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Session, error) {
	session, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, err
	}
	dbSess, err := db.FindSession(req.SessionId)
	if err != nil {
		return nil, err
	} else if dbSess == nil {
		return nil, nil
	}
	session, err = core.RecoverSession(dbSess)
	if err != nil {
		return nil, err
	}
	core.Sessions.Add(session)
	return session.ToProtobuf(), nil
}

func (rpc *Server) SessionManage(ctx context.Context, req *clientpb.BasicUpdateSession) (*clientpb.Empty, error) {
	switch req.Op {
	case "delete":
		core.Sessions.Remove(req.SessionId)
		err := db.RemoveSession(req.SessionId)
		if err != nil {
			return nil, err
		}
	case "note":
		session, err := core.Sessions.Get(req.SessionId)
		if err != nil {
			return nil, err
		}
		session.Name = req.Arg
		err = db.UpdateSession(req.SessionId, req.Arg, "")
		if err != nil {
			return nil, err
		}
	case "group":
		session, err := core.Sessions.Get(req.SessionId)
		if err != nil {
			return nil, err
		}
		session.Group = req.Arg
		err = db.UpdateSession(req.SessionId, "", req.Arg)
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

func (rpc *Server) GetSessionHistory(ctx context.Context, req *clientpb.Int) (*clientpb.TasksContext, error) {
	sid, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	var tid int
	contexts := &clientpb.TasksContext{
		Contexts: make([]*clientpb.TaskContext, 0),
	}

	taskDir := filepath.Join(configs.ContextPath, sid, consts.TaskPath)
	session, err := core.Sessions.Get(sid)
	if err != nil {
		_, taskId, err := db.FindTaskAndMaxTasksID(sid)
		if err != nil {
			return nil, err
		}
		tid = int(taskId)
	} else {
		tid = int(session.Taskseq)
	}

	startTaskID := tid - int(req.Limit) + 1
	if startTaskID < 1 {
		startTaskID = 1
	}
	endTaskID := tid
	re := regexp.MustCompile(`^(\d+)_(\d+)$`)

	taskIDs := make(map[int]struct{})
	for i := startTaskID; i <= endTaskID; i++ {
		taskIDs[i] = struct{}{}
	}

	files, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			matches := re.FindStringSubmatch(file.Name())
			if matches == nil {
				continue
			}

			taskID, _ := strconv.Atoi(matches[1])

			if _, exists := taskIDs[taskID]; exists {
				taskPath := filepath.Join(taskDir, file.Name())
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

				taskIDStr := sid + "-" + strconv.Itoa(taskID)
				task, err := db.GetTaskPB(taskIDStr)
				if err != nil {
					return nil, err
				}
				session, err := db.FindSession(sid)
				if err != nil {
					return nil, err
				}
				contexts.Contexts = append(contexts.Contexts, &clientpb.TaskContext{
					Task:    task,
					Session: session.ToProtobuf(),
					Spite:   spite,
				})
			}
		}
	}

	return contexts, nil
}

func (rpc *Server) Ping(ctx context.Context, req *implantpb.Ping) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgPing)
	return greq.Task.ToProtobuf(), nil
}
