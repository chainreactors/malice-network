package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) Register(ctx context.Context, req *lispb.RegisterSession) (*implantpb.Empty, error) {
	_, success := core.Sessions.Get(req.SessionId)
	if success == true {
		return &implantpb.Empty{}, nil
	}
	sess := core.NewSession(req)
	core.Sessions.Add(sess)
	dbSession := db.Session()
	d := dbSession.Create(models.ConvertToSessionDB(sess))
	if d.Error != nil {
		logs.Log.Warnf("session %s re-register ", sess.ID)
		return &implantpb.Empty{}, nil
	}
	err := sess.Load(sess.CachePath)
	if err != nil {
		return &implantpb.Empty{}, nil
	}
	logs.Log.Importantf("init new session %s from %s", sess.ID, sess.ListenerId)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) Ping(ctx context.Context, req *implantpb.Ping) (*implantpb.Empty, error) {
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := core.Sessions.Get(id); !ok {
		// 如果内存中不存在, 则从数据库中恢复
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
		newSess.Load(newSess.CachePath)
		logs.Log.Debugf("recover session %s", id)
	}

	err = db.UpdateLast(id)
	if err != nil {
		return nil, err
	}

	return &implantpb.Empty{}, nil
}
