package taskschd

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
)

// TaskSchdQueryCmd queries the detailed configuration of a scheduled task by name.
func TaskSchdQueryCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	task, err := TaskSchdQuery(con.Rpc, session, name, taskFolder)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdQuery(rpc clientrpc.MaliceRPCClient, session *client.Session, name, taskFolder string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdQuery,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
			Path: taskFolder,
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
