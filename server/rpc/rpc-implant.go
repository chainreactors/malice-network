package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"

	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func newSession(reg *clientpb.RegisterSession) (*core.Session, error) {
	sess := core.NewSession(reg)
	d := db.Session().Create(models.ConvertToSessionDB(sess))
	return sess, d.Error
}

func (rpc *Server) Register(ctx context.Context, req *clientpb.RegisterSession) (*clientpb.Empty, error) {
	sess, ok := core.Sessions.Get(req.SessionId)
	if ok {
		logs.Log.Infof("alive session %s re-register", sess.ID)
		sess.Update(req)
		err := db.UpdateSessionInfo(sess)
		if err != nil {
			logs.Log.Errorf("update session %s info failed in db, %s", sess.ID, err.Error())
		}
		return &clientpb.Empty{}, nil
	}

	// 如果内存中不存在, 则尝试从数据库中恢复
	reqsess, err := db.FindSession(req.SessionId)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	} else if errors.Is(err, db.ErrRecordNotFound) {
		// new session and save to db
		sess, err = newSession(req)
		if err != nil {
			return nil, err
		} else {
			sess.Publish(consts.CtrlSessionRegister, fmt.Sprintf("session %s from %s start at %s", sess.ID, sess.Target, sess.PipelineID))
			logs.Log.Importantf("init new session %s from %s", sess.ID, sess.PipelineID)
		}
	} else {
		// 数据库中已存在, update
		sess = core.NewSession(reqsess)
		logs.Log.Warnf("session %s re-register ", sess.ID)
		_, taskID, err := db.FindTaskAndMaxTasksID(req.SessionId)
		if err != nil {
			logs.Log.Errorf("cannot find max task id , %s ", err.Error())
			return &clientpb.Empty{}, nil
		}
		sess.SetLastTaskId(uint32(taskID))
		sess.Publish(consts.CtrlSessionReRegister, fmt.Sprintf("session %s from %s re-register at %s", sess.ID, sess.Target, sess.PipelineID))
	}
	core.Sessions.Add(sess)
	sess.Load()
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SysInfo(ctx context.Context, req *implantpb.SysInfo) (*clientpb.Empty, error) {
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	sess, ok := core.Sessions.Get(id)
	if !ok {
		return nil, nil
	}
	sess.UpdateSysInfo(req)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Checkin(ctx context.Context, req *implantpb.Ping) (*clientpb.Empty, error) {
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	if s, ok := core.Sessions.Get(id); !ok {
		sess, err := db.FindSession(id)
		if err != nil {
			return nil, err
		}
		newSess := core.NewSession(sess)
		_, taskID, err := db.FindTaskAndMaxTasksID(id)
		if err != nil {
			logs.Log.Errorf("cannot find max task id , %s ", err.Error())
		}
		newSess.SetLastTaskId(uint32(taskID))
		core.Sessions.Add(newSess)
		newSess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("session %s from %s reborn at %s", newSess.ID, newSess.Target, newSess.PipelineID))
		newSess.Load()
		logs.Log.Debugf("recover session %s", id)
	} else {
		s.UpdateLastCheckin()
	}

	err = db.UpdateLast(id)
	if err != nil {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

// sleep
func (rpc *Server) Sleep(ctx context.Context, req *implantpb.Timer) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Suicide(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) InitBindSession(ctx context.Context, req *implantpb.Request) (*clientpb.Empty, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	_, err = rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
