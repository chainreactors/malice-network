package exec

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	sheshellquote "github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

func ExecuteAssemblyCmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	//output, _ := cmd.Flags().GetBool("output")
	task, err := ExecAssembly(con.Rpc, session, path, sheshellquote.Join(params...))
	if err != nil {
		console.Log.Errorf("Execute error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetAssemblyResponse()
		con.SessionLog(sid).Infof("%s output:\n%s", filepath.Base(path), string(resp.Data))
	})
}

func ExecAssembly(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path, args string) (*clientpb.Task, error) {
	binData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	argsList, err := sheshellquote.Split(args)
	if err != nil {
		return nil, err
	}
	task, err := rpc.ExecuteAssembly(console.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(path),
		Bin:    binData,
		Params: argsList,
		Output: true,
		Type:   consts.ModuleExecuteAssembly,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
