package tasks

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
)

func Command(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks",
		Long:  help.GetHelpFor("tasks"),
		Run: func(cmd *cobra.Command, args []string) {
			listTasks(cmd, con)
			return
		},
	}
	return []*cobra.Command{
		taskCmd,
	}
}

func listTasks(cmd *cobra.Command, con *repl.Console) {
	err := con.UpdateTasks(con.GetInteractive())
	if err != nil {
		repl.Log.Errorf("Error updating tasks: %v", err)
		return
	}
	tasks := con.GetInteractive().Tasks.GetTasks()
	if 0 < len(tasks) {
		printTasks(tasks, con)
	} else {
		repl.Log.Info("No sessions")
	}

}

func printTasks(tasks []*clientpb.Task, con *repl.Console) {
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
			status = "Complete"
		} else {
			status = "Run"
		}
		row = table.Row{
			strconv.Itoa(int(task.TaskId)),
			task.Type,
			status,
			string(task.Cur),
			string(task.Total),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	//newTable := tui.NewModel(tableModel, nil, false, false)
	//err := newTable.Run()
	//if err != nil {
	//	con.SessionLog(sid).Errorf("Error running table: %v", err)
	//}

}

//
//func TasksCmd(ctx *grumble.Context, con *console.Console) {
//	err := con.UpdateTasks(con.GetInteractive())
//	if err != nil {
//		console.Log.Errorf("Error updating tasks: %v", err)
//		return
//	}
//	sid := con.GetInteractive().SessionId
//	Tasks, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(), con.GetInteractive())
//	if err != nil {
//		con.SessionLog(sid).Errorf("Error getting tasks: %v", err)
//	}
//	if 0 < len(Tasks.Tasks) {
//		PrintTasks(Tasks.Tasks, con)
//	} else {
//		console.Log.Info("No sessions")
//	}
//}
//
//type description struct {
//	Name string `json:"name"`
//	Path string `json:"path"`
//}
//
//func PrintTasks(tasks []*clientpb.TaskDesc, con *console.Console) {
//	sid := con.GetInteractive().SessionId
//	var rowEntries []table.Row
//	var row table.Row
//	tableModel := tui.NewTable([]table.Column{
//		{Title: "ID", Width: 4},
//		{Title: "Type", Width: 10},
//		{Title: "Status", Width: 8},
//		{Title: "Process", Width: 10},
//		{Title: "FileName", Width: 15},
//		{Title: "FilePath", Width: 40},
//	}, true)
//	for _, task := range tasks {
//		var desc description
//		var processValue string
//		var status string
//		if task.Status != 0 {
//			status = "Error"
//			processValue = "0%"
//		} else if task.Cur/task.Total == 1 {
//			status = "Complete"
//			processValue = "100%"
//		} else {
//			status = "Run"
//			processValue = fmt.Sprintf("%.2f%%", float64(task.Cur)/float64(task.Total)*100)
//		}
//		err := json.Unmarshal([]byte(task.Description), &desc)
//		if err != nil {
//			con.SessionLog(sid).Errorf("Error parsing JSON:", err)
//			return
//		}
//		row = table.Row{
//			strconv.Itoa(int(task.TaskId)),
//			task.Type,
//			status,
//			processValue,
//			desc.Name,
//			desc.Path,
//		}
//		rowEntries = append(rowEntries, row)
//	}
//	tableModel.SetRows(rowEntries)
//	newTable := tui.NewModel(tableModel, nil, false, false)
//	err := newTable.Run()
//	if err != nil {
//		con.SessionLog(sid).Errorf("Error running table: %v", err)
//	}
//}
