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

func CdCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		repl.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := Cd(con.Rpc, con.GetInteractive(), path)
	if err != nil {
		repl.Log.Errorf("Cd error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		_ = msg.(*implantpb.Spite).GetResponse()
		con.SessionLog(sid).Consolef("Changed directory to: %s\n", path)
	})

}

func Cd(rpc clientrpc.MaliceRPCClient, session *repl.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Cd(repl.Context(session), &implantpb.Request{
		Name:  consts.ModuleCd,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
