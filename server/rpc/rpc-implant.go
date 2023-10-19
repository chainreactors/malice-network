package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) Register(ctx context.Context, req *lispb.RegisterSession) (*commonpb.Empty, error) {
	sess := &core.Session{
		ID:         hash.Md5Hash([]byte(req.SessionId)),
		ListenerId: req.ListenerId,
		RemoteAddr: req.RemoteAddr,
		Os:         req.RegisterData.Os,
		Process:    req.RegisterData.Process,
		Timer:      req.RegisterData.Timer,
	}
	core.Sessions.Add(sess)
	logs.Log.Importantf("init new session %s from %s", sess.ID, sess.ListenerId)
	return &commonpb.Empty{}, nil
}

func (rpc *Server) Ping(ctx context.Context, req *commonpb.Ping) (*commonpb.Empty, error) {
	fmt.Println(req)
	return &commonpb.Empty{}, nil
}
