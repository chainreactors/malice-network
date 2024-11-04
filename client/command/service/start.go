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

// ServiceStartCmd starts an existing service by its name.
func ServiceStartCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := ServiceStart(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("start service: %s", name))
	return nil
}

// ServiceStart 通过 gRPC 调用启动服务
func ServiceStart(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.ServiceRequest{
		Type: consts.ModuleServiceCreate,
		Service: &implantpb.ServiceConfig{
			Name: name,
		},
	}

	return rpc.ServiceStart(session.Context(), request)
}

// RegisterServiceStartFunc 注册 ServiceStartCmd 到 Console
func RegisterServiceStartFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceStart,
		ServiceStart,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
	con.AddInternalFuncHelper(
		consts.ModuleServiceStart,
		consts.ModuleServiceStart,
		consts.ModuleServiceStart+`(active(),"service_name")`,
		[]string{
			"session: special session",
			"name: service name",
		},
		[]string{"task"})
}
