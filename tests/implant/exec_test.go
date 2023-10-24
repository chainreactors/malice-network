package implant

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestExec(t *testing.T) {
	client := common.NewClient(common.DefaultListenerAddr, "1234")
	client.Register()
	rpc := common.NewRPC(common.DefaultGRPCAddr)
	fmt.Println(hash.Md5Hash([]byte(client.Sid)))
	go func() {
		res, err := client.Read()
		fmt.Printf("res %v %v\n", res, err)
		spite := &commonpb.Spite{
			TaskId: 1,
			End:    true,
		}
		resp := &pluginpb.ExecResponse{
			Stdout:     []byte("admin"),
			Pid:        999,
			StatusCode: 0,
		}
		types.BuildSpite(spite, resp)
		err = client.WriteSpite(spite)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	resp, err := rpc.Client.Execute(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", hash.Md5Hash([]byte(client.Sid)))), &pluginpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"}})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("resp %v\n", resp)
}
