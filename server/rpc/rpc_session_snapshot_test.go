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
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
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

func TestReconnectJobStreamReconcilesMissingEnabledPipeline(t *testing.T) {
	newRPCTestEnv(t)
	const listenerID = "listener-runtime-reconnect"
	pipeline := &clientpb.Pipeline{
		Name:       "AA",
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.HTTPPipeline,
		Body: &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{
			Name:       "AA",
			ListenerId: listenerID,
			Host:       "127.0.0.1",
			Port:       8899,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	server := &Server{}
	if _, err := server.RegisterListener(context.Background(), &clientpb.RegisterListener{
		Name:      listenerID,
		Host:      "127.0.0.1",
		Pipelines: &clientpb.Pipelines{},
	}); err != nil {
		t.Fatalf("RegisterListener failed: %v", err)
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		t.Fatalf("listener not found: %v", err)
	}

	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("listener_id", listenerID))
	streamCtx, cancel := context.WithCancel(baseCtx)
	defer cancel()
	statuses := make(chan *clientpb.JobStatus, 1)
	starts := make(chan *clientpb.JobCtrl, 1)
	stream := &testJobStreamServer{
		ctx: streamCtx,
		send: func(msg *clientpb.JobCtrl) error {
			if msg.GetCtrl() == consts.CtrlPipelineStart {
				starts <- msg
				statuses <- &clientpb.JobStatus{
					ListenerId: listenerID,
					Ctrl:       msg.GetCtrl(),
					CtrlId:     msg.GetId(),
					Status:     consts.CtrlStatusSuccess,
					Job:        msg.GetJob(),
				}
			}
			return nil
		},
		recv: func() (*clientpb.JobStatus, error) {
			select {
			case msg := <-statuses:
				return msg, nil
			case <-streamCtx.Done():
				return nil, io.EOF
			}
		},
	}

	done := make(chan error, 1)
	go func() { done <- server.JobStream(stream) }()

	var start *clientpb.JobCtrl
	select {
	case start = <-starts:
		if got := start.GetJob().GetPipeline(); got.GetName() != "AA" || got.GetListenerId() != listenerID {
			t.Fatalf("recovery control pipeline = %#v, want %s/AA", got, listenerID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reconnected JobStream did not receive a recovery control")
	}

	deadline := time.NewTimer(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		if statusValue, ok := lns.CtrlJob.Load(start.GetId()); ok && statusValue != nil {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("server did not process the recovery status")
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("JobStream failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("JobStream did not stop after cancellation")
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
