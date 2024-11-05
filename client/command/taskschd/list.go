package taskschd

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"strings"
)

// TaskSchdListCmd lists all scheduled tasks.
func TaskSchdListCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := TaskSchdList(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "list all scheduled tasks")
	return nil
}

func TaskSchdList(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	request := &implantpb.Request{
		Name: consts.ModuleTaskSchdList,
	}
	return rpc.TaskSchdList(session.Context(), request)
}

func RegisterTaskSchdListFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdList,
		TaskSchdList,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			return fmt.Sprintf("Scheduled Tasks: %v", content.Spite.GetBody()), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			taskScheduleResponse := content.Spite.GetSchedulesResponse()
			if len(taskScheduleResponse.Schedules) == 0 {
				return "", errors.New("no scheduled tasks found")
			}
			var result []string
			for _, schedule := range taskScheduleResponse.Schedules {
				result = append(result, tui.RenderColoredKeyValue(schedule, 5, 0))
			}
			return strings.Join(result, "\n"), nil
		},
		//	taskScheduleResponse := content.Spite.GetSchedulesResponse()
		//	if len(taskScheduleResponse.Schedules) == 0 {
		//		return "", errors.New("no scheduled tasks found")
		//	}
		//
		//	tableModel := tui.NewTable([]table.Column{
		//		{Title: "Name", Width: 20},
		//		{Title: "Path & Executable Path", Width: 60},
		//		{Title: "Trigger Type", Width: 13},
		//		{Title: "Start Boundary", Width: 20},
		//		{Title: "Description", Width: 50},
		//		{Title: "Last RunTime", Width: 20},
		//		{Title: "Next RunTime", Width: 20},
		//	}, true)
		//
		//	var rowEntries []table.Row
		//	for _, schedule := range taskScheduleResponse.Schedules {
		//		row := table.Row{
		//			schedule.Name,
		//			schedule.Path + "\n" + schedule.ExecutablePath,
		//			fmt.Sprintf("%d", schedule.TriggerType),
		//			schedule.StartBoundary,
		//			schedule.Description,
		//			schedule.LastRunTime,
		//			schedule.NextRunTime,
		//		}
		//		rowEntries = append(rowEntries, row)
		//	}
		//
		//	tableModel.SetRows(rowEntries)
		//	return tableModel.View(), nil
		//},
	)

	con.AddInternalFuncHelper(
		consts.ModuleTaskSchdList,
		consts.ModuleTaskSchdList,
		//session *core.Session, namespace string, args []string
		consts.ModuleTaskSchdList+"(active())",
		[]string{
			"sess: special session",
		},
		[]string{"task"})
}
