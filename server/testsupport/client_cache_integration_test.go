//go:build integration

package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	clientcore "github.com/chainreactors/malice-network/client/core"
	servercore "github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/protobuf/proto"
)

func TestNewServerDoesNotReplayStalePipelineEventOverSnapshot(t *testing.T) {
	h := NewControlPlaneHarness(t)

	const pipelineName = "rem-cache-snapshot"
	fresh := h.NewREMPipeline(pipelineName, "tcp://127.0.0.1:19966")
	fresh.GetRem().Link = "simplex+sharepoint://fresh@example.test:443"
	h.SeedPipeline(t, fresh, true)

	stale := proto.Clone(fresh).(*clientpb.Pipeline)
	stale.GetRem().Link = ""
	servercore.EventBroker.Publish(servercore.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineSync,
		Job:       &clientpb.Job{Name: pipelineName, Pipeline: stale},
		Important: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer conn.Close()

	client, err := clientcore.NewServer(conn, h.Admin)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	got := client.Pipelines[pipelineName]
	if got == nil || got.GetRem() == nil {
		t.Fatalf("pipeline cache entry = %#v, want REM pipeline", got)
	}
	if got.GetRem().GetLink() != fresh.GetRem().GetLink() {
		t.Fatalf("pipeline link after snapshot/replay = %q, want authoritative snapshot %q",
			got.GetRem().GetLink(), fresh.GetRem().GetLink())
	}
}
