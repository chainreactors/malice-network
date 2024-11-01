package sessions

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func ObserveCmd(cmd *cobra.Command, con *repl.Console) error {
	var session *core.Session
	isList, _ := cmd.Flags().GetBool("list")
	if isList {
		for i, ob := range con.Observers {
			con.Log.Infof("%s: %s", i, ob.SessionId())
		}
		return nil
	}

	idArg := cmd.Flags().Args()
	if idArg == nil {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			var i int
			for _, ob := range con.Observers {
				con.Log.Infof("%d: %s", i, ob.SessionId())
				i++
			}
			return nil
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			con.Log.Warn(repl.ErrNotFoundSession.Error())
			return nil
		}
		isRemove, _ := cmd.Flags().GetBool("remove")
		if isRemove {
			con.RemoveObserver(session.SessionId)
		} else {
			con.AddObserver(session)
		}
	}
	return nil
}
