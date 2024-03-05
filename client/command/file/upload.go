package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"google.golang.org/protobuf/proto"
	"os"
)

func upload(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	name := ctx.Flags.String("name")
	path := ctx.Flags.String("path")
	target := ctx.Flags.String("target")
	priv := ctx.Flags.Int("priv")
	hidden := ctx.Flags.Bool("hidden")
	data, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("Can't open file: %s", err)
	}
	uploadTask, err := con.Rpc.Upload(con.ActiveTarget.Context(), &implantpb.UploadRequest{
		Name:   name,
		Target: target,
		Priv:   uint32(priv),
		Data:   data,
		Hidden: hidden,
	})
	if err != nil {
		console.Log.Errorf("Download error: %v", err)
		return
	}
	con.AddCallback(uploadTask.TaskId, func(msg proto.Message) {
	})
}
