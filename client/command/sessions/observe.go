package sessions

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func ObserveCmd(cmd *cobra.Command, con *core.Console) error {
	var session *client.Session
	isList, _ := cmd.Flags().GetBool("list")
	if isList {
		for i, ob := range con.Observers {
			con.Log.Infof("%s: %s\n", i, ob.SessionId)
		}
		return nil
	}

	idArg := cmd.Flags().Args()
	if len(idArg) == 0 {
		if con.GetInteractive() != nil {
			idArg = []string{con.GetInteractive().SessionId}
		} else {
			var i int
			for _, ob := range con.Observers {
				con.Log.Infof("%d: %s\n", i, ob.SessionId)
				i++
			}
			return nil
		}
	}
	for _, sid := range idArg {
		session = con.Sessions[sid]

		if session == nil {
			con.Log.Warn(core.ErrNotFoundSession.Error())
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
