package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"io"
	"math"
)

func ParseAssembly(ctx *clientpb.TaskContext) (interface{}, error) {
	return intermediate.ParseAssembly(ctx.Spite)
}

func NewSacrifice(ppid int64, hidden, block_dll, disable_etw bool, argue string) *implantpb.SacrificeProcess {
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

func UpdateClrBinary(binary *implantpb.ExecuteBinary, bypassETW, bypassAMSI bool) {
	if !bypassETW && bypassAMSI {
		return
	}

	binary.Param = make(map[string]string)
	if bypassETW {
		binary.Param["bypass_etw"] = ""
	}

	if bypassAMSI {
		binary.Param["bypass_amsi"] = ""
	}
}

func NewBinary(module string, path string, args []string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	if name, ok := consts.ModuleAliases[module]; ok {
		module = name
	}

	return intermediate.NewBinary(module, path, args, output, timeout, arch, process, sac)
}

func ParseStatus(ctx *clientpb.TaskContext) (interface{}, error) {
	return intermediate.ParseStatus(ctx.Spite)
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

func ParseBOFResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	reader := bytes.NewReader(ctx.Spite.GetBinaryResponse().GetData())
	var bofResps pe.BOFResponses

	for {
		bofResp := &pe.BOFResponse{}

		err := binary.Read(reader, binary.LittleEndian, &bofResp.OutputType)
		if err != nil {
			return nil, fmt.Errorf("failed to read OutputType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.CallbackType)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read CallbackType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.Length)
		if err != nil {
			return nil, fmt.Errorf("failed to read Length: %v", err)
		}

		strData := make([]byte, bofResp.Length)
		_, err = io.ReadFull(reader, strData)
		if err != nil {
			return nil, fmt.Errorf("failed to read Str: %v", err)
		}
		bofResp.Data = strData

		bofResps = append(bofResps, bofResp)
	}

	return bofResps, nil
}
