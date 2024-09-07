package file

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"path/filepath"

	"google.golang.org/protobuf/proto"
)

func DownloadCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	task, err := Download(con.Rpc, session, path)
	if err != nil {
		repl.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		session.Log.Importantf("Downloaded file %s ", path)
	})
}

func Download(rpc clientrpc.MaliceRPCClient, session *repl.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Download(repl.Context(session), &implantpb.DownloadRequest{
		Name: filepath.Base(path),
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
