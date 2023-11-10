package client

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestExec(t *testing.T) {
	rpc := common.NewRPC(common.DefaultGRPCAddr)
	resp, err := rpc.Client.Execute(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", hash.Md5Hash([]byte{1, 2, 3, 4}))), &pluginpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"}})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(resp)
}
