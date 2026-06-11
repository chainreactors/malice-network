package build

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
)

func TestResolveProfilePipelineFlagsUsesScopedCacheKey(t *testing.T) {
	con := &core.Console{
		Server: &core.Server{
			ServerState: &iomclient.ServerState{
				Pipelines: map[string]*clientpb.Pipeline{},
			},
		},
	}
	con.Pipelines["listener-a:pipe-a"] = &clientpb.Pipeline{
		Name:       "pipe-a",
		ListenerId: "listener-a",
	}

	pipelineID, listenerID := resolveProfilePipelineFlags(con, "listener-a:pipe-a", "")
	if pipelineID != "pipe-a" || listenerID != "listener-a" {
		t.Fatalf("resolved pipeline = %q/%q, want pipe-a/listener-a", pipelineID, listenerID)
	}
	if scoped := scopedProfilePipelineID(pipelineID, listenerID); scoped != "listener-a:pipe-a" {
		t.Fatalf("scoped pipeline = %q, want listener-a:pipe-a", scoped)
	}
}

func TestResolveProfilePipelineFlagsParsesScopedValueWithoutCache(t *testing.T) {
	pipelineID, listenerID := resolveProfilePipelineFlags(nil, "listener-b:pipe-b", "")
	if pipelineID != "pipe-b" || listenerID != "listener-b" {
		t.Fatalf("resolved pipeline = %q/%q, want pipe-b/listener-b", pipelineID, listenerID)
	}
}
