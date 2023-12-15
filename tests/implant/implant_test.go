package implant

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
	"time"
)

func TestImplant(t *testing.T) {
	logs.Log.SetLevel(logs.Debug)
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(5 * time.Second)
	go implant.Run()
	client := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	//resp, err := client.Call(consts.ExecutionStr, &pluginpb.ExecRequest{
	//	Path: "/bin/bash",
	//	Args: []string{"whoami"},
	//})
	resp, err := client.Call(consts.UploadStr, &pluginpb.UploadRequest{
		Name:   "test.txt",
		Target: ".",
		Priv:   0o644,
		Data:   make([]byte, 1000),
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(resp)
}
