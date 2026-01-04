package sessions

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *core.Console) error {
	sid := cmd.Flags().Arg(0)

	// Try exact match first
	if session, err := con.GetOrUpdateSession(sid); err == nil {
		return Use(con, session)
	}

	// Exact match failed, try prefix match
	session, err := findSessionByPrefix(con, sid)
	if err != nil {
		return err
	}

	return Use(con, session)
}

// findSessionByPrefix finds session by prefix, returns error if multiple matches
func findSessionByPrefix(con *core.Console, prefix string) (*client.Session, error) {
	var matches []*client.Session
	var matchIDs []string

	for id, sess := range con.Sessions {
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, sess)
			matchIDs = append(matchIDs, id[:8]) // 只显示前 8 位
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

func Use(con *core.Console, sess *client.Session) error {
	err := addon.RefreshAddonCommand(sess.Addons, con)
	if err != nil {
		return err
	}
	con.SwitchImplant(sess, consts.CalleeCMD)
	count := con.RefreshCmd(sess)
	con.Log.Importantf("os: %s, arch: %s, process: %d %s, pipeline: %s\n", sess.Os.Name, sess.Os.Arch, sess.Process.Ppid, sess.Process.Name, sess.PipelineId)
	con.Log.Importantf("%d modules, %d available cmds, %d addons\n", len(sess.Modules), count, len(sess.Addons))
	con.Log.Infof("Active session %s (%s), group: %s\n", sess.Note, sess.SessionId, sess.GroupName)
	return nil
}
