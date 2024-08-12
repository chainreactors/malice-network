package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"os"
)

func upload(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	path := ctx.Args.String("source")
	target := ctx.Args.String("destination")
	priv := ctx.Flags.Int("priv")
	hidden := ctx.Flags.Bool("hidden")
	data, err := os.ReadFile(path)
	if err != nil {
		con.SessionLog(sid).Errorf("Can't open file: %s", err)
	}
	uploadTask, err := con.Rpc.Upload(con.ActiveTarget.Context(), &implantpb.UploadRequest{
		Name:   filepath.Base(path),
		Target: target,
		Priv:   uint32(priv),
		Data:   data,
		Hidden: hidden,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Download error: %v", err)
		return
	}
	total := uploadTask.Total
	cur := uploadTask.Cur
	con.AddCallback(uploadTask.TaskId, func(msg proto.Message) {
		cur++
		barModel := tui.NewBar()
		barModel.SetProgressPercent(float64(cur) / float64(total))
		//err := tui.Run(barModel)
		//if err != nil {
		//	con.SessionLog(sid).Errorf("Error running bar: %v", err)
		//}
	})
}
