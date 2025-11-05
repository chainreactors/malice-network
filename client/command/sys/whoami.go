package sys

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func WhoamiCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Whoami(con.Rpc, session)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Whoami(rpc clientrpc.MaliceRPCClient, session *client.Session) (*clientpb.Task, error) {
	task, err := rpc.Whoami(session.Context(), &implantpb.Request{
		Name: consts.ModuleWhoami,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterWhoamiFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleWhoami,
		Whoami,
		"bwhoami",
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
			return Whoami(rpc, sess)
		},
		output.ParseResponse,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleWhoami,
		consts.ModuleWhoami,
		"whoami(active())",
		[]string{
			"sess: special session",
		},
		[]string{"task"})
}
