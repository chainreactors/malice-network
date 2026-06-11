package tasks

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListFiles(cmd *cobra.Command, con *core.Console) error {
	//resp, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(),
	//	&clientpb.Session{SessionId: con.GetInteractive().SessionId})
	resp, err := con.Rpc.GetFiles(
		con.ActiveTarget.Context(),
		&clientpb.Session{
			SessionId: con.GetInteractive().SessionId,
		})
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

func printFiles(files *clientpb.Files, con *core.Console) {
	var rowEntries []table.Row
	var row table.Row
	for _, file := range files.Files {
		row = table.NewRow(
			table.RowData{
				"FileID":     file.TaskId,
				"Name":       file.Name,
				"Type":       file.Op,
				"LocalName":  file.Local,
				"RemotePath": file.Remote,
				//"Checksum":   file.Checksum,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("FileID", "File ID", 8),
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Type", "Type", 10),
		table.NewFlexColumn("LocalName", "Local Name", 1),
		table.NewFlexColumn("RemotePath", "Remote Path", 2),
		//table.NewColumn("Checksum", "Checksum", maxLengths["Checksum"]),
	}, true)
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}

func updateMaxLength(maxLengths *map[string]int, key string, newLength int) {
	if (*maxLengths)[key] < newLength {
		(*maxLengths)[key] = newLength
	}
}
