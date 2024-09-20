package exec

import (
	"bytes"
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
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
	task, err := PowerPick(con.Rpc, session, script, cmdline)
	if err != nil {
		con.Log.Errorf("Execute Powershell error: %v", err)
		return
	}
	con.GetInteractive().Console(task, fmt.Sprintf("%s, args: %v", script, cmdline))
}

func PowerPick(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string, ps []string) (*clientpb.Task, error) {
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
		Name:   "",
		Bin:    psBin.Bytes(),
		Type:   consts.ModulePowerpick,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
