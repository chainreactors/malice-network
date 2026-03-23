package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/protobuf/proto"
)

func (rpc *Server) GetSessions(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Sessions, error) {
	var sessions *clientpb.Sessions
	if req.All {
		modelSessions, err := db.ListSessions()
		if err != nil {
			return nil, err
		}
		sessions = modelSessions.ToProtobuf()
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

func (rpc *Server) GetSessionCount(ctx context.Context, req *clientpb.Empty) (*clientpb.SessionCount, error) {
	alive := int32(len(core.Sessions.All()))
	total, err := db.NewSessionQuery().WhereRemoved(false).Count()
	if err != nil {
		return nil, err
	}
	return &clientpb.SessionCount{
		Alive: alive,
		Total: int32(total),
	}, nil
}

func (rpc *Server) GetSession(ctx context.Context, req *clientpb.SessionRequest) (*clientpb.Session, error) {
	session, err := core.Sessions.Get(req.SessionId)
	if err == nil {
		return session.ToProtobuf(), nil
	}
	// Session not in memory (dead/offline). Return DB data directly
	// without recovering to memory — only Checkin/Register should revive.
	dbSess, err := db.FindSession(req.SessionId)
	if err != nil {
		return nil, err
	}
	if dbSess == nil {
		return nil, nil
	}
	return dbSess.ToProtobuf(), nil
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
		if err == nil {
			session.Note = req.Arg
			if err = session.SaveAndNotify(fmt.Sprintf("session %s note updated to %s", req.SessionId, req.Arg)); err != nil {
				return nil, err
			}
		} else {
			err = db.UpdateSession(req.SessionId, req.Arg, "")
			if err != nil {
				return nil, err
			}
		}
	case "group":
		session, err := core.Sessions.Get(req.SessionId)
		if err == nil {
			session.Group = req.Arg
			if err = session.SaveAndNotify(fmt.Sprintf("session %s group updated to %s", req.SessionId, req.Arg)); err != nil {
				return nil, err
			}
		} else {
			err = db.UpdateSession(req.SessionId, "", req.Arg)
			if err != nil {
				return nil, err
			}
		}
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) Info(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleSysInfo, types.MsgSysInfo, func(greq *GenericRequest, spite *implantpb.Spite) {
		greq.Session.UpdateSysInfo(spite.GetSysinfo())
		greq.Session.SaveAndNotify("")
	})
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

	taskDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sid, consts.TaskPath))
	if err != nil {
		return nil, err
	}
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
		if os.IsNotExist(err) {
			return contexts, nil
		}
		return nil, err
	}

	// Sort files numerically by taskID_index (os.ReadDir returns lexicographic order
	// which puts "10_0" before "2_0")
	sort.Slice(files, func(i, j int) bool {
		mi := re.FindStringSubmatch(files[i].Name())
		mj := re.FindStringSubmatch(files[j].Name())
		if mi == nil || mj == nil {
			return files[i].Name() < files[j].Name()
		}
		ti, _ := strconv.Atoi(mi[1])
		tj, _ := strconv.Atoi(mj[1])
		if ti != tj {
			return ti < tj
		}
		ii, _ := strconv.Atoi(mi[2])
		ij, _ := strconv.Atoi(mj[2])
		return ii < ij
	})

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
				modelTask, err := db.GetTask(taskIDStr)
				if err != nil {
					return nil, err
				}
				task := modelTask.ToProtobuf()
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

	greq.HandlerResponse(ch, types.MsgPing)
	return greq.Task.ToProtobuf(), nil
}
