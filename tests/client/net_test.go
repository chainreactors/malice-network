package client

import (
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func Test_Netstat(t *testing.T) {
	t.Log("Testing netstat")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Client.Netstat(rpc.Meta(), &implantpb.Request{
		Name: "netstat",
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

func Test_Curl(t *testing.T) {
	t.Log("Testing curl")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Client.Curl(rpc.Meta(), &implantpb.CurlRequest{
		Url:     "https://www.baidu.com",
		Timeout: 10,
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
