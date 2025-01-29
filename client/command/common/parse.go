package common

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"math"
)

func NewSacrifice(ppid uint32, hidden, block_dll, disable_etw bool, argue string) *implantpb.SacrificeProcess {
	sac, _ := intermediate.NewSacrificeProcessMessage(ppid, hidden, block_dll, disable_etw, argue)
	return sac
}

func NewExecutable(module string, path string, args []string, arch string, output bool, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	bin, err := intermediate.NewBinary(module, path, args, output, math.MaxUint32, arch, "", sac)
	if err != nil {
		return nil, err
	}
	bin.Output = output
	return bin, nil
}

func NewBinary(module string, path string, args []string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	if name, ok := consts.ModuleAliases[module]; ok {
		module = name
	}

	return intermediate.NewBinary(module, path, args, output, timeout, arch, process, sac)
}

func NewBinaryData(module string, path string, data string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	if name, ok := consts.ModuleAliases[module]; ok {
		module = name
	}

	return intermediate.NewBinaryData(module, path, data, output, timeout, arch, process, sac)
}
