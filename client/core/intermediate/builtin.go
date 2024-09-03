package intermediate

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
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

func NewBinaryMessage(pluginName, module, filename, args string, sarcifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	content, _ := ReadResourceFile(pluginName, filename)
	params, err := shellquote.Split(args)
	if err != nil {
		return nil, err
	}
	return &implantpb.ExecuteBinary{
		Name:      filename,
		Bin:       []byte(content),
		Type:      module,
		Params:    params,
		Output:    true,
		Sacrifice: sarcifice,
	}, nil
}

func NewSacrificeProcessMessage(processName string, ppid int64, block_dll bool, argue string, args string) (*implantpb.SacrificeProcess, error) {
	params, err := shellquote.Split(processName + " " + args)
	if err != nil {
		return nil, err
	}
	return &implantpb.SacrificeProcess{
		Ppid:     uint32(ppid),
		Output:   true,
		BlockDll: block_dll,
		Argue:    argue,
		Params:   params,
	}, err
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
	if task.Spite == nil {
		handler.HandleMaleficError(task.Spite)
		return nil, nil
	}
	logs.Log.Consolef("%v", task.Spite.GetBody())
	return task.Spite, nil
}

func PrintAssembly(response *implantpb.AssemblyResponse) (string, error) {
	if response.GetErr() != "" {
		return "", fmt.Errorf("exit status: %d, %s", response.Status, response.Err)
	}
	logs.Log.Console(string(response.GetData()))
	return string(response.GetData()), nil
}
