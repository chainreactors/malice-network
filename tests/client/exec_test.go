package client

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"

	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestExec(t *testing.T) {
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call(consts.ModuleExecution, &implantpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"}})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(resp)
}
