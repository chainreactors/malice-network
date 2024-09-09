package exec

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func ExecuteCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	//token := ctx.Flags.Bool("token")
	//output, _ := cmd.Flags().GetBool("output")
	cmdStr := shellquote.Join(cmd.Flags().Args()...)
	task, err := Execute(con.Rpc, session, cmdStr)
	if err != nil {
		con.Log.Errorf("Execute error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetExecResponse()
		session.Log.Infof("pid: %d, status: %d", resp.Pid, resp.StatusCode)
		session.Log.Consolef("%s, output:\n%s", cmdStr, string(resp.Stdout))
	})

}

func Execute(rpc clientrpc.MaliceRPCClient, sess *repl.Session, cmd string) (*clientpb.Task, error) {
	cmdStrList, err := shellquote.Split(cmd)
	if err != nil {
		return nil, err
	}
	task, err := rpc.Execute(sess.Context(), &implantpb.ExecRequest{
		Path:   cmdStrList[0],
		Args:   cmdStrList[1:],
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
