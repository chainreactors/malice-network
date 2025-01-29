package taskschd

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// TaskSchdDeleteCmd deletes a scheduled task by name.
func TaskSchdDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := TaskSchdDelete(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("delete scheduled task: %s", name))
	return nil
}

func TaskSchdDelete(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdDelete,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
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
		},
		[]string{"task"})
}
