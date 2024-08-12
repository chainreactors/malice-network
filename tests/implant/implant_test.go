package implant

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/tests/common"
	"testing"
	"time"
)

var (
	uploadResp = &implantpb.UploadRequest{
		Name:   "test.txt",
		Target: ".",
		Priv:   0644,
		Data:   make([]byte, 1000),
	}
)

func TestImplant(t *testing.T) {
	logs.Log.SetLevel(logs.Debug)
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(5 * time.Second)
	go implant.Run()
	select {}
	//client := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	//resp, err := client.Call(consts.ExecutionStr, &implantpb.ExecRequest{
	//	Path: "/bin/bash",
	//	Args: []string{"whoami"},
	//})
	//if err != nil {
	//	panic(err.Error())
	//}
	//fmt.Println(resp)
}
