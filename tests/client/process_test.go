package client

import (
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func Test_Kill(t *testing.T) {
	t.Log("Testing kill")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Client.Kill(rpc.Meta(), &implantpb.Request{
		Name:  "kill",
		Input: "10148",
	})
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(task)
	resp, err := rpc.WaitResponse(task)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(resp)
}

func Test_Ps(t *testing.T) {
	t.Log("Testing ps")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Client.Ps(rpc.Meta(), &implantpb.Request{
		Name: "ps",
	})
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(task)
	resp, err := rpc.WaitResponse(task)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(resp)
}
