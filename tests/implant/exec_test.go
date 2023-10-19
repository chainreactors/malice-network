package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestExec(t *testing.T) {
	client := common.NewClient(common.DefaultListenerAddr)

	spite := &commonpb.Spite{
		TaskId: 2,
	}

	exec := &pluginpb.ExecRequest{
		Path: "/bin/bash",
		Args: []string{"whoami"},
	}

	client.BuildSpite(spite, exec)

	resp := client.RequestSpite(spite)
	fmt.Println(resp)
}
