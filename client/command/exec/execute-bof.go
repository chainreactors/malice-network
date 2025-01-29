package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func ExecuteBofCmd(cmd *cobra.Command, con *repl.Console) error {
	path, args, output, _ := common.ParseBinaryFlags(cmd)
	task, err := ExecBof(con.Rpc, con.GetInteractive(), path, args, output)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, path)
	return nil
}

func ExecBof(rpc clientrpc.MaliceRPCClient, sess *core.Session, bofPath string, args []string, out bool) (*clientpb.Task, error) {
	binary, err := common.NewExecutable(consts.ModuleExecuteBof, bofPath, args, sess.Os.Arch, out, nil)
	if err != nil {
		return nil, err
	}
	task, err := rpc.ExecuteBof(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterBofFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecuteBof,
		ExecBof,
		"binline_execute",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, args string) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecBof(rpc, sess, path, cmdline, true)
		},
		output.ParseBOFResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleExecuteBof,
		consts.ModuleExecuteBof,
		consts.ModuleExecuteBof+`(active(),"/path/dir.x64.o",{"/path/to/list"},true)`,
		[]string{
			"session: special session",
			"bofPath: path to BOF",
			"args: arguments",
			"output: output",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"binline_execute",
		"binline_execute",
		`binline_execute(active(),"/path/dir.x64.o","/path/to/list")`,
		[]string{
			"session: special session",
			"bofPath: path to BOF",
			"args: arguments",
		},
		[]string{"task"})
}
