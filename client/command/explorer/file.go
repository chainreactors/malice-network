package explorer

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
)

func explorerCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	dirEntriesChan := make(chan []os.DirEntry, 1)
	var path = ""

	task, err := con.Rpc.Ls(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: "./",
	})
	if err != nil {
		repl.Log.Errorf("load directory error: %v", err)
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		var dirEntries []os.DirEntry
		for _, protoFile := range resp.GetFiles() {
			dirEntries = append(dirEntries, ProtobufDirEntry{FileInfo: protoFile})
		}
		path = resp.GetPath()
		path = path[4:]
		dirEntriesChan <- dirEntries
	})

	var dirEntries []os.DirEntry
	explorer := NewExplorer(dirEntries, con)
	explorer.FilePicker.CurrentDirectory = "./"
	explorer.FilePicker.Height = 50
	for {
		select {
		case newEntries := <-dirEntriesChan:
			if len(newEntries) > 0 {
				dirEntries = newEntries
				err := SetFiles(&explorer.FilePicker, dirEntries)
				if err != nil {
					repl.Log.Errorf("Error setting files: %v", err)
					return
				}
				explorer.Files = dirEntries
				explorer.FilePicker.CurrentDirectory = path
				explorer.max = max(explorer.max, explorer.FilePicker.Height-1)
				if _, err := tea.NewProgram(explorer, tea.WithAltScreen()).Run(); err != nil {
					repl.Log.Errorf("Error running explorer: %v", err)
				}
				//newExplorer := tui.NewModel(explorer, nil, false, false)
				//err = newExplorer.Run()
				//if err != nil {
				//	con.SessionLog(sid).Errorf("Error running explorer: %v", err)
				//}
				return
			}

		}
	}
}
