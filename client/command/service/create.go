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

// ServiceCreateCmd creates a new service with the specified configuration.
func ServiceCreateCmd(cmd *cobra.Command, con *repl.Console) error {
	name, _ := cmd.Flags().GetString("name")
	displayName, _ := cmd.Flags().GetString("display")
	executablePath, _ := cmd.Flags().GetString("path")
	startType, _ := cmd.Flags().GetUint32("start_type")
	errorControl, _ := cmd.Flags().GetUint32("error")
	accountName, _ := cmd.Flags().GetString("account")

	session := con.GetInteractive()
	task, err := ServiceCreate(con.Rpc, session, name, displayName, executablePath, startType, errorControl, accountName)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("create service: %s %s", name, executablePath))
	return nil
}

func ServiceCreate(rpc clientrpc.MaliceRPCClient, session *core.Session, name, displayName, executablePath string, startType, errorControl uint32, accountName string) (*clientpb.Task, error) {
	request := &implantpb.ServiceRequest{
		Type: consts.ModuleServiceCreate,
		Service: &implantpb.ServiceConfig{
			Name:           name,
			DisplayName:    displayName,
			ExecutablePath: executablePath,
			StartType:      startType,
			ErrorControl:   errorControl,
			AccountName:    accountName,
		},
	}

	// 执行创建服务的 gRPC 请求
	return rpc.ServiceCreate(session.Context(), request)
}

// RegisterServiceCreateFunc 注册 ServiceCreateCmd 到 Console
func RegisterServiceCreateFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceCreate,
		ServiceCreate,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
	con.AddInternalFuncHelper(
		consts.ModuleServiceCreate,
		consts.ModuleServiceCreate,
		consts.ModuleServiceCreate+`(active(), "service_name", "display", "path", 0, 0, "account")`,
		[]string{
			"session: special session",
			"name: service name",
			"displayName: display name",
			"executablePath: executable path",
			"startType: start type",
			"errorControl: error control",
			"accountName: account name",
		},
		[]string{"task"})
}
