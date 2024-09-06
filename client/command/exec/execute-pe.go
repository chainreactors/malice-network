package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
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
	sac, _ := builtin.NewSacrificeProcessMessage(processname, int64(ppid), isBlockDll, argue, shellquote.Join(params...))
	task, err := ExecPE(con.Rpc, con.GetInteractive(), path, sac)
	if err != nil {
		console.Log.Errorf("Execute PE error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		con.SessionLog(con.GetInteractive().SessionId).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func ExecPE(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, pePath string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(pePath)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")
	}
	task, err := rpc.ExecutePE(console.Context(sess), &implantpb.ExecuteBinary{
		Name:      filepath.Base(pePath),
		Bin:       peBin,
		Type:      consts.ModuleExecutePE,
		Output:    true,
		Sacrifice: sac,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// InlinePECmd - Execute PE in current process
func InlinePECmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := con.GetInteractive().SessionId
	pePath := cmd.Flags().Arg(0)
	task, err := InlinePE(con.Rpc, session, pePath)
	if err != nil {
		console.Log.Errorf("Execute PE error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			con.SessionLog(sid).Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}

func InlinePE(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path string) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")

	}
	task, err := rpc.ExecutePE(console.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(path),
		Bin:    peBin,
		Type:   consts.ModuleExecutePE,
		Output: true,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
