package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

func UploadCommand(con *console.Console) []*grumble.Command {
	return []*grumble.Command{{
		Name: "upload",
		Help: "upload file",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "filename")
			f.String("p", "path", "", "filepath")
			f.String("t", "target", "", "file in implant target")
			f.Int("", "priv", 0o644, "file Privilege")
			f.Bool("", "hidden", false, "filename")
		},
		Run: func(ctx *grumble.Context) error {
			upload(ctx, con)
			return nil
		},
	}}
}

func upload(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	// TODO tui: 同download

	//name := ctx.Flags.String("name")
	//path := ctx.Flags.String("path")
	//target := ctx.Flags.String("target")
	//priv := ctx.Flags.Int("priv")
	//hidden := ctx.Flags.Bool("hidden")
	//data, err := os.ReadFile(path)
	//if err != nil {
	//	console.Log.Errorf("Can't open file: %s", err)
	//}
	//var download *clientpb.Task
	//ctrl := make(chan float64)
	//download, err = con.Rpc.Upload(con.ActiveTarget.Context(), &pluginpb.UploadRequest{
	//	Name:   name,
	//	Target: target,
	//	Priv:   uint32(priv),
	//	Data:   data,
	//	Hidden: hidden,
	//})
	//ctrl <- float64(download.Cur / download.Total)
	//go func() {
	//	m := tui.ProcessBarModel{
	//		Progress:        progress.New(progress.WithDefaultGradient()),
	//		ProgressPercent: <-ctrl,
	//	}
	//	m.Run()
	//}()
	//if err != nil {
	//	console.Log.Errorf("")
	//}
}
