package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"os"
)

func SyncCommand(con *console.Console) []*grumble.Command {
	return []*grumble.Command{{
		Name: "sync",
		Help: "sync file",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "filename")
			f.String("t", "target", "", "filepath")
		},
		Run: func(ctx *grumble.Context) error {
			sync(ctx, con)
			return nil
		},
	}}
}

func sync(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	name := ctx.Flags.String("name")
	target := ctx.Flags.String("target")
	spinner := styles.SpinnerModel{}
	go func() {
		syncResp, err := con.Rpc.Sync(con.ActiveTarget.Context(), &clientpb.Sync{
			Name:   name,
			Target: target,
		})
		if err != nil {
			console.Log.Errorf("Can't syncResp file: %s", err)
			spinner.Quitting = true
			return
		}
		file, err := os.OpenFile(syncResp.Target, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			console.Log.Errorf("Can't Open file: %s", err)
			spinner.Quitting = true
			return
		}
		_, err = file.Write(syncResp.Content)
		if err != nil {
			console.Log.Errorf("Can't write file: %s", err)
			spinner.Quitting = true
			return
		}
		spinner.Quitting = true
	}()
	_, err := spinner.Run()
	if err != nil {
		console.Log.Errorf("Console has an error: %s", err)
		os.Exit(1)
	}
}
