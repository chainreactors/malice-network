package client

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestBroadcast(t *testing.T) {
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	_, err := rpc.Call(consts.BroadcastStr, &clientpb.Event{
		EventType: consts.EventBroadcast,
		Data:      []byte("broadcast test"),
	})
	if err != nil {
		fmt.Println(err.Error())
	}
}
