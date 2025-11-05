package taskschd

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// TaskSchdRunCmd runs a scheduled task immediately by name.
func TaskSchdRunCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	task, err := TaskSchdRun(con.Rpc, session, name, taskFolder)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdRun(rpc clientrpc.MaliceRPCClient, session *client.Session, name, taskFolder string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdRun,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
			Path: taskFolder,
		},
	}
	return rpc.TaskSchdRun(session.Context(), request)
}

func RegisterTaskSchdRunFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdRun,
		TaskSchdRun,
		"",
		nil,
		output.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdRun,
		consts.ModuleTaskSchdRun,
		consts.ModuleTaskSchdRun+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
			"task_folder: task folder",
		},
		[]string{"task"})
}
