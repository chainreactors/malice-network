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

// TaskSchdStartCmd starts a scheduled task by name.
func TaskSchdStartCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := TaskSchdStart(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("start scheduled task: %s", name))
	return nil
}

func TaskSchdStart(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdStart,
		Taskschd: &implantpb.TaskSchedule{
			Name: name,
		},
	}
	return rpc.TaskSchdStart(session.Context(), request)
}

func RegisterTaskSchdStartFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdStart,
		TaskSchdStart,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
}
