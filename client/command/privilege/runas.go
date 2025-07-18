package privilege

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

// RunasCmd executes a program under another user's credentials.
func RunasCmd(cmd *cobra.Command, con *repl.Console) error {
	username, _ := cmd.Flags().GetString("username")
	domain, _ := cmd.Flags().GetString("domain")
	password, _ := cmd.Flags().GetString("password")
	program, _ := cmd.Flags().GetString("path")
	args, _ := cmd.Flags().GetString("args")
	useProfile, _ := cmd.Flags().GetBool("use-profile")
	useEnv, _ := cmd.Flags().GetBool("use-env")
	netonly, _ := cmd.Flags().GetBool("netonly")

	session := con.GetInteractive()
	task, err := Runas(con.Rpc, session, username, domain, password, program, args, useProfile, useEnv, netonly)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("runas user: %s on domain: %s", username, domain))
	return nil
}

func Runas(rpc clientrpc.MaliceRPCClient, session *core.Session, username, domain, password, program, args string, useProfile, useEnv, netonly bool) (*clientpb.Task, error) {
	request := &implantpb.RunAsRequest{
		Username:   username,
		Domain:     domain,
		Password:   password,
		Program:    program,
		Args:       args,
		UseEnv:     useEnv,
		UseProfile: useProfile,
		Netonly:    netonly,
	}
	return rpc.Runas(session.Context(), request)
}

func RegisterRunasFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRunas,
		Runas,
		"",
		nil,
		output.ParseExecResponse,
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
			"use profile",
			"use env",
			"netonly",
		},
		[]string{"task"})
}
