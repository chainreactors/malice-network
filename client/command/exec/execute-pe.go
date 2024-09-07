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

// ExecutePECmd - Execute PE on sacrifice process
func ExecutePECmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	sac, _ := common.ParseSacrifice(cmd)
	task, err := ExecPE(con.Rpc, con.GetInteractive(), path, sac)
	if err != nil {
		repl.Log.Errorf("Execute PE error: %v", err)
		return
	}
	session := con.GetInteractive()
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		session.Log.Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func ExecPE(rpc clientrpc.MaliceRPCClient, sess *repl.Session, pePath string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(pePath)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")
	}
	task, err := rpc.ExecutePE(repl.Context(sess), &implantpb.ExecuteBinary{
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
func InlinePECmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	pePath := cmd.Flags().Arg(0)
	task, err := InlinePE(con.Rpc, session, pePath)
	if err != nil {
		repl.Log.Errorf("Execute PE error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		if !(resp.Status.Error != "") {
			session.Log.Consolef("Executed PE on target: %s\n", resp.GetAssemblyResponse().GetData())
		}
	})
}

func InlinePE(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
	peBin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(peBin) != consts.EXEFile {
		return nil, errors.New("the file is not a PE file")

	}
	task, err := rpc.ExecutePE(repl.Context(sess), &implantpb.ExecuteBinary{
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
