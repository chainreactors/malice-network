package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) GetSessions(ctx context.Context, _ *clientpb.Empty) (*clientpb.Sessions, error) {
	resp := &clientpb.Sessions{
		Sessions: []*clientpb.Session{},
	}
	for _, session := range core.Sessions.All() {
		resp.Sessions = append(resp.Sessions, session.ToProtobuf())
	}
	return resp, nil
}
