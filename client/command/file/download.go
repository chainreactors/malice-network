package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
)

func DownloadCommand(con *console.Console) []*grumble.Command {
	return []*grumble.Command{{
		Name: "download",
		Help: "download file",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "filename")
			f.String("p", "path", "", "filepath")
		},
		Run: func(ctx *grumble.Context) error {
			download(ctx, con)
			return nil
		},
	}}
}

func download(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	name := ctx.Flags.String("name")
	path := ctx.Flags.String("path")
	downloadTask, err := con.Rpc.Download(con.ActiveTarget.Context(), &pluginpb.DownloadRequest{
		Name: name,
		Path: path,
	})
	if err != nil {
		console.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(downloadTask.TaskId, func(msg proto.Message) {
	})

}
