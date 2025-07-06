package tasks

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"strconv"
)

func GetTasksCmd(cmd *cobra.Command, con *repl.Console) error {
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
		table.NewColumn("Type", "Type", 20),
		table.NewColumn("Status", "Status", 15),
		table.NewColumn("cur", "cur", 5),
		table.NewColumn("total", "total", 5),
		table.NewColumn("callby", "callby", 10),
		//table.NewColumn("timeout", "timeout", 8),
	}, true)
	for _, task := range tasks {
		var status string
		if task.Status != 0 {
			status = "Error"
		} else if task.Cur != task.Total {
			status = "Running"
		} else {
			status = "Complete"
		}
		row = table.NewRow(
			table.RowData{
				"ID":     task.TaskId,
				"Type":   task.Type,
				"Status": status,
				"cur":    strconv.Itoa(int(task.Cur)),
				"total":  strconv.Itoa(int(task.Total)),
				"callby": task.Callby,
				//"timeout": strconv.FormatBool(task.Timeout),
			})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetAscSort("ID")
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}

// fetchTaskByIDs 根据逗号分隔的任务ID字符串获取任务详情
func fetchTaskByID(idStr string, con *repl.Console) (*clientpb.TaskContexts, error) {

	taskId, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		con.Log.Errorf("Invalid task ID '%s': %v", idStr, err)
	}
	task := &clientpb.Task{
		SessionId: con.GetInteractive().SessionId,
		TaskId:    uint32(taskId),
		Need:      -1,
	}
	tasksContext, err := con.Rpc.GetAllTaskContent(con.Context(), task)

	return tasksContext, err
}

func TaskFetchCmd(cmd *cobra.Command, con *repl.Console) error {
	// 检查是否使用 --ids 参数
	taskId := cmd.Flags().Arg(0)
	tasksContext, err := fetchTaskByID(taskId, con)
	if err != nil {
		return err
	}
	taskContexts := make([]*clientpb.TaskContext, 0)
	for _, spite := range tasksContext.Spites {
		eachTask := &clientpb.TaskContext{
			Task:    tasksContext.Task,
			Session: tasksContext.Session,
			Spite:   spite,
		}
		taskContexts = append(taskContexts, eachTask)
	}
	for _, context := range taskContexts {
		core.HandlerTask(con.GetInteractive(), context, []byte{}, consts.CalleeCMD, true)
	}
	return nil
}

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
