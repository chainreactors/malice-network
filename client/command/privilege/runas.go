package privilege

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

// RunasCmd executes a program under another user's credentials.
func RunasCmd(cmd *cobra.Command, con *repl.Console) error {
	username, _ := cmd.Flags().GetString("username")
	domain, _ := cmd.Flags().GetString("domain")
	password, _ := cmd.Flags().GetString("password")
	program, _ := cmd.Flags().GetString("program")
	args, _ := cmd.Flags().GetString("args")
	show, _ := cmd.Flags().GetInt32("show")
	netonly, _ := cmd.Flags().GetBool("netonly")

	session := con.GetInteractive()
	task, err := Runas(con.Rpc, session, username, domain, password, program, args, show, netonly)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("runas user: %s on domain: %s", username, domain))
	return nil
}

func Runas(rpc clientrpc.MaliceRPCClient, session *core.Session, username, domain, password, program, args string, show int32, netonly bool) (*clientpb.Task, error) {
	request := &implantpb.RunAsRequest{
		Username: username,
		Domain:   domain,
		Password: password,
		Program:  program,
		Args:     args,
		Show:     show,
		Netonly:  netonly,
	}
	return rpc.Runas(session.Context(), request)
}

func RegisterRunasFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRunas,
		Runas,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
	//session *core.Session, username, domain, password, program, args string, show int32, netonly bool
	// sys runas --username admin --domain EXAMPLE --password admin123 --program /path/to/program --args "arg1 arg2"
	con.AddCommandFuncHelper(
		consts.ModuleRunas,
		consts.ModuleRunas,
		consts.ModuleRunas+`(active(),"admin","EXAMPLE","password123","/path/to/program","arg1 arg2",0,false)`,
		[]string{
			"session: special session",
			"username",
			"domain",
			"password",
			"program",
			"args",
			"show",
			"netonly",
		},
		[]string{"task"})
}
