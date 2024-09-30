package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestRegister(t *testing.T) {
	rpc := common.NewClient(common.DefaultGRPCAddr, []byte{1, 2, 3, 4})
	_, err := rpc.Listener.RegisterListener(context.Background(), &lispb.RegisterListener{Id: "test"})
	if err != nil {
		fmt.Println(err)
		return
	}
}
