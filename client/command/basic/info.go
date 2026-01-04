package basic

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func SessionInfoCmd(cmd *cobra.Command, con *core.Console) error {
	var session *client.Session

	// If argument provided, use the specified session
	if len(cmd.Flags().Args()) > 0 {
		sessionID := cmd.Flags().Args()[0]
		sess, err := findSessionByPrefix(con, sessionID)
		if err != nil {
			return err
		}
		session = sess
	} else {
		// Otherwise use the current active session
		session = con.GetInteractive()
		if session == nil {
			return core.ErrNotFoundSession
		}
	}

	result := tui.RendStructDefault(session.Session, "Tasks")
	con.Log.Info("\n" + result)
	return nil
}

// findSessionByPrefix finds session by prefix, returns error if multiple matches
func findSessionByPrefix(con *core.Console, prefix string) (*client.Session, error) {
	// Try exact match first
	if sess, ok := con.Sessions[prefix]; ok {
		return sess, nil
	}

	// Prefix match
	var matches []*client.Session
	var matchIDs []string

	for id, sess := range con.Sessions {
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, sess)
			matchIDs = append(matchIDs, id[:8])
		}
	}

	switch len(matches) {
	case 0:
		return nil, core.ErrNotFoundSession
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous session prefix '%s', matches: %s", prefix, strings.Join(matchIDs, ", "))
	}
}
