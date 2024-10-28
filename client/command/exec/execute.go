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

func ExecuteCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	quiet, _ := cmd.Flags().GetBool("quiet")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Execute(con.Rpc, session, cmdStr, !quiet)
	if err != nil {
		return err
	}
	con.GetInteractive().Console(task, "exec: "+cmdStr)
	return nil
}

func Execute(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, output bool) (*clientpb.Task, error) {
	cmdStrList, err := shellquote.Split(cmd)
	if err != nil {
		return nil, err
	}
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:   cmdStrList[0],
		Args:   cmdStrList[1:],
		Output: output,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterExecuteFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleExecution,
		Execute,
		"",
		nil,
		common.ParseExecResponse,
		nil,
	)
}
