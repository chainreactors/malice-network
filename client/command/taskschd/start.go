package taskschd

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// TaskSchdStartCmd starts a scheduled task by name.
func TaskSchdStartCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	task, err := TaskSchdStart(con.Rpc, session, name, taskFolder)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdStart(rpc clientrpc.MaliceRPCClient, session *client.Session, name, taskFolder string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdStart,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
			Path: taskFolder,
		},
	}
	return rpc.TaskSchdStart(session.Context(), request)
}

func RegisterTaskSchdStartFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdStart,
		TaskSchdStart,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdStart,
		consts.ModuleTaskSchdStart,
		//session *core.Session, namespace string, args []string
		consts.ModuleTaskSchdCreate+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
			"task_folder: task folder",
		},
		[]string{"task"})
}
