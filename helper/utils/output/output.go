package output

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"math"
	"strings"
)

func ParseBinaryResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	return intermediate.ParseBinaryResponse(ctx.Spite)
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
	ok, err := intermediate.ParseStatus(ctx.Spite)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("task: %d %t", ctx.Task.TaskId, ok), nil
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
	if resp.Stdout != nil || resp.Stderr != nil {
		var prefix string = ""
		if ctx.Task.Cur == 1 {
			prefix = "\n"
		}
		return fmt.Sprintf("%spid: %d ,task: %d cur: %d \n%s\n%s", prefix, resp.Pid, ctx.Task.TaskId, ctx.Task.Cur, encoders.AutoDecode(resp.Stdout), tui.RedFg.Render(encoders.AutoDecode(resp.Stderr))), nil
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
		table.NewColumn("Key", "Key", 30),
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
