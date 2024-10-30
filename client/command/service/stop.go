package service

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

// ServiceStopCmd stops an existing service by its name.
func ServiceStopCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := ServiceStop(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("stop service: %s", name))
	return nil
}

func ServiceStop(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.ServiceRequest{
		Type: consts.ModuleServiceStop,
		Service: &implantpb.ServiceConfig{
			Name: name,
		},
	}
	return rpc.ServiceStop(session.Context(), request)
}

func RegisterServiceStopFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceStop,
		ServiceStop,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
	con.AddInternalFuncHelper(
		consts.ModuleServiceStop,
		consts.ModuleServiceStop,
		consts.ModuleServiceStop+"(active(),"+"\"service_name\""+")",
		[]string{
			"session: special session",
			"name: service name",
		},
		[]string{"task"})
}
