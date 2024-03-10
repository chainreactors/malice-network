package client

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

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
