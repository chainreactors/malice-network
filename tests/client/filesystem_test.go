package client

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestPwd(t *testing.T) {
	t.Log("Testing pwd")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModulePwd, &implantpb.Empty{})
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(task)
	resp, err := rpc.WaitResponse(task.(*clientpb.Task))
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(resp)
}
