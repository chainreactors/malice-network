package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func UploadCommand() {

}

func Upload(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	var err error
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
	var download *clientpb.Task
	ctrl := make(chan float64)
	download, err = con.Rpc.Upload(con.ActiveTarget.Context(), &pluginpb.UploadRequest{
		Name:   name,
		Target: target,
		Priv:   uint32(priv),
		Data:   data,
		Hidden: hidden,
	})
	ctrl <- float64(download.Cur / download.Total)
	go func() {
		m := styles.ProcessBarModel{
			Progress:        progress.New(progress.WithDefaultGradient()),
			ProgressPercent: <-ctrl,
		}

		if _, err := tea.NewProgram(m).Run(); err != nil {
			console.Log.Errorf("console has an error: %s", err)
			os.Exit(1)
		}
	}()
	if err != nil {
		console.Log.Errorf("")
	}

}
