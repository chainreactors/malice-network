package common

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"math"
)

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
	return nil, fmt.Errorf("no response")
}

func ParseExecResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	resp := ctx.Spite.GetExecResponse()
	if resp.Stdout != nil {
		return fmt.Sprintf("pid: %d\n%s", resp.Pid, resp.Stdout), nil
	}
	return nil, fmt.Errorf("no response")
}
