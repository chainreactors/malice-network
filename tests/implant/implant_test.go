package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestImplant(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, []byte{1, 2, 3, 4})
	go implant.Run()
	client := common.NewClient(common.DefaultGRPCAddr, []byte{1, 2, 3, 4})
	resp, err := client.Call(consts.ExecutionStr, &pluginpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"},
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(resp)
}
