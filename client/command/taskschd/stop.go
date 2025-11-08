package taskschd

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// TaskSchdStopCmd stops a scheduled task by name.
func TaskSchdStopCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	task, err := TaskSchdStop(con.Rpc, session, name, taskFolder)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdStop(rpc clientrpc.MaliceRPCClient, session *client.Session, name, taskFolder string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdStop,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
			Path: taskFolder,
		},
	}
	return rpc.TaskSchdStop(session.Context(), request)
}

func RegisterTaskSchdStopFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdStop,
		TaskSchdStop,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdStop,
		consts.ModuleTaskSchdStop,
		consts.ModuleTaskSchdStop+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
			"task_folder: task folder",
		},
		[]string{"task"})
}
