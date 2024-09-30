package file

import (
	"github.com/spf13/cobra"
	"os"

	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

func SyncCmd(cmd *cobra.Command, con *repl.Console) {
	tid := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	syncTask, err := con.Rpc.Sync(con.ActiveTarget.Context(), &clientpb.Sync{
		FileId: session.SessionId + "-" + tid,
	})
	if err != nil {
		con.Log.Errorf("Can't sync file: %s", err)
		return
	}
	file, err := os.OpenFile(syncTask.Name, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		con.Log.Errorf("Can't Open file: %s", err)
		return
	}
	defer file.Close()
	_, err = file.Write(syncTask.Content)
	if err != nil {
		con.Log.Errorf("Can't write file: %s", err)
		return
	}
}
