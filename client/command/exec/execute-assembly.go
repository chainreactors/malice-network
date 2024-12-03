package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func ExecuteAssemblyCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	path, args, output, _ := common.ParseBinaryFlags(cmd)
	clrparam := common.ParseCLRFlags(cmd)
	task, err := ExecAssembly(con.Rpc, session, path, args, output, clrparam)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, path)
	return nil
}

func ExecAssembly(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args []string, output bool, param map[string]string) (*clientpb.Task, error) {
	binary, err := common.NewExecutable(consts.ModuleExecuteAssembly, path, args, sess.Os.Arch, output, nil)
	if err != nil {
		return nil, err
	}
	binary.Param = param
	task, err := rpc.ExecuteAssembly(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterAssemblyFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecuteAssembly,
		ExecAssembly,
		"bexecute_assembly",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecAssembly(rpc, sess, path, cmdline, true, map[string]string{
				"bypass_amsi": "",
				"bypass_etw":  "",
				"bypass_wldp": "",
			})
		},
		common.ParseAssembly,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleExecuteAssembly,
		consts.ModuleExecuteAssembly,
		consts.ModuleExecuteAssembly+`(active(),"sharp.exe",{}, true, new_bypass_all())`,
		[]string{
			"sessions",
			"path",
			"args",
			"output",
			"param, bypass amsi,wldp,etw",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"bexecute_assembly",
		"bexecute_assembly",
		`bexecute_assembly(active(),"sharp.exe",{})`,
		[]string{
			"sessions",
			"path",
			"args",
		},
		[]string{"task"})

}
