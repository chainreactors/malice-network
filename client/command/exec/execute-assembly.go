package exec

import (
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
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

func ExecuteAssemblyCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	//output, _ := cmd.Flags().GetBool("output")
	task, err := ExecAssembly(con.Rpc, session, path, sheshellquote.Join(params...))
	if err != nil {
		con.Log.Errorf("Execute error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp, _ := builtin.ParseAssembly(msg.(*implantpb.Spite))
		session.Log.Console(resp)
	})
}

func ExecAssembly(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, args string) (*clientpb.Task, error) {
	binData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	argsList, err := sheshellquote.Split(args)
	if err != nil {
		return nil, err
	}
	task, err := rpc.ExecuteAssembly(sess.Context(), &implantpb.ExecuteBinary{
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
