package output

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
)

const (
	CALLBACK_OUTPUT      = 0
	CALLBACK_FILE        = 0x02
	CALLBACK_FILE_WRITE  = 0x08
	CALLBACK_FILE_CLOSE  = 0x09
	CALLBACK_SCREENSHOT  = 0x03
	CALLBACK_ERROR       = 0x0d
	CALLBACK_OUTPUT_OEM  = 0x1e
	CALLBACK_OUTPUT_UTF8 = 0x20
)

type BOFResponse struct {
	CallbackType uint8
	OutputType   uint8
	Length       uint32
	Data         []byte
}

type BOFResponses []*BOFResponse

func (bofResps BOFResponses) String() string {
	var results strings.Builder
	for _, resp := range bofResps {
		switch resp.CallbackType {
		case CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8:
			results.WriteString(string(resp.Data))
		case CALLBACK_ERROR:
			results.WriteString(fmt.Sprintf("Error occurred: %s", string(resp.Data)))
		case CALLBACK_SCREENSHOT:
			results.WriteString(fmt.Sprintf("Screenshot data received (size: %d)\n", len(resp.Data)-4))
		case CALLBACK_FILE:
			results.WriteString(fmt.Sprintf("[>] File operation started: %s\n", string(resp.Data[8:])))
		case CALLBACK_FILE_WRITE:
			results.WriteString(fmt.Sprintf("[+] File data received (size: %d) ...\n", len(resp.Data)-4))
		case CALLBACK_FILE_CLOSE:
			results.WriteString("[✓] File operation completed\n")
		default:
			results.WriteString(fmt.Sprintf("Callback type %d: %s\n", resp.CallbackType, string(resp.Data)))
		}
	}
	return results.String()
}

func ParseAssembly(ctx *clientpb.TaskContext) (interface{}, error) {
	return intermediate.ParseAssembly(ctx.Spite)
}

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
	if resp == nil || resp.Stdout != nil || resp.Stderr != nil {
		return fmt.Sprintf("pid: %d\n%s\n%s", resp.Pid, resp.Stdout, tui.RedFg.Render(string(resp.Stderr))), nil
	}

	return nil, fmt.Errorf("no response")
}

func ParseArrayResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	array := ctx.Spite.GetResponse().GetArray()
	if array == nil {
		return nil, fmt.Errorf("no response")
	}

	return array, nil
}

func FormatArrayResponse(ctx *clientpb.TaskContext) (string, error) {
	array, err := ParseArrayResponse(ctx)
	if err != nil {
		return "", err
	}
	return ctx.Task.Type + ":\n\t" + strings.Join(array.([]string), "\n\t"), nil
}

func ParseKVResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	set := ctx.Spite.GetResponse().GetKv()
	if set == nil {
		return nil, fmt.Errorf("no response")
	}
	return set, nil
}

func FormatKVResponse(ctx *clientpb.TaskContext) (string, error) {
	set, err := ParseKVResponse(ctx)
	if err != nil {
		return "", err
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Key", "Key", 20),
		table.NewColumn("Value", "Value", 70),
		//{Title: "Key", Width: 20},
		//{Title: "Value", Width: 70},
	}, true)
	for k, v := range set.(map[string]string) {
		row = table.NewRow(
			table.RowData{
				"Key":   k,
				"Value": v,
			})
		//	table.Row{
		//	k,
		//	v,
		//}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.Title = ctx.Task.Type
	return tableModel.View(), nil
}

func ParseBOFResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	reader := bytes.NewReader(ctx.Spite.GetBinaryResponse().GetData())
	var bofResps BOFResponses

	for {
		bofResp := &BOFResponse{}

		err := binary.Read(reader, binary.LittleEndian, &bofResp.OutputType)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read OutputType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.CallbackType)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read CallbackType: %v", err)
		}

		err = binary.Read(reader, binary.LittleEndian, &bofResp.Length)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read Length: %v", err)
		}

		strData := make([]byte, bofResp.Length)
		_, err = io.ReadFull(reader, strData)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read StrData: %v", err)
		}

		bofResp.Data = strData

		bofResps = append(bofResps, bofResp)
	}

	return bofResps, nil
}
