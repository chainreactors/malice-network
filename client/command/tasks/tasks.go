package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"sort"
	"strconv"
)

func listTasks(cmd *cobra.Command, con *repl.Console) {
	err := con.UpdateTasks(con.GetInteractive())
	if err != nil {
		con.Log.Errorf("Error updating tasks: %v", err)
		return
	}
	isAll, _ := cmd.Flags().GetBool("all")
	tasks := con.GetInteractive().Tasks.GetTasks()
	if 0 < len(tasks) {
		printTasks(tasks, con, isAll)
	} else {
		con.Log.Info("No tasks")
	}
}

func printTasks(tasks []*clientpb.Task, con *repl.Console, isAll bool) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "ID", Width: 4},
		{Title: "Type", Width: 10},
		{Title: "Status", Width: 8},
		{Title: "cur", Width: 5},
		{Title: "total", Width: 5},
	}, true)
	for _, task := range tasks {
		var status string
		if task.Status != 0 {
			status = "Error"
		} else if task.Cur/task.Total == 1 {
			if !isAll {
				continue
			}
			status = "Complete"
		} else {
			status = "Run"
		}
		row = table.Row{
			strconv.Itoa(int(task.TaskId)),
			task.Type,
			status,
			strconv.Itoa(int(task.Cur)),
			strconv.Itoa(int(task.Total)),
		}
		rowEntries = append(rowEntries, row)
	}
	sort.Slice(rowEntries, func(i, j int) bool {
		id1, _ := strconv.Atoi(rowEntries[i][0])
		id2, _ := strconv.Atoi(rowEntries[j][0])
		return id1 < id2
	})

	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	//newTable := tui.NewModel(tableModel, nil, false, false)
	//err := newTable.Run()
	//if err != nil {
	//	con.SessionLog(sid).Errorf("Error running table: %v", err)
	//}

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
