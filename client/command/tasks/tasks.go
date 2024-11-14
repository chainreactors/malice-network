package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"strconv"
)

func ListTasks(cmd *cobra.Command, con *repl.Console) error {
	err := con.UpdateTasks(con.GetInteractive())
	if err != nil {
		return err
	}
	isAll, _ := cmd.Flags().GetBool("all")
	tasks := con.GetInteractive().Tasks.GetTasks()
	if 0 < len(tasks) {
		printTasks(tasks, con, isAll)
	} else {
		con.Log.Info("No tasks\n")
	}
	return nil
}

func printTasks(tasks []*clientpb.Task, con *repl.Console, isAll bool) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 4),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Status", "Status", 8),
		table.NewColumn("cur", "cur", 5),
		table.NewColumn("total", "total", 5),
		table.NewColumn("callby", "callby", 10),
		table.NewColumn("timeout", "timeout", 8),
	}, true)
	for _, task := range tasks {
		var status string
		if task.Status != 0 {
			status = "Error"
		} else if task.Cur != task.Total {
			if !isAll {
				continue
			}
			status = "Complete"
		} else {
			status = "Running"
		}
		row = table.NewRow(
			table.RowData{
				"ID":      strconv.Itoa(int(task.TaskId)),
				"Type":    task.Type,
				"Status":  status,
				"cur":     strconv.Itoa(int(task.Cur)),
				"total":   strconv.Itoa(int(task.Total)),
				"callby":  task.Callby,
				"timeout": strconv.FormatBool(task.Timeout),
			})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetAscSort("ID")
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())

}

//	func TasksCmd(ctx *grumble.Context, con *console.Console) {
//		err := con.UpdateTasks(con.GetInteractive())
//		if err != nil {
//			console.Log.Errorf("Error updating tasks: %v", err)
//			return
//		}
//		sid := con.GetInteractive().SessionId
//		Tasks, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(), con.GetInteractive())
//		if err != nil {
//			con.SessionLog(sid).Errorf("Error getting tasks: %v", err)
//		}
//		if 0 < len(Tasks.Tasks) {
//			PrintTasks(Tasks.Tasks, con)
//		} else {
//			console.Log.Info("No sessions")
//		}
//	}
