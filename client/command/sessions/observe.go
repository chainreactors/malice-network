package sessions

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func ObserveCmd(cmd *cobra.Command, con *console.Console) {
	var session *clientpb.Session
	isList, _ := cmd.Flags().GetBool("list")
	if isList {
		for i, ob := range con.Observers {
			console.Log.Infof("%d: %s", i, ob.SessionId())
		}
		return
	}

	idArg := cmd.Flags().Args()
	if idArg == nil {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			for i, ob := range con.Observers {
				console.Log.Infof("%d: %s", i, ob.SessionId())
			}
			return
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			console.Log.Warn(console.ErrNotFoundSession.Error())
		}
		isRemove, _ := cmd.Flags().GetBool("remove")
		if isRemove {
			con.RemoveObserver(session.SessionId)
		} else {
			con.AddObserver(session)
		}
	}
}
