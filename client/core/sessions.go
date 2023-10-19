package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

var (
	Sessions sessions = make(sessions)
)

type sessions map[string]*clientpb.Session

func (s sessions) Update(con *console.Console) {
	sessions, err := con.Rpc.GetSessions(context.Background(), &clientpb.Empty{})
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return
	}

	if len(sessions.GetSessions()) == 0 {
		return
	}

	for _, session := range sessions.GetSessions() {
		Sessions[session.SessionId] = session
	}
}
