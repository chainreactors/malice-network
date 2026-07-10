package pivot

import (
	"strings"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
)

func TestGetRemLinkFallsBackToScopedPipelineCache(t *testing.T) {
	const (
		listenerID   = "listener-main"
		pipelineName = "rem_graph_api_01"
		wantLink     = "simplex+sharepoint://user:pass@example.test:443"
	)

	staleBare := &clientpb.Pipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{Rem: &clientpb.REM{
			Name:       pipelineName,
			ListenerId: listenerID,
		}},
	}
	freshScoped := &clientpb.Pipeline{
		Name:       pipelineName,
		ListenerId: listenerID,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{Rem: &clientpb.REM{
			Name:       pipelineName,
			ListenerId: listenerID,
			Link:       wantLink,
		}},
	}

	con := &core.Console{Server: &core.Server{ServerState: &iomclient.ServerState{
		Pipelines: map[string]*clientpb.Pipeline{
			pipelineName:                            staleBare,
			iomclient.PipelineCacheKey(freshScoped): freshScoped,
		},
	}}}

	got, err := GetRemLink(con, pipelineName)
	if err != nil {
		t.Fatalf("GetRemLink returned error: %v", err)
	}
	if got != wantLink {
		t.Fatalf("GetRemLink = %q, want %q", got, wantLink)
	}
}

func TestGetRemLinkReportsNoLinkWhenOnlyCachedPipelineHasEmptyLink(t *testing.T) {
	const pipelineName = "rem_graph_api_01"

	con := &core.Console{Server: &core.Server{ServerState: &iomclient.ServerState{
		Pipelines: map[string]*clientpb.Pipeline{
			pipelineName: {
				Name: pipelineName,
				Type: consts.RemPipeline,
				Body: &clientpb.Pipeline_Rem{Rem: &clientpb.REM{Name: pipelineName}},
			},
		},
	}}}

	_, err := GetRemLink(con, pipelineName)
	if err == nil || !strings.Contains(err.Error(), "has no link address") {
		t.Fatalf("GetRemLink error = %v, want no link address error", err)
	}
}
