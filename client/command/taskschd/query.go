package taskschd

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

// TaskSchdQueryCmd queries the detailed configuration of a scheduled task by name.
func TaskSchdQueryCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := TaskSchdQuery(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("query scheduled task: %s", name))
	return nil
}

func TaskSchdQuery(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdQuery,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
		},
	}
	return rpc.TaskSchdQuery(session.Context(), request)
}

func RegisterTaskSchdQueryFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdQuery,
		TaskSchdQuery,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			task := content.Spite.GetScheduleResponse()
			return fmt.Sprintf("Task Name: %s\nPath: %s\nExecutable Path: %s\nTrigger Type: %d\nStart Boundary: %s\nDescription: %s\nEnabled: %v\nLast Run Time: %s\nNext Run Time: %s",
				task.Name, task.Path, task.ExecutablePath, task.TriggerType, task.StartBoundary, task.Description, task.Enabled, task.LastRunTime, task.NextRunTime), nil
		},
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdQuery,
		consts.ModuleTaskSchdQuery,
		consts.ModuleTaskSchdQuery+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
		},
		[]string{"task"})
}
