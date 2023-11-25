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

func TestUpload(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
	go func() {
		conn := implant.MustConnect()
		res, err := implant.Read(conn)
		fmt.Printf("res %v %v\n", res, err)
		spite := &commonpb.Spite{
			TaskId: 1,
		}
		resp := &pluginpb.UploadRequest{
			Name:   "test.exe",
			Target: ".",
			Priv:   0644,
			Data:   make([]byte, 1000),
		}
		types.BuildSpite(spite, resp)
		err = implant.WriteSpite(conn, spite)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	resp, err := rpc.Client.Upload(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", hash.Md5Hash([]byte(implant.Sid)))), &pluginpb.UploadRequest{
		Name:   "test.exe",
		Target: ".",
		Priv:   0644,
		Data:   make([]byte, 1000),
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("resp %v\n", resp)
}
