package client

import (
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/tests/common"
	"os"
	"testing"
)

func Test_Panic(t *testing.T) {
	t.Log("Testing panic")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Call("panic", &implantpb.Request{
		Name: "panic",
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

func Test_List_Module(t *testing.T) {
	t.Log("Testing list module")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	task, err := rpc.Client.ListModules(rpc.Meta(), &implantpb.Empty{})
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

func Test_Load_Module(t *testing.T) {
	t.Log("Testing load module")
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	bin, _ := os.ReadFile("modules.dll")
	task, err := rpc.Client.LoadModule(rpc.Meta(), &implantpb.LoadModule{
		Bin: bin,
		//Bundle: "netstat",
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
