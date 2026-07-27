package rpc

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/metadata"
)

func newSnapshotTestSession(t *testing.T, id, listenerID string, rawID uint32) *core.Session {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &core.Session{
		ID:         id,
		RawID:      rawID,
		ListenerID: listenerID,
		PipelineID: "snapshot-pipeline",
		CreatedAt:  time.Now(),
		Tasks:      core.NewTasks(),
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{
				Os:      &implantpb.Os{},
				Process: &implantpb.Process{},
			},
			KeyPair: &clientpb.KeyPair{
				PublicKey:  id + "-public",
				PrivateKey: id + "-private",
			},
			Argue: map[string]string{},
			Any:   map[string]interface{}{},
		},
		Ctx:    ctx,
		Cancel: cancel,
	}
}

func TestJobStreamSendsAtomicSessionSnapshotBeforePipelineStart(t *testing.T) {
	newRPCTestEnv(t)
	core.Sessions.Add(newSnapshotTestSession(t, "session-a", "listener-a", 1))
	core.Sessions.Add(newSnapshotTestSession(t, "session-b", "listener-b", 2))

	lns := core.NewPendingListener("listener-a", "127.0.0.1")
	core.Listeners.Add(lns)
	lns.Ctrl <- &clientpb.JobCtrl{
		Id:   77,
		Ctrl: consts.CtrlPipelineStart,
		Job:  &clientpb.Job{Name: "snapshot-pipeline"},
	}

	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("listener_id", lns.Name))
	streamCtx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	var (
		mu       sync.Mutex
		controls []*clientpb.JobCtrl
	)
	stream := &testJobStreamServer{
		ctx: streamCtx,
		send: func(msg *clientpb.JobCtrl) error {
			mu.Lock()
			controls = append(controls, msg)
			mu.Unlock()
			if msg.Ctrl == consts.CtrlPipelineStart {
				cancel()
			}
			return nil
		},
		recv: func() (*clientpb.JobStatus, error) {
			<-streamCtx.Done()
			return nil, io.EOF
		},
	}

	if err := (&Server{}).JobStream(stream); err != nil {
		t.Fatalf("JobStream failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(controls) != 4 {
		t.Fatalf("controls = %#v, want begin, one session, end, pipeline start", controls)
	}
	if controls[0].Ctrl != core.CtrlListenerSessionSnapshotBegin {
		t.Fatalf("first control = %q, want snapshot begin", controls[0].Ctrl)
	}
	if controls[1].Ctrl != consts.CtrlListenerSyncSession ||
		controls[1].Session == nil ||
		controls[1].Session.ListenerId != "listener-a" ||
		controls[1].Session.SessionId != "session-a" {
		t.Fatalf("session snapshot control = %#v, want listener-a/session-a", controls[1])
	}
	if controls[2].Ctrl != core.CtrlListenerSessionSnapshotEnd {
		t.Fatalf("third control = %q, want snapshot end", controls[2].Ctrl)
	}
	if controls[3].Ctrl != consts.CtrlPipelineStart {
		t.Fatalf("last control = %q, want pipeline start", controls[3].Ctrl)
	}
}

func TestActivateRecoveredSessionSyncsListenerAfterAddingRuntime(t *testing.T) {
	newRPCTestEnv(t)
	model := persistStartupRecoverySession(t, "runtime-recovery-sync", true)
	recovered, err := core.RecoverSession(model)
	if err != nil {
		t.Fatalf("RecoverSession failed: %v", err)
	}
	t.Cleanup(recovered.Cancel)

	lns := core.NewListener(model.ListenerID, "127.0.0.1")
	core.Listeners.Add(lns)
	activateRecoveredSession(recovered)

	if _, err := core.Sessions.Get(recovered.ID); err != nil {
		t.Fatalf("recovered session was not added to runtime: %v", err)
	}
	select {
	case msg := <-lns.Ctrl:
		if msg.Ctrl != consts.CtrlListenerSyncSession ||
			msg.Session == nil ||
			msg.Session.SessionId != recovered.ID {
			t.Fatalf("listener sync control = %#v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("listener did not receive recovered session sync")
	}
}
