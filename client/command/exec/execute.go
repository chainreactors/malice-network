package exec

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"strings"

	"google.golang.org/protobuf/proto"
)

func ExecuteCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	//token := ctx.Flags.Bool("token")
	output, _ := cmd.Flags().GetBool("output")
	execute(path, params, output, con)
}

func execute(cmd string, args []string, output bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	var resp *clientpb.Task
	var err error
	resp, err = con.Rpc.Execute(con.ActiveTarget.Context(), &implantpb.ExecRequest{
		Path:   cmd,
		Args:   args,
		Output: output,
	})
	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}

	con.AddCallback(resp.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetExecResponse()
		con.SessionLog(sid).Infof("pid: %d, status: %d", resp.Pid, resp.StatusCode)
		con.SessionLog(sid).Consolef("%s %s , output:\n%s", cmd, strings.Join(args, " "), string(resp.Stdout))
	})
}
