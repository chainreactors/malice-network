package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) Register(ctx context.Context, req *lispb.RegisterSession) (*implantpb.Empty, error) {
	sess, success := core.Sessions.Get(req.SessionId)
	if success {
		sess.Update(req)
		err := db.UpdateSessionInfo(sess)
		if err != nil {
			logs.Log.Errorf("update session %s info failed in db, %s", sess.ID, err.Error())
		}
		return &implantpb.Empty{}, nil
	}

	sess = core.NewSession(req)
	core.Sessions.Add(sess)
	dbSession := db.Session()
	d := dbSession.Create(models.ConvertToSessionDB(sess))
	if d.Error != nil {
		sess.Update(req)
		logs.Log.Warnf("session %s re-register ", sess.ID)
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventSession,
			Session:   sess,
			IsNotify:  true,
			Message:   fmt.Sprintf("session %s from %s re-register at %s", sess.ID, sess.RemoteAddr, sess.PipelineID),
		})
		return &implantpb.Empty{}, nil
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventSession,
			Session:   sess,
			IsNotify:  true,
			Message:   fmt.Sprintf("session %s from %s start at %s", sess.ID, sess.RemoteAddr, sess.PipelineID),
		})
		logs.Log.Importantf("init new session %s from %s", sess.ID, sess.PipelineID)
		return &implantpb.Empty{}, nil
	}
}

func (rpc *Server) SysInfo(ctx context.Context, req *implantpb.SysInfo) (*implantpb.Empty, error) {
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	sess, ok := core.Sessions.Get(id)
	if !ok {
		return nil, nil
	}
	sess.UpdateSysInfo(req)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) Ping(ctx context.Context, req *implantpb.Ping) (*implantpb.Empty, error) {
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
		newSess.Load()
		logs.Log.Debugf("recover session %s", id)
	} else {
		s.UpdateLastCheckin()
	}

	err = db.UpdateLast(id)
	if err != nil {
		return nil, err
	}

	return &implantpb.Empty{}, nil
}
