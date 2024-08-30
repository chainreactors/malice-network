package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func CdCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	cd(path, con)
}

func cd(path string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	cdTask, err := con.Rpc.Cd(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: path,
	})
	if err != nil {
		console.Log.Errorf("Cd error: %v", err)
		return
	}
	con.AddCallback(cdTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("Changed directory to: %s\n", path)
	})
}
