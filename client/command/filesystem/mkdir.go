package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func MkdirCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	mkdir(path, con)
}

func mkdir(path string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId

	mkdirTask, err := con.Rpc.Mkdir(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleMkdir,
		Input: path,
	})
	if err != nil {
		console.Log.Errorf("Mkdir error: %v", err)
		return
	}
	con.AddCallback(mkdirTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Created directory\n")
	})
}
