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

func ExecuteBofCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	name := filepath.Base(path)
	binData, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	execBof(name, binData, params, con)
}

func execBof(name string, binData []byte, args []string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	var task *clientpb.Task
	task, err := con.Rpc.ExecuteBof(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   name,
		Bin:    binData,
		Params: args,
		Output: true,
		Type:   consts.ModuleExecuteBof,
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
