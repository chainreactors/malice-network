package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"math"
)

func ParseSacrifice(cmd *cobra.Command) (*implantpb.SacrificeProcess, error) {
	ppid, _ := cmd.Flags().GetUint("ppid")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	hidden, _ := cmd.Flags().GetBool("hidden")
	disableEtw, _ := cmd.Flags().GetBool("etw")
	return builtin.NewSacrificeProcessMessage(int64(ppid), hidden, isBlockDll, disableEtw, argue)
}

func ParseBinaryParams(cmd *cobra.Command) (string, []string, bool, uint32) {
	path := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()[1:]
	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return path, args, !quiet, timeout
}

func ParseFullBinaryParams(cmd *cobra.Command) (string, []string, bool, uint32, string, string) {
	path, args, output, timeout := ParseBinaryParams(cmd)
	arch, _ := cmd.Flags().GetString("arch")
	process, _ := cmd.Flags().GetString("process")
	return path, args, output, timeout, arch, process
}

func ParseAssembly(ctx *clientpb.TaskContext) (interface{}, error) {
	return builtin.ParseAssembly(ctx.Spite)
}

func NewExecutable(module string, path string, args []string, arch string, output bool, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	binary, err := builtin.NewBinary(module, path, args, output, math.MaxUint32, arch, "", sac)
	if err != nil {
		return nil, err
	}
	binary.Output = output
	return binary, nil
}

func NewBinary(module string, path string, args []string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	if name, ok := consts.ModuleAliases[module]; ok {
		module = name
	}

	return builtin.NewBinary(module, path, args, output, timeout, arch, process, sac)
}

func ParseStatus(ctx *clientpb.TaskContext) (interface{}, error) {
	return builtin.ParseStatus(ctx.Spite)
}

func ParseResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	resp := ctx.Spite.GetResponse()
	if resp != nil {
		return resp.GetOutput(), nil
	}
	return nil, fmt.Errorf("not response")
}
