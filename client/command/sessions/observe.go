package sessions

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

func ObserveCmd(cmd *cobra.Command, con *repl.Console) {
	var session *repl.Session
	isList, _ := cmd.Flags().GetBool("list")
	if isList {
		for i, ob := range con.Observers {
			repl.Log.Infof("%s: %s", i, ob.SessionId())
		}
		return
	}

	idArg := cmd.Flags().Args()
	if idArg == nil {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			for i, ob := range con.Observers {
				repl.Log.Infof("%d: %s", i, ob.SessionId())
			}
			return
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			repl.Log.Warn(repl.ErrNotFoundSession.Error())
		}
		isRemove, _ := cmd.Flags().GetBool("remove")
		if isRemove {
			con.RemoveObserver(session.SessionId)
		} else {
			con.AddObserver(session)
		}
	}
}
