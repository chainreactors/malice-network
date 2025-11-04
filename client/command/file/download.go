package file

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
	"path/filepath"
)

func DownloadCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	is_dir, _ := cmd.Flags().GetBool("dir")
	task, err := Download(con.Rpc, session, path, is_dir)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Download(rpc clientrpc.MaliceRPCClient, session *session.Session, path string, is_dir bool) (*clientpb.Task, error) {
	task, err := rpc.Download(session.Context(), &implantpb.DownloadRequest{
		Name: filepath.Base(path),
		Path: path,
		Dir:  is_dir,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
