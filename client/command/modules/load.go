package modules

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
)

func LoadModuleCmd(cmd *cobra.Command, con *console.Console) {
	bundle := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	data, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("Error reading file: %v", err)
		return
	}
	loadModule(bundle, data, con)
}

func loadModule(bundle string, data []byte, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	loadTask, err := con.Rpc.LoadModule(con.ActiveTarget.Context(), &implantpb.LoadModule{
		Bundle: bundle,
		Bin:    data,
	})
	if err != nil {
		console.Log.Errorf("LoadModule error: %v", err)
		return
	}
	con.AddCallback(loadTask.TaskId, func(msg proto.Message) {
		//modules := msg.(*implantpb.Spite).GetModules()
		con.SessionLog(sid).Infof("LoadModule: success")
	})
}
