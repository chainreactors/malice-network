package exec

import (
	"bytes"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"strings"
)

func ExecutePowershellCmd(cmd *cobra.Command, con *console.Console) {
	psPath := cmd.Flags().Arg(0)
	argsString := cmd.Flags().Arg(1)
	paramString := strings.Split(argsString, ",")
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
	psBin.WriteString(strings.Join(paramString, " "))

	execPowershell(psPath, psBin.Bytes(), con)
}

func execPowershell(psPath string, psBin []byte, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := con.Rpc.ExecutePowershell(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name: filepath.Base(psPath),
		Bin:  psBin,
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
