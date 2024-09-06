package exec

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

func ExecuteBofCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	task, err := ExecBof(con.Rpc, con.GetInteractive(), path, shellquote.Join(params...))
	if err != nil {
		repl.Log.Errorf("Execute BOF error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(con.GetInteractive().SessionId).Consolef("Executed BOF on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func ExecBof(rpc clientrpc.MaliceRPCClient, sess *repl.Session, bofPath string, paramString string) (*clientpb.Task, error) {
	bofBin, err := os.ReadFile(bofPath)
	if err != nil {
		return nil, err
	}
	param, _ := shellquote.Split(paramString)
	task, err := rpc.ExecuteBof(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(bofPath),
		Bin:    bofBin,
		Type:   consts.ModuleExecuteBof,
		Params: param,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
