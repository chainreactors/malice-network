package build

import (
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
)

func TestPrintArtifactsStaticDoesNotBlock(t *testing.T) {
	con := &core.Console{Log: iomclient.Log}
	restore := con.WithNonInteractiveExecution(true)
	defer restore()
	artifacts := &clientpb.Artifacts{
		Artifacts: []*clientpb.Artifact{
			{
				Id:        1,
				Name:      "TEST_ARTIFACT",
				Type:      "beacon",
				Target:    "x86_64-pc-windows-gnu",
				Source:    "saas",
				Profile:   "tcp_default",
				Pipeline:  "tcp",
				Status:    "completed",
				CreatedAt: time.Now().Unix(),
			},
		},
	}

	if err := PrintArtifacts(artifacts, con); err != nil {
		t.Fatalf("PrintArtifacts static returned error: %v", err)
	}
}
