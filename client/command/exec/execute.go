package exec

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ExecuteCmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	//token := ctx.Flags.Bool("token")
	//output, _ := cmd.Flags().GetBool("output")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Execute(con.Rpc, session, cmdStr)
	if err != nil {
		console.Log.Errorf("Execute error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetExecResponse()
		con.SessionLog(sid).Infof("pid: %d, status: %d", resp.Pid, resp.StatusCode)
		con.SessionLog(sid).Consolef("%s, output:\n%s", cmdStr, string(resp.Stdout))
	})

}

func Execute(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, cmd string) (*clientpb.Task, error) {
	cmdStrList, err := shellquote.Split(cmd)
	if err != nil {
		return nil, err
	}
	task, err := rpc.Execute(console.Context(sess), &implantpb.ExecRequest{
		Path:   cmdStrList[0],
		Args:   cmdStrList[1:],
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
