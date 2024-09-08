package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

// ExecuteExeCmd - Execute PE on sacrifice process
func ExecuteExeCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	sac, _ := common.ParseSacrifice(cmd)
	task, err := ExecExe(con.Rpc, con.GetInteractive(), path, sac)
	if err != nil {
		con.Log.Errorf("Execute PE error: %v", err)
		return
	}
	session := con.GetInteractive()
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		session.Log.Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func ExecExe(rpc clientrpc.MaliceRPCClient, sess *repl.Session, pePath string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(pePath)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")
	}
	task, err := rpc.ExecuteEXE(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:      filepath.Base(pePath),
		Bin:       peBin,
		Type:      consts.ModuleExecuteExe,
		Output:    true,
		Sacrifice: sac,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// InlineExeCmd - Execute PE in current process
func InlineExeCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()

	pePath := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()
	task, err := InlineExe(con.Rpc, session, pePath, args)
	if err != nil {
		con.Log.Errorf("Execute PE error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			session.Log.Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}

func InlineExe(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string, args []string) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")

	}
	task, err := rpc.ExecuteEXE(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:   filepath.Base(path),
		Bin:    peBin,
		Type:   consts.ModuleExecuteExe,
		Output: true,
		Params: args,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}
