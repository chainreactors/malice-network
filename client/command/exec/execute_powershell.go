package exec

import (
	"bytes"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"strings"
)

func ExecutePowershellCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	psPath := ctx.Flags.String("path")
	var err error
	var psBin bytes.Buffer
	if psPath != "" {
		content, err := os.ReadFile(psPath)
		if err != nil {
			console.Log.Errorf("%s\n", err.Error())
			return
		}
		psBin.Write(content)
		psBin.WriteString("\n")
	}
	paramString := ctx.Args.StringList("args")
	psBin.WriteString(strings.Join(paramString, " "))

	task, err := con.Rpc.ExecutePowershell(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name: filepath.Base(psPath),
		Bin:  psBin.Bytes(),
		Type: consts.ModulePowershell,
	})
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed Powershell on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}
