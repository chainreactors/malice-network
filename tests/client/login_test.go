package client

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/tests/common"
	"google.golang.org/grpc"
	"testing"
)

func TestLogin(t *testing.T) {
	options := common.RpcOptions()
	conn, err := grpc.Dial("localhost:5004", options...)
	if err != nil {
		fmt.Println(err)
	}
	t.Log("Dialing")
	client := clientrpc.NewMaliceRPCClient(conn)
	regReq := &clientpb.LoginReq{
		Host: "localhost",
		Port: 30009,
		Name: "test",
	}
	t.Log("Calling")
	// 调用服务器的 Register 方法并等待响应
	res, err := client.LoginClient(context.Background(), regReq)
	if err != nil {
		t.Log(err)
	}
	fmt.Println(res)
}
