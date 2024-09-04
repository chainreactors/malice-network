package file

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"path/filepath"

	"google.golang.org/protobuf/proto"
)

func DownloadCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(1)
	download(path, con)
}

func download(path string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	downloadTask, err := con.Rpc.Download(con.ActiveTarget.Context(), &implantpb.DownloadRequest{
		Name: filepath.Base(path),
		Path: path,
	})
	if err != nil {
		console.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(downloadTask.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Importantf("Downloaded file %s ", path)
	})
}
