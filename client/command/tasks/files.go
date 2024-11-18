package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListFiles(cmd *cobra.Command, con *repl.Console) error {
	resp, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(),
		&clientpb.Session{SessionId: con.GetInteractive().SessionId})
	if err != nil {
		return err
	}
	if 0 < len(resp.Files) {
		printFiles(resp, con)
	} else {
		con.Log.Info("No files\n")
	}

	return nil
}

func printFiles(files *clientpb.Files, con *repl.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("FileID", "FileID", 8),
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("TempID", "TempID", 10),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("LocalName", "LocalName", 30),
		table.NewColumn("RemotePath", "RemotePath", 30),
	}, true)
	for _, file := range files.Files {
		row = table.NewRow(
			table.RowData{
				"FileID":     file.TaskId,
				"Name":       file.Name,
				"TempID":     file.TempId,
				"Type":       file.Op,
				"LocalName":  file.Local,
				"RemotePath": file.Remote,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
