package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
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
	var resp *clientpb.Task

	//con.SpinUntil(fmt.Sprintf("Executing %s %s ...", cmdPath, strings.Join(args, " ")), ctrl)
	resp, err = con.Rpc.ExecuteAssembly(con.ActiveTarget.Context(), &pluginpb.ExecuteLoadAssembly{
		Name:   name,
		Bin:    binData,
		Params: args,
		Type:   consts.CSharpPlugin,
	})

	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}
	con.AddCallback(resp.TaskId, func(task *clientpb.Task) {
		if task.Status == 0 {
			console.Log.Infof("%s output:\n%s", name, string(resp.Data))
		} else {
			console.Log.Errorf("%s %s ", ctx.Command.Name, task.Error)
		}
	})
}
