package exec

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/spf13/cobra"
	"strings"
)

// ExecuteLocalCmd - Execute local PE on sacrifice process
func ExecuteLocalCmd(cmd *cobra.Command, con *repl.Console) {
	args := cmd.Flags().Args()
	process, _ := cmd.Flags().GetString("process")
	output, _ := cmd.Flags().GetBool("output")
	sac, _ := common.ParseSacrificeFlags(cmd)
	task, err := ExecLocal(con.Rpc, con.GetInteractive(), args, output, process, sac)
	if err != nil {
		con.Log.Errorf("Execute EXE error: %v", err)
		return
	}
	con.GetInteractive().Console(task, strings.Join(args, " "))
}

func ExecLocal(rpc clientrpc.MaliceRPCClient, sess *core.Session,
	args []string, output bool, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	args[0] = file.FormatWindowPath(args[0])
	if process == "" {
		process = args[0]
	}

	binary := &implantpb.ExecuteBinary{
		ProcessName: process,
		Bin:         []byte(args[0]),
		Args:        args[1:],
		Output:      output,
		Sacrifice:   sac,
	}

	task, err := rpc.ExecuteLocal(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, nil
}
