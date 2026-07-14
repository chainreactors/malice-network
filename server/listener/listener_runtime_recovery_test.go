package listener

import (
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestStartPipelineReplacesDisabledRuntime(t *testing.T) {
	lns := &listener{
		Name:      "runtime-recovery-listener",
		pipelines: core.NewPipelines(),
	}
	existing := NewCustomPipeline(&clientpb.Pipeline{
		Name:       "runtime-recovery-pipeline",
		ListenerId: lns.Name,
		Enable:     false,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "runtime-recovery-pipeline", ListenerId: lns.Name},
		},
	})
	lns.pipelines.Add(existing)

	replacement, err := lns.startPipeline(&clientpb.Pipeline{
		Name:       existing.ID(),
		ListenerId: lns.Name,
		Enable:     true,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: existing.ID(), ListenerId: lns.Name},
		},
	})
	if err != nil {
		t.Fatalf("startPipeline failed: %v", err)
	}
	if replacement == existing {
		t.Fatal("startPipeline reused a disabled runtime")
	}
	if got := lns.pipelines.Get(existing.ID()); got != replacement {
		t.Fatalf("registered runtime = %#v, want replacement %#v", got, replacement)
	}
	if state := replacement.ToProtobuf(); state == nil || !state.Enable {
		t.Fatalf("replacement state = %#v, want enabled", state)
	}
}

func TestStartPipelineKeepsActiveRuntimeIdempotent(t *testing.T) {
	lns := &listener{
		Name:      "runtime-idempotent-listener",
		pipelines: core.NewPipelines(),
	}
	existing := NewCustomPipeline(&clientpb.Pipeline{
		Name:       "runtime-idempotent-pipeline",
		ListenerId: lns.Name,
		Enable:     true,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "runtime-idempotent-pipeline", ListenerId: lns.Name},
		},
	})
	lns.pipelines.Add(existing)

	got, err := lns.startPipeline(existing.ToProtobuf())
	if err != nil {
		t.Fatalf("startPipeline failed: %v", err)
	}
	if got != existing {
		t.Fatalf("startPipeline runtime = %#v, want existing %#v", got, existing)
	}
}

func TestStartPipelineRemovesDisabledRuntimeForward(t *testing.T) {
	lns := &listener{
		Name:      "runtime-forward-listener",
		pipelines: core.NewPipelines(),
	}
	existing := NewCustomPipeline(&clientpb.Pipeline{
		Name:       "runtime-forward-pipeline",
		ListenerId: lns.Name,
		Enable:     false,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "runtime-forward-pipeline", ListenerId: lns.Name},
		},
	})
	lns.pipelines.Add(existing)

	stream := &rollbackForwardStream{}
	forward, err := core.NewForward(&rollbackPipelineRPC{stream: stream}, existing)
	if err != nil {
		t.Fatalf("create stale forward: %v", err)
	}
	forward.ListenerId = lns.Name
	key := core.PipelineRuntimeKey(lns.Name, existing.ID())
	core.Forwarders.Add(forward)
	t.Cleanup(func() { _ = core.Forwarders.Remove(key) })

	if _, err := lns.startPipeline(&clientpb.Pipeline{
		Name:       existing.ID(),
		ListenerId: lns.Name,
		Enable:     true,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: existing.ID(), ListenerId: lns.Name},
		},
	}); err != nil {
		t.Fatalf("startPipeline failed: %v", err)
	}
	if got := core.Forwarders.Get(key); got != nil {
		t.Fatalf("stale forward remained registered: %#v", got)
	}
	if got := stream.closeCalls.Load(); got != 1 {
		t.Fatalf("stale forward CloseSend calls = %d, want 1", got)
	}
}
