package file

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"path/filepath"
)

func DownloadCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	task, err := Download(con.Rpc, session, path)
	if err != nil {
		con.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		return "Downloaded file " + path, nil
	})
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
