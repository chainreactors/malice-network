package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/tests/common"
	"testing"
	"time"
)

func TestUpload(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(1 * time.Second)
	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
	go func() {

		upload, err := implant.Request(nil)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		taskid := upload.(*implantpb.Spites).Spites[0].TaskId
		fmt.Printf("res %v %v\n", upload, err)
		time.Sleep(1 * time.Second)

		implant.Request(implant.BuildCommonSpite(common.StatusSpite, taskid))
		time.Sleep(1 * time.Second)
		block, err := implant.Request(nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		implant.Request(implant.BuildCommonSpite(common.AckSpite, taskid))
		fmt.Println(block)
	}()
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call("upload", &implantpb.UploadRequest{
		Name:   "test.txt",
		Target: ".",
		Priv:   0o644,
		Data:   make([]byte, 1000),
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("resp %v\n", resp)
	select {}
}

func TestDownload(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(1 * time.Second)
	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
	go func() {
		download, err := implant.Request(nil)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		taskid := download.(*implantpb.Spites).Spites[0].TaskId
		fmt.Printf("res %v %v\n", download, err)
		time.Sleep(1 * time.Second)

		_, err = implant.Request(implant.BuildCommonSpite(common.StatusSpite, taskid))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		time.Sleep(1 * time.Second)

		block, _ := implant.BuildTaskSpite(&implantpb.Block{
			BlockId: 0,
			Content: make([]byte, 100),
		}, taskid)
		ack, err := implant.Request(block)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(ack)
	}()
	time.Sleep(1)
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call("download", &implantpb.DownloadRequest{
		Name: "test",
		Path: "/test.txt",
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("resp %v\n", resp)
	select {}
}
