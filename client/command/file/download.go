package file

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"path/filepath"
)

func DownloadCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := Download(con.Rpc, session, path)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, "Downloaded file "+path)
	return nil
}

func Download(rpc clientrpc.MaliceRPCClient, session *core.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Download(session.Context(), &implantpb.DownloadRequest{
		Name: filepath.Base(path),
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
