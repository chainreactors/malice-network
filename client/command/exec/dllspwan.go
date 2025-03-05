package exec

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"math"
	"os"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/spf13/cobra"
)

func ExecuteDLLSpawnCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	sac := common.ParseSacrificeFlags(cmd)
	entrypoint, _ := cmd.Flags().GetString("entrypoint")
	binPath, _ := cmd.Flags().GetString("binPath")
	path, data, output, timeout, arch, process := common.ParseFullBinaryDataFlags(cmd)
	task, err := ExecuteDLLSpawn(con.Rpc, session, path, entrypoint, data, binPath, output, timeout, arch, process, sac)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func ExecuteDLLSpawn(rpc clientrpc.MaliceRPCClient, sess *core.Session, dllPath string, entrypoint string, data string, binPath string, out bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	binary, err := output.NewBinaryData(consts.ModuleDllSpawn, dllPath, data, out, timeout, arch, process, sac)
	if err != nil {
		return nil, err
	}

	binPath = fileutils.FormatWindowPath(binPath)
	if _, err := os.Stat(binPath); err == nil {
		binData, err := os.ReadFile(binPath)
		if err != nil {
			return nil, err
		}
		binary.Data = binData
	}

	if arch == "" {
		arch = sess.Os.Arch
	}

	binary.EntryPoint = entrypoint
	if pe.CheckPEType(binary.Bin) != consts.DLLFile {
		return nil, errors.New("the file is not a DLL file")
	}
	task, err := rpc.ExecuteEXE(sess.Context(), binary)
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterDLLSpawnFunc(con *repl.Console) {

	con.RegisterImplantFunc(
		consts.ModuleDllSpawn,
		ExecuteDLLSpawn,
		"bdllspawn",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, ppid uint32, path string) (*clientpb.Task, error) {
			sac, _ := intermediate.NewSacrificeProcessMessage(ppid, false, true, true, "")
			return ExecuteDLLSpawn(rpc, sess, path, "", "", "", true, math.MaxUint32, sess.Os.Arch, "", sac)
		},
		output.ParseAssembly,
		nil)
	// sess *core.Session, dllPath string, entrypoint string, args []string, binPath string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess
	con.AddCommandFuncHelper(
		consts.ModuleDllSpawn,
		consts.ModuleDllSpawn,
		consts.ModuleDllSpawn+`(active(),"example.dll",{},true,60,"","",new_sacrifice(1234,false,true,true,""))`,
		[]string{
			"session: special session",
			"dllPath",
			"entrypoint",
			"args",
			"binPath",
			"output",
			"timeout",
			"arch",
			"process",
			"sac: sacrifice process",
		},
		[]string{"task"})

}
