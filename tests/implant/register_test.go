package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestRegister(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, []byte{1, 2, 3, 4})
	implant.Enc = true
	implant.Tls = true
	spite := &implantpb.Spite{
		TaskId: 1,
	}
	body := &implantpb.Register{
		Os: &implantpb.Os{
			Name: "windows",
		},
		Process: &implantpb.Process{
			Name: "test",
			Pid:  123,
			Uid:  "admin",
			Gid:  "root",
		},
		Timer: &implantpb.Timer{
			Interval: 10,
		},
	}
	types.BuildSpite(spite, body)
	conn := implant.MustConnect()
	implant.WriteSpite(conn, spite)
	resp, err := implant.Read(conn)
	fmt.Println(resp, err)
}
