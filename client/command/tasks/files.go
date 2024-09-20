package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
)

func listFiles(cmd *cobra.Command, con *repl.Console) {
	resp, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(),
		&clientpb.Session{SessionId: con.GetInteractive().SessionId})
	if err != nil {
		con.Log.Errorf("Error getting tasks: %v", err)
	}
	if 0 < len(resp.Files) {
		printFiles(resp, con)
	} else {
		con.Log.Info("No files")
	}

}

func printFiles(files *clientpb.Files, con *repl.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "FileID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "TempID", Width: 10},
		{Title: "Type", Width: 10},
		{Title: "LocalName", Width: 30},
		{Title: "RemotePath", Width: 30},
	}, true)
	for _, file := range files.Files {
		row = table.Row{
			file.TaskId,
			file.Name,
			file.TempId,
			file.Op,
			file.Local,
			file.Remote,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	//newTable := tui.NewModel(tableModel, nil, false, false)
	//err := newTable.Run()
	//if err != nil {
	//	con.Log.Errorf("Error running table: %v", err)
	//}
}
