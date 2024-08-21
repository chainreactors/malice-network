package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"strings"

	"google.golang.org/protobuf/proto"
)

func ExecuteCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}

	cmdPath := ctx.Args.String("command")
	args := ctx.Args.StringList("arguments")
	//token := ctx.Flags.Bool("token")
	output := ctx.Flags.Bool("output")
	stdout := ctx.Flags.String("stdout")
	stderr := ctx.Flags.String("stderr")
	//saveLoot := ctx.Flags.Bool("loot")
	//saveOutput := ctx.Flags.Bool("save")
	//ppid := ctx.Flags.Uint("ppid")
	//hostName := getHostname(session, beacon)

	var resp *clientpb.Task
	var err error
	resp, err = con.Rpc.Execute(con.ActiveTarget.Context(), &implantpb.ExecRequest{
		Path:   cmdPath,
		Args:   args,
		Output: output,
		Stderr: stderr,
		Stdout: stdout,
	})
	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}

	con.AddCallback(resp.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetExecResponse()
		sid := con.GetInteractive().SessionId
		con.SessionLog(sid).Infof("pid: %d, status: %d", resp.Pid, resp.StatusCode)
		con.SessionLog(sid).Consolef("%s %s , output:\n%s", cmdPath, strings.Join(args, " "), string(resp.Stdout))
	})

}
