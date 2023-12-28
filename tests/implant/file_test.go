package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"net"
	"testing"
	"time"
)

func TestUpload(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(1 * time.Second)
	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
	go func() {
		var err error
		var conn net.Conn
		conn = implant.MustConnect()
		implant.WriteEmpty(conn)
		upload, err := implant.Read(conn)
		conn.Close()
		fmt.Printf("res %v %v\n", upload, err)
		time.Sleep(1 * time.Second)
		conn = implant.MustConnect()
		implant.WriteAsync(conn, upload.(*commonpb.Spites).Spites[0].TaskId)
		conn.Close()

		time.Sleep(1 * time.Second)
		conn = implant.MustConnect()
		implant.WriteEmpty(conn)
		block, err := implant.Read(conn)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(block)
	}()
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call("upload", &pluginpb.UploadRequest{
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
		var err error
		var conn net.Conn
		conn = implant.MustConnect()
		implant.WriteEmpty(conn)
		download, err := implant.Read(conn)
		conn.Close()
		fmt.Printf("res %v %v\n", download, err)
		time.Sleep(1 * time.Second)
		conn = implant.MustConnect()
		err = implant.WriteAsync(conn, download.(*commonpb.Spites).Spites[0].TaskId)
		if err != nil {
			fmt.Println(err)
			return
		}
		conn.Close()
		time.Sleep(1 * time.Second)
		conn = implant.MustConnect()
		err = implant.Write(conn, &commonpb.Block{
			BlockId: 0,
			Content: make([]byte, 100),
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}()
	time.Sleep(1)
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	resp, err := rpc.Call("download", &pluginpb.DownloadRequest{
		Name: "test",
		Path: "/test.txt",
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("resp %v\n", resp)
	select {}
}
