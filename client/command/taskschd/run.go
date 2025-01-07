package taskschd

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

// TaskSchdRunCmd runs a scheduled task immediately by name.
func TaskSchdRunCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := TaskSchdRun(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("run scheduled task: %s", name))
	return nil
}

func TaskSchdRun(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdRun,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
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
		common.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdRun,
		consts.ModuleTaskSchdRun,
		consts.ModuleTaskSchdRun+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
		},
		[]string{"task"})
}
