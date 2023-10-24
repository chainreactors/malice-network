package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestRegister(t *testing.T) {
	rpc := common.NewRPC(common.DefaultGRPCAddr)
	_, err := rpc.Listener.RegisterListener(context.Background(), &lispb.RegisterListener{ListenerId: "test"})
	if err != nil {
		fmt.Println(err)
		return
	}
}
