package exec

import (
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	path := cmd.Flags().Arg(0)
	params := cmd.Flags().Args()[1:]
	ppid, _ := cmd.Flags().GetUint("ppid")
	processName, _ := cmd.Flags().GetString("process")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	// TODO arch judgment
	//arch, _ := cmd.Flags().GetString("arch")
	sac, _ := builtin.NewSacrificeProcessMessage(processName, int64(ppid), isBlockDll, argue, shellquote.Join(params...))
	task, err := ExecShellcode(con.Rpc, session, path, sac)
	if err != nil {
		repl.Log.Errorf("Execute shellcode error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
	})

}

func ExecShellcode(rpc clientrpc.MaliceRPCClient, sess *repl.Session, shellcodePath string,
	sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	shellcodeBin, err := os.ReadFile(shellcodePath)
	if err != nil {
		return nil, err
	}

	task, err := rpc.ExecuteShellcode(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:      filepath.Base(shellcodePath),
		Bin:       shellcodeBin,
		Type:      consts.ModuleExecuteShellcode,
		Output:    true,
		Sacrifice: sac,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func InlineShellcodeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	path := cmd.Flags().Arg(0)
	task, err := InlineShellcode(con.Rpc, session, path)
	if err != nil {
		repl.Log.Errorf("Execute inline shellcode error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(sid).Consolef("Executed inline shellcode on target: %s\n", resp.GetAssemblyResponse().GetData())
	})

}

func InlineShellcode(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	shellcodeTask, err := rpc.ExecuteShellcode(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(path),
		Bin:    data,
		Type:   consts.ModuleExecuteShellcode,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return shellcodeTask, err
}
