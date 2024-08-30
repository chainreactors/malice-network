package file

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"

	"google.golang.org/protobuf/proto"
)

func DownloadCmd(cmd *cobra.Command, con *console.Console) {

	name := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)

	download(name, path, con)
}

func download(name string, path string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	downloadTask, err := con.Rpc.Download(con.ActiveTarget.Context(), &implantpb.DownloadRequest{
		Name: name,
		Path: path,
	})
	if err != nil {
		console.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(downloadTask.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Importantf("Downloaded file %s from %s", name, path)
	})
}
