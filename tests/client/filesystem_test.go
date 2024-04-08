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
	task, err := rpc.Call(consts.ModulePwd, &implantpb.Request{})
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

func TestCd(t *testing.T) {
	t.Log("Testing cd")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleCd, &implantpb.Request{
		Name:  "cd",
		Input: "D:\\",
	})
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

func TestLs(t *testing.T) {
	t.Log("Testing ls")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleLs, &implantpb.Request{
		Name:  "ls",
		Input: ".",
	})
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

func TestRm(t *testing.T) {
	t.Log("Testing rm")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleRm, &implantpb.Request{
		Name:  "rm",
		Input: "test.txt",
	})
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

func TestMv(t *testing.T) {
	t.Log("Testing mv")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleMv, &implantpb.Request{
		Name:  "mv",
		Input: "test.txt test1.txt",
	})
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

func TestMkdir(t *testing.T) {
	t.Log("Testing mkdir")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call(consts.ModuleMkdir, &implantpb.Request{
		Name:  "mkdir",
		Input: "test",
	})
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
