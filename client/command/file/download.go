package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"google.golang.org/protobuf/proto"
)

func download(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	name := ctx.Flags.String("name")
	path := ctx.Flags.String("path")
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
