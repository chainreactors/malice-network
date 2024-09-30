package client

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestExec(t *testing.T) {
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleExecution, &implantpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"}})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	resp, err := rpc.WaitResponse(task.(*clientpb.Task))
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(resp)
	fmt.Println(resp)
}

func TestExecuteShellcode(t *testing.T) {
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleExecuteShellcode, &implantpb.ExecuteShellcode{
		Name:   "mimikatz",
		Bin:    []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90},
		Params: []string{"token::list", "exit"},
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	resp, err := rpc.WaitResponse(task.(*clientpb.Task))
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(resp)
	fmt.Println(resp)
}
