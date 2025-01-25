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
	//resp, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(),
	//	&clientpb.Session{SessionId: con.GetInteractive().SessionId})
	resp, err := con.Rpc.GetContextFiles(
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

func printFiles(files *clientpb.Files, con *repl.Console) {
	var rowEntries []table.Row
	var row table.Row
	maxLengths := map[string]int{
		"FileID":     6,
		"Name":       16,
		"Checksum":   64,
		"Type":       12,
		"LocalName":  16,
		"RemotePath": 16,
	}

	for _, file := range files.Files {
		updateMaxLength(&maxLengths, "FileID", len(file.TaskId))
		updateMaxLength(&maxLengths, "Name", len(file.Name))
		//updateMaxLength(&maxLengths, "Checksum", len(file.TempId[:8]))
		updateMaxLength(&maxLengths, "Type", len(file.Op))
		updateMaxLength(&maxLengths, "LocalName", len(file.Local))
		updateMaxLength(&maxLengths, "RemotePath", len(file.Remote))
		row = table.NewRow(
			table.RowData{
				"FileID":     file.TaskId,
				"Name":       file.Name,
				"Type":       file.Op,
				"LocalName":  file.Local,
				"RemotePath": file.Remote,
				"Checksum":   file.Checksum,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("FileID", "FileID", maxLengths["FileID"]),
		table.NewColumn("Name", "Name", maxLengths["Name"]),
		table.NewColumn("Type", "Type", maxLengths["Type"]),
		table.NewColumn("LocalName", "LocalName", maxLengths["LocalName"]),
		table.NewColumn("RemotePath", "RemotePath", maxLengths["RemotePath"]),
		table.NewColumn("Checksum", "Checksum", maxLengths["Checksum"]),
	}, true)
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}

func updateMaxLength(maxLengths *map[string]int, key string, newLength int) {
	if (*maxLengths)[key] < newLength {
		(*maxLengths)[key] = newLength
	}
}
