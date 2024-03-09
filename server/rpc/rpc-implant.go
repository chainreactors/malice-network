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
	sess := core.NewSession(req)
	core.Sessions.Add(sess)
	dbSession := db.Session()
	d := dbSession.Create(models.ConvertToSessionDB(sess))
	if d.Error != nil {
		logs.Log.Warnf("session %s re-register ", sess.ID)
		return &implantpb.Empty{}, nil
	}
	logs.Log.Importantf("init new session %s from %s", sess.ID, sess.ListenerId)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) Ping(ctx context.Context, req *implantpb.Empty) (*implantpb.Empty, error) {
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
		core.Sessions.Add(core.NewSession(sess))
		logs.Log.Debugf("recover session %s", id)
	}

	err = db.UpdateLast(id)
	if err != nil {
		return nil, err
	}

	return &implantpb.Empty{}, nil
}
