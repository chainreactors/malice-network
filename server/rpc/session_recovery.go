package rpc

import "github.com/chainreactors/malice-network/server/internal/core"

func activateRecoveredSession(session *core.Session) {
	if session == nil {
		return
	}
	core.Sessions.Add(session)
	session.PushCtrl()
}
