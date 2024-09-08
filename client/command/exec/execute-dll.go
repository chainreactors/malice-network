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

func ExecuteDLLCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	path := cmd.Flags().Arg(0)
	sac, _ := common.ParseSacrifice(cmd)
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	task, err := ExecDLL(con.Rpc, session, path, entrypoint, sac)
	if err != nil {
		con.Log.Errorf("Execute DLL error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		session.Log.Consolef("Executed DLL on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func ExecDLL(rpc clientrpc.MaliceRPCClient, sess *repl.Session, pePath, entrypoint string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	dllBin, err := os.ReadFile(pePath)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(dllBin) != consts.DLLFile {
		return nil, errors.New("the file is not a DLL file")
	}
	task, err := rpc.ExecuteEXE(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:       filepath.Base(pePath),
		Bin:        dllBin,
		Type:       consts.ModuleExecuteExe,
		EntryPoint: entrypoint,
		Output:     true,
		Sacrifice:  sac,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func InlineDLLCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	pePath := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()
	entryPoint, _ := cmd.Flags().GetString("entrypoint")
	task, err := InlineDLL(con.Rpc, session, pePath, entryPoint, args)
	if err != nil {
		con.Log.Errorf("Execute Inline DLL error: %s", err)
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite)
		session.Log.Consolef("Execute Inline DLL error on target: %s\n", resp.GetAssemblyResponse().GetData())
	})
}

func InlineDLL(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path, entryPoint string, args []string) (*clientpb.Task, error) {
	dllBin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if helper.CheckPEType(dllBin) != consts.DLLFile {
		return nil, errors.New("the file is not a DLL file")
	}
	task, err := rpc.ExecuteEXE(repl.Context(sess), &implantpb.ExecuteBinary{
		Name:       filepath.Base(path),
		Bin:        dllBin,
		Type:       consts.ModuleExecuteExe,
		Params:     args,
		EntryPoint: entryPoint,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
