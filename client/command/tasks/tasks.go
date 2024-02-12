package tasks

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/charmbracelet/bubbles/table"
	"strconv"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "tasks",
			Help: "List tasks",
			Flags: func(f *grumble.Flags) {
				f.String("k", "kill", "", "kill the designated task")
				f.Bool("K", "kill-all", false, "kill all the tasks")
				f.Bool("C", "clean", false, "clean out any tasks marked as error")
				//f.String("f", "filter", "", "filter sessions by substring")
				//f.String("e", "filter-re", "", "filter sessions by regular expression")

				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				TasksCmd(ctx, con)
				return nil
			},
		},
	}
}

func TasksCmd(ctx *grumble.Context, con *console.Console) {
	con.UpdateTasks(con.ActiveTarget.GetInteractive())
	sid := con.ActiveTarget.GetInteractive().SessionId
	if 0 < len(con.Sessions[sid].Tasks) {
		PrintTasks(con.Sessions[sid].Tasks, con)
	} else {
		console.Log.Info("No sessions")
	}
}

func PrintTasks(tasks []*clientpb.Task, con *console.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "ID", Width: 4},
		{Title: "Type", Width: 4},
		{Title: "Status", Width: 8},
		{Title: "Process", Width: 10},
	})
	for _, task := range tasks {
		var processValue string
		var status string
		if task.Status == 0 {
			status = "Run"
			processValue = fmt.Sprintf("%.2f%%", float64(task.Cur)/float64(task.Total)*100)
		} else if task.Cur/task.Total == 1 {
			status = "Complete"
			processValue = "100%"
		} else {
			status = "Error"
			processValue = "0%"
		}
		row = table.Row{
			strconv.Itoa(int(task.TaskId)),
			task.Type,
			status,
			processValue,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	tui.Run(tableModel)
}
