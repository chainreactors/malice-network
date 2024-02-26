package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) Register(ctx context.Context, req *lispb.RegisterSession) (*commonpb.Empty, error) {
	sess := core.NewSession(req)
	core.Sessions.Add(sess)
	dbSession := db.Session()
	d := dbSession.Create(models.ConvertToSessionDB(sess))
	if d.Error != nil {
		logs.Log.Warnf("session %s re-register ", sess.ID)
		return &commonpb.Empty{}, nil
	}
	logs.Log.Importantf("init new session %s from %s", sess.ID, sess.ListenerId)
	return &commonpb.Empty{}, nil
}

func (rpc *Server) Ping(ctx context.Context, req *commonpb.Ping) (*commonpb.Empty, error) {
	//fmt.Println(req)
	return &commonpb.Empty{}, nil
}
