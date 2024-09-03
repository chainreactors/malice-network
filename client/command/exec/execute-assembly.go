package exec

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

func ExecuteAssemblyCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	output, _ := cmd.Flags().GetBool("output")
	name := filepath.Base(path)
	binData, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}
	execAssembly(name, binData, params, output, con)
}

func execAssembly(name string, binData []byte, args []string, output bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	var task *clientpb.Task
	task, err := con.Rpc.ExecuteAssembly(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   name,
		Bin:    binData,
		Output: output,
		Params: args,
		Type:   consts.ModuleExecuteAssembly,
	})

	if err != nil {
		console.Log.Errorf("%s", err.Error())
		return
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetAssemblyResponse()
		con.SessionLog(sid).Infof("%s output:\n%s", name, string(resp.Data))
	})
}
