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

// ServiceQueryCmd queries the status of an existing service by its name.
func ServiceQueryCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	session := con.GetInteractive()
	task, err := ServiceQuery(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("query service: %s", name))
	return nil
}

func ServiceQuery(rpc clientrpc.MaliceRPCClient, session *core.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.ServiceRequest{
		Type: consts.ModuleServiceQuery,
		Service: &implantpb.ServiceConfig{
			Name: name,
		},
	}
	return rpc.ServiceQuery(session.Context(), request)
}

func RegisterServiceQueryFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceQuery,
		ServiceQuery,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			service := content.Spite.GetServiceResponse()
			config := service.GetConfig()
			status := service.GetStatus()

			return fmt.Sprintf(
				"Service Information:\n"+
					"  Name:            %s\n"+
					"  Display Name:    %s\n"+
					"  Executable Path: %s\n"+
					"  Start Type:      %d\n"+
					"  Error Control:   %d\n"+
					"  Account Name:    %s\n\n"+
					"Service Status:\n"+
					"  Current State:   %d\n"+
					"  Process ID:      %d\n"+
					"  Exit Code:       %d\n"+
					"  Checkpoint:      %d\n"+
					"  Wait Hint:       %d ms",
				config.GetName(),
				config.GetDisplayName(),
				config.GetExecutablePath(),
				config.GetStartType(),
				config.GetErrorControl(),
				config.GetAccountName(),
				status.GetCurrentState(),
				status.GetProcessId(),
				status.GetExitCode(),
				status.GetCheckpoint(),
				status.GetWaitHint(),
			), nil
		},
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleServiceQuery,
		consts.ModuleServiceQuery,
		consts.ModuleServiceQuery+`(active(),"service_name")`,
		[]string{
			"session: special session",
			"name: service name",
		},
		[]string{"task"})
}
