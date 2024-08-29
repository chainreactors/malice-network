package filesystem

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ChmodCmd(cmd *cobra.Command, con *console.Console) {
	mode := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)
	if mode == "" || path == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	chmod(path, mode, con)

}

func chmod(path, mode string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	chmodTask, err := con.Rpc.Chmod(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleChmod,
		Args: []string{path, mode},
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Chmod error: %v", err)
		return
	}
	con.AddCallback(chmodTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		console.Log.Consolef("Chmod success\n")
	})
}
