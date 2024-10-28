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

// TaskSchdListCmd lists all scheduled tasks.
func TaskSchdListCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := TaskSchdList(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "list all scheduled tasks")
	return nil
}

func TaskSchdList(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	request := &implantpb.Request{
		Name: consts.ModuleTaskSchdList,
	}
	return rpc.TaskSchdList(session.Context(), request)
}

func RegisterTaskSchdListFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdList,
		TaskSchdList,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			return fmt.Sprintf("Scheduled Tasks: %v", content.Spite.GetBody()), nil
		},
		nil,
	)
}
