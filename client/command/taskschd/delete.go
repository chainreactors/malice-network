package taskschd

import (
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// TaskSchdDeleteCmd deletes a scheduled task by name.
func TaskSchdDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	task, err := TaskSchdDelete(con.Rpc, session, name, taskFolder)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdDelete(rpc clientrpc.MaliceRPCClient, session *session.Session, name, taskFolder string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdDelete,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
			Path: taskFolder,
		},
	}
	return rpc.TaskSchdDelete(session.Context(), request)
}

func RegisterTaskSchdDeleteFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdDelete,
		TaskSchdDelete,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdDelete,
		consts.ModuleTaskSchdDelete,
		//session *core.Session, namespace string, args []string
		consts.ModuleTaskSchdDelete+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
			"task_folder: task folder",
		},
		[]string{"task"})
}
