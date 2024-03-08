package client

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestPwd(t *testing.T) {
	t.Log("Testing pwd")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call(consts.ModulePwd, &implantpb.Empty{})
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(resp)
}
