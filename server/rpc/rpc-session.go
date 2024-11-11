package rpc

import (
	"context"
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
	session, ok := core.Sessions.Get(req.SessionId)
	if ok {
		return session.ToProtobuf(), nil
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

//type taskFile struct {
//	TaskID int
//	Cur    int
//	Name   string
//}

func (rpc *Server) GetSessionHistory(ctx context.Context, req *clientpb.SessionLog) (*clientpb.TasksContext, error) {
	var tid int
	contexts := &clientpb.TasksContext{
		Contexts: make([]*clientpb.TaskContext, 0),
	}
	taskDir := filepath.Join(configs.LogPath, req.SessionId)
	session, ok := core.Sessions.Get(req.SessionId)
	if !ok {
		_, taskId, err := db.FindTaskAndMaxTasksID(req.SessionId)
		if err != nil {
			return nil, err
		}
		tid = int(taskId)
	} else {
		tid = int(session.Taskseq)
	}

	startTaskID := max(1, tid-int(req.Limit)+1)
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

				taskIDStr := req.SessionId + "-" + strconv.Itoa(taskID)
				task, err := db.GetTaskPB(taskIDStr)
				if err != nil {
					return nil, err
				}
				session, err := db.FindSession(req.SessionId)
				if err != nil {
					return nil, err
				}
				if session == nil {
					logs.Log.Warnf("session %s has removed", req.SessionId)
					continue
				}
				contexts.Contexts = append(contexts.Contexts, &clientpb.TaskContext{
					Task:    task,
					Session: session,
					Spite:   spite,
				})
			}
		}
	}

	return contexts, nil
	//for _, file := range files {
	//	if !file.IsDir() {
	//		matches := re.FindStringSubmatch(file.Name())
	//		if matches != nil {
	//			taskid, _ := strconv.Atoi(matches[1])
	//			cur, _ := strconv.Atoi(matches[2])
	//			taskFiles = append(taskFiles, taskFile{
	//				TaskID: taskid,
	//				Cur:    cur,
	//				Name:   file.Name(),
	//			})
	//		}
	//	}
	//}
	//
	//if len(taskFiles) == 0 {
	//	return contexts, nil
	//}
	//
	//sort.Slice(taskFiles, func(i, j int) bool {
	//	if taskFiles[i].TaskID == taskFiles[j].TaskID {
	//		return taskFiles[i].Cur < taskFiles[j].Cur
	//	}
	//	return taskFiles[i].TaskID > taskFiles[j].TaskID
	//})
	//
	//taskIDCount := 0
	//lastTaskID := -1
	//for _, f := range taskFiles {
	//	if f.TaskID != lastTaskID {
	//		taskIDCount++
	//		lastTaskID = f.TaskID
	//	}
	//
	//	if taskIDCount > int(req.Limit) {
	//		break
	//	}
	//
	//	taskPath := filepath.Join(taskDir, f.Name)
	//	content, err := os.ReadFile(taskPath)
	//	if err != nil {
	//		logs.Log.Errorf("Error reading file: %s", err)
	//		continue
	//	}
	//	spite := &implantpb.Spite{}
	//	err = proto.Unmarshal(content, spite)
	//	if err != nil {
	//		logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
	//		continue
	//	}
	//
	//	taskID := req.SessionId + "-" + strconv.Itoa(f.TaskID)
	//	task, err := db.GetTaskPB(taskID)
	//	if err != nil {
	//		return nil, err
	//	}
	//	session, err := db.FindSession(req.SessionId)
	//	if err != nil {
	//		return nil, err
	//	}
	//	contexts.Contexts = append(contexts.Contexts, &clientpb.TaskContext{
	//		Task:    task,
	//		Session: session,
	//		Spite:   spite,
	//	})
	//}
	//return contexts, nil
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
