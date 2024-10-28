package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func ShellCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	quiet, _ := cmd.Flags().GetBool("quiet")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Shell(con.Rpc, session, cmdStr, !quiet)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "exec: "+cmdStr)
	return nil
}

func Shell(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, output bool) (*clientpb.Task, error) {
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:   `C:\Windows\System32\cmd.exe`,
		Args:   []string{"/c", cmd},
		Output: output,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterShellFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleAliasShell,
		Shell,
		"bshell",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string) (*clientpb.Task, error) {
			return Shell(rpc, sess, cmd, true)
		},
		common.ParseExecResponse,
		nil,
	)
}
