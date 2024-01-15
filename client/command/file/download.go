package file

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{{
		Name: "download",
		Help: "download file",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "filename")
			f.String("p", "path", "", "filepath")
		},
		Run: func(ctx *grumble.Context) error {
			Download(ctx, con)
			return nil
		},
	}}
}

func Download(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	name := ctx.Flags.String("name")
	path := ctx.Flags.String("path")

	var download *clientpb.Task
	var err error
	ctrl := make(chan float64)
	download, err = con.Rpc.Download(con.ActiveTarget.Context(), &pluginpb.DownloadRequest{
		Name: name,
		Path: path,
	})
	ctrl <- float64(download.Cur / download.Total)
	go func() {
		m := styles.ProcessBarModel{
			Progress:        progress.New(progress.WithDefaultGradient()),
			ProgressPercent: <-ctrl,
		}

		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Println("Oh no!", err)
			os.Exit(1)
		}
	}()
	if err != nil {
		console.Log.Errorf("")
	}

}
