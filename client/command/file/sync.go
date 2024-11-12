package file

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

func SyncCmd(cmd *cobra.Command, con *repl.Console) error {
	tid := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	syncTask, err := con.Rpc.Sync(con.ActiveTarget.Context(), &clientpb.Sync{
		FileId: session.SessionId + "-" + tid,
	})
	if err != nil {
		return err
	}
	path := filepath.Join(assets.GetTempDir(), syncTask.Name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(syncTask.Content)
	if err != nil {
		return err
	}
	con.Log.Infof("sync file in path: %s\n", path)
	return nil
}
