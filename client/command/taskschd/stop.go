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

// TaskSchdStopCmd stops a scheduled task by name.
func TaskSchdStopCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := TaskSchdStop(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("stop scheduled task: %s", name))
	return nil
}

func TaskSchdStop(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdStop,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
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
		common.ParseStatus,
		nil,
	)
	con.AddInternalFuncHelper(
		consts.ModuleTaskSchdStop,
		consts.ModuleTaskSchdStop,
		consts.ModuleTaskSchdStop+`(active(), "task_name")`,
		[]string{
			"session: special session",
			"name: name of the scheduled task",
		},
		[]string{"task"})
}
