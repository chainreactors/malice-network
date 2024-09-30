package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"

	"github.com/chainreactors/malice-network/tests/common"
	"testing"
	"time"
)

var (
	execResp = &implantpb.ExecResponse{
		Stdout:     []byte("admin"),
		Pid:        999,
		StatusCode: 0,
	}
)

func TestExec(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
	time.Sleep(1 * time.Second)
	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
	go func() {
		conn := implant.MustConnect()
		implant.WriteEmpty(conn)
		res, err := implant.Read(conn)
		fmt.Printf("res %v %v\n", res, err)
		spite := &implantpb.Spite{
			TaskId: 0,
		}
		types.BuildSpite(spite, execResp)
		err = implant.WriteSpite(conn, spite)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()
	time.Sleep(1 * time.Second)
	exec := &implantpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"},
	}
	resp, err := rpc.Call(consts.ModuleExecution, exec)
	if err != nil {
		return
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("resp %v\n", resp)

}
