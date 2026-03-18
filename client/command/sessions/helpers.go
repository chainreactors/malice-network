package sessions

import (
	"errors"
	"fmt"

	"github.com/chainreactors/malice-network/client/core"
)

func shortSessionID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func resolveSessionID(con *core.Console, sid string) (string, error) {
	if sid == "" {
		if con.GetInteractive() == nil {
			return "", fmt.Errorf("no session selected")
		}
		return con.GetInteractive().Session.GetSessionId(), nil
	}

	if session, ok := con.Sessions[sid]; ok && session != nil {
		return session.SessionId, nil
	}

	session, err := findSessionByPrefix(con, sid)
	if err == nil {
		return session.SessionId, nil
	}
	if errors.Is(err, core.ErrNotFoundSession) {
		return sid, nil
	}
	return "", err
}
