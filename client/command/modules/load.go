package modules

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
)

func LoadModuleCmd(cmd *cobra.Command, con *console.Console) {
	bundle := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := LoadModule(con.Rpc, session, bundle, path)
	if err != nil {
		console.Log.Errorf("LoadModule error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		//modules := msg.(*implantpb.Spite).GetModules()
		con.SessionLog(sid).Infof("LoadModule: success")
	})
}

func LoadModule(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, bundle string, path string) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	task, err := rpc.LoadModule(console.Context(session), &implantpb.LoadModule{
		Bundle: bundle,
		Bin:    data,
	})

	if err != nil {
		return nil, err
	}
	return task, nil
}
