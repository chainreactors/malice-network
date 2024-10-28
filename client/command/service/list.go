package service

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

func ServiceListCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := ServiceList(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, "service list")
	return nil
}

func ServiceList(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	return rpc.ServiceList(session.Context(), &implantpb.Request{
		Name: consts.ModuleServiceList,
	})
}

func RegisterServiceListFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceList,
		ServiceList,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			return fmt.Sprintf("%v", content.Spite.GetBody()), nil
		},
		nil,
	)
}
