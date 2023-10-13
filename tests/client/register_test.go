package client

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
	"testing"
)

func TestRegister(t *testing.T) {
	conn, err := grpc.Dial("127.0.0.1:51004", grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	client := clientrpc.NewMaliceRPCClient(conn)
	regReq := &clientpb.RegisterReq{
		Host: "127.0.0.1",
		User: "test",
	}

	// 调用服务器的 Register 方法并等待响应
	res, err := client.RegisterCA(context.Background(), regReq)
	fmt.Println(res)
}
