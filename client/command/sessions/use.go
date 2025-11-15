package sessions

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func UseSessionCmd(cmd *cobra.Command, con *repl.Console) error {
	var session *client.Session
	sid := cmd.Flags().Arg(0)
	session, err := con.GetOrUpdateSession(sid)
	if err != nil {
		return err
	}

	return Use(con, session)
}

func Use(con *repl.Console, sess *client.Session) error {
	err := addon.RefreshAddonCommand(sess.Addons, con)
	if err != nil {
		return err
	}
	con.SwitchImplant(sess)
	count := con.RefreshCmd(sess)
	con.Log.Importantf("os: %s, arch: %s, process: %d %s, pipeline: %s\n", sess.Os.Name, sess.Os.Arch, sess.Process.Ppid, sess.Process.Name, sess.PipelineId)
	con.Log.Importantf("%d modules, %d available cmds, %d addons\n", len(sess.Modules), count, len(sess.Addons))
	con.Log.Infof("Active session %s (%s), group: %s\n", sess.Note, sess.SessionId, sess.GroupName)
	return nil
}
