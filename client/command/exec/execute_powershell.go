package exec

import (
	"bytes"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"strings"
)

func ExecutePowershellCmd(cmd *cobra.Command, con *console.Console) {
	psList := cmd.Flags().Args()
	ps := shellquote.Join(psList...)
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := ExecPowershell(con.Rpc, session, ps)
	if err != nil {
		console.Log.Errorf("Execute Powershell error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed Powershell on target: %s\n", resp.GetAssemblyResponse().GetData())

	})
}

func ExecPowershell(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, ps string) (*clientpb.Task, error) {
	psCmd, _ := shellquote.Split(ps)
	var psBin bytes.Buffer
	if psCmd[0] != "" {
		content, err := os.ReadFile(psCmd[0])
		if err != nil {
			return nil, err
		}
		psBin.Write(content)
		psBin.WriteString("\n")
	}
	psBin.WriteString(strings.Join(psCmd[1:], " "))
	task, err := rpc.ExecutePowershell(console.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(psCmd[0]),
		Bin:    psBin.Bytes(),
		Type:   consts.ModulePowershell,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
