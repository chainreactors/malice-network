package builtin

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"math"
	"os"
	"path/filepath"
)

func GetResourceFile(pluginName, filename string) (string, error) {
	resourcePath := filepath.Join(assets.GetMalsDir(), pluginName, "resources", filename)
	return resourcePath, nil
}

func ReadResourceFile(pluginName, filename string) (string, error) {
	resourcePath, _ := GetResourceFile(pluginName, filename)
	content, err := os.ReadFile(resourcePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func NewSacrificeProcessMessage(ppid int64, hidden, block_dll, disable_etw bool, argue string) (*implantpb.SacrificeProcess, error) {
	return &implantpb.SacrificeProcess{
		Ppid:     uint32(ppid),
		Hidden:   hidden,
		BlockDll: block_dll,
		Argue:    argue,
		Etw:      !disable_etw,
	}, nil
}

func NewBinary(module string, path string, args []string, output bool, timeout uint32, arch string, process string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	bin, err := os.ReadFile(file.FormatWindowPath(path))
	if err != nil {
		return nil, err
	}

	return &implantpb.ExecuteBinary{
		Name:        filepath.Base(path),
		Bin:         bin,
		Type:        module,
		Args:        args,
		Output:      output,
		Timeout:     timeout,
		Arch:        consts.ArchMap[arch],
		ProcessName: process,
		Sacrifice:   sac,
	}, nil
}

func NewExecutable(module string, path string, args []string, arch string, sac *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	process := filepath.Base(path)
	return &implantpb.ExecuteBinary{
		Name:        filepath.Base(path),
		Bin:         bin,
		Type:        module,
		Args:        args,
		Output:      true,
		Timeout:     math.MaxUint32,
		Arch:        consts.ArchMap[arch],
		ProcessName: process,
		Sacrifice:   sac,
	}, nil
}

func WaitResult(rpc clientrpc.MaliceRPCClient, task *clientpb.Task) (*clientpb.TaskContext, error) {
	task.Need = -1
	content, err := rpc.WaitTaskFinish(context.Background(), task)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func GetResult(rpc clientrpc.MaliceRPCClient, task *clientpb.Task, index int32) (*clientpb.TaskContext, error) {
	task.Need = index
	content, err := rpc.GetTaskContent(context.Background(), task)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func PrintTask(task *clientpb.TaskContext) (*implantpb.Spite, error) {
	logs.Log.Consolef("Session: %s, Task: %d, Index:%d \n", task.Task.SessionId, task.Task.TaskId, task.Task.Need)
	err := handler.HandleMaleficError(task.Spite)
	if err != nil {
		return nil, err
	}
	logs.Log.Consolef("%v", task.Spite.GetBody())
	return task.Spite, nil
}

func ParseAssembly(spite *implantpb.Spite) (string, error) {
	err := handler.HandleMaleficError(spite)
	if err != nil {
		return "", err
	}
	response := spite.GetAssemblyResponse()
	if response == nil {
		return "", fmt.Errorf("assembly response is nil")
	}
	if response.GetErr() != "" {
		return fmt.Sprintf("exit status: %d, %s", response.Status, response.Err), nil
	}
	return string(response.GetData()), nil
}

func ParseStatus(spite *implantpb.Spite) (bool, error) {
	if spite.Error == 6 {
		return false, nil
	} else if spite.Error == 0 {
		return true, nil
	} else {
		return false, handler.HandleMaleficError(spite)
	}
}
