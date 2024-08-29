package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func KillCmd(cmd *cobra.Command, con *console.Console) {
	pid := cmd.Flags().Arg(0)
	if pid == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	kill(pid, con)
}

func kill(pid string, con *console.Console) {
	session := con.GetInteractive()
	sid := con.GetInteractive().SessionId
	if session == nil {
		return
	}
	killTask, err := con.Rpc.Kill(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleKill,
		Input: pid,
	})
	if err != nil {
		console.Log.Errorf("Kill error: %v", err)
		return
	}
	con.AddCallback(killTask.TaskId, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Killed process\n")
	})
}
