package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestRegister(t *testing.T) {
	client := common.NewClient(common.DefaultListenerAddr)
	promise := &commonpb.Promise{
		TaskId:    1,
		SessionId: 1,
		Body: &commonpb.Promise_Register{
			Register: &commonpb.Register{},
		},
	}

	resp := client.Request(promise)
	fmt.Println(resp)
}
