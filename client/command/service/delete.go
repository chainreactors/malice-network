package service

import (
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/session"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// ServiceDeleteCmd deletes a specified service by name.
func ServiceDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)

	session := con.GetInteractive()
	task, err := ServiceDelete(con.Rpc, session, name)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func ServiceDelete(rpc clientrpc.MaliceRPCClient, session *session.Session, name string) (*clientpb.Task, error) {
	request := &implantpb.ServiceRequest{
		Type: consts.ModuleServiceDelete,
		Service: &implantpb.ServiceConfig{
			Name: name,
		},
	}
	return rpc.ServiceDelete(session.Context(), request)
}

func RegisterServiceDeleteFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleServiceDelete,
		ServiceDelete,
		"",
		nil,
		output.ParseStatus,
		nil,
	)
	con.AddCommandFuncHelper(
		consts.ModuleServiceDelete,
		consts.ModuleServiceDelete,
		consts.ModuleServiceDelete+`(active(),"service_name")`,
		[]string{
			"session: special session",
			"name: service name",
		},
		[]string{"task"})
}
