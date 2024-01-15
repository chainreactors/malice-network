package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

func ExecuteAssemblyCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	path := ctx.Args.String("path")
	args := ctx.Args.StringList("arguments")
	name := filepath.Base(path)
	binData, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}
	var task *clientpb.Task

	//con.SpinUntil(fmt.Sprintf("Executing %s %s ...", cmdPath, strings.Join(args, " ")), ctrl)
	task, err = con.Rpc.ExecuteAssembly(con.ActiveTarget.Context(), &pluginpb.ExecuteLoadAssembly{
		Name:   name,
		Bin:    binData,
		Params: args,
		Type:   consts.CSharpPlugin,
	})

	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*pluginpb.AssemblyResponse)
		if resp.Status == 0 {
			console.Log.Infof("%s output:\n%s", name, string(resp.Data))
		} else {
			console.Log.Errorf("%s %s ", ctx.Command.Name, resp.Err)
		}
	})
}
