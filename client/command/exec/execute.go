package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
	"strings"
)

func ExecuteCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	cmdPath := ctx.Args.String("command")
	args := ctx.Args.StringList("arguments")
	//token := ctx.Flags.Bool("token")
	output := ctx.Flags.Bool("output")
	stdout := ctx.Flags.String("stdout")
	stderr := ctx.Flags.String("stderr")
	saveLoot := ctx.Flags.Bool("loot")
	saveOutput := ctx.Flags.Bool("save")
	//ppid := ctx.Flags.Uint("ppid")
	//hostName := getHostname(session, beacon)
	var captureOutput bool = output || saveLoot || saveOutput

	if output {
		console.Log.Error("Using --output in beacon mode, if the command blocks the task will never complete\n")
	}

	var resp *clientpb.Task
	var err error
	resp, err = con.Rpc.Execute(con.ActiveTarget.Context(), &pluginpb.ExecRequest{
		Path:   cmdPath,
		Args:   args,
		Output: captureOutput,
		Stderr: stderr,
		Stdout: stdout,
	})
	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}

	con.AddCallback(resp.TaskId, func(msg proto.Message) {
		resp := msg.(*commonpb.Spite).GetExecResponse()
		sid := con.ActiveTarget.GetInteractive().SessionId
		con.SessionLog(sid).Infof("pid: %d, status: %d", resp.Pid, resp.StatusCode)
		if resp.StatusCode == 0 {
			con.SessionLog(sid).Consolef("%s %s , output:\n%s", cmdPath, strings.Join(args, " "), string(resp.Stdout))
		} else {
			con.SessionLog(sid).Errorf("%s %s ", ctx.Command.Name, resp.Stderr)
		}
	})

}
