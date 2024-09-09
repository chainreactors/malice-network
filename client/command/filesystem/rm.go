package filesystem

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func RmCmd(cmd *cobra.Command, con *repl.Console) {
	fileName := cmd.Flags().Arg(0)
	if fileName == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	task, err := Rm(con.Rpc, session, fileName)
	if err != nil {
		repl.Log.Errorf("Rm error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		session.Log.Consolef("Removed file success\n")
	})
}

func Rm(rpc clientrpc.MaliceRPCClient, session *repl.Session, fileName string) (*clientpb.Task, error) {
	task, err := rpc.Rm(session.Context(), &implantpb.Request{
		Name:  consts.ModuleRm,
		Input: fileName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
