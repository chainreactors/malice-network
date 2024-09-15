package exec

import (
	"bytes"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func ExecutePowershellCmd(cmd *cobra.Command, con *repl.Console) {
	script, _ := cmd.Flags().GetString("script")
	cmdline := cmd.Flags().Args()
	session := con.GetInteractive()
	task, err := ExecPowershell(con.Rpc, session, script, cmdline)
	if err != nil {
		con.Log.Errorf("Execute Powershell error: %v", err)
		return
	}
	con.AddCallback(task, func(msg *implantpb.Spite) (string, error) {
		resp, _ := builtin.ParseAssembly(msg)
		return resp, nil
	})
}

func ExecPowershell(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, ps []string) (*clientpb.Task, error) {
	var psBin bytes.Buffer
	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		psBin.Write(content)
		psBin.WriteString("\n")
	}

	psBin.WriteString(strings.Join(ps, " "))

	task, err := rpc.ExecutePowershell(sess.Context(), &implantpb.ExecuteBinary{
		Name:   "ps",
		Bin:    psBin.Bytes(),
		Type:   consts.ModulePowershell,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
