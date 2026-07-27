package listener

import (
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestHandleJobCtrlCommitsSessionSnapshotAtomically(t *testing.T) {
	oldConnections := core.Connections
	oldForwarders := core.Forwarders
	oldListenerSessions := core.ListenerSessions
	core.ResetTransientTransportState()
	t.Cleanup(func() {
		core.Connections = oldConnections
		core.Forwarders = oldForwarders
		core.ListenerSessions = oldListenerSessions
	})

	core.ListenerSessions.Add(&clientpb.Session{RawId: 1, SessionId: "old"})
	lns := &listener{Name: "snapshot-listener"}

	if status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Ctrl: core.CtrlListenerSessionSnapshotBegin,
	}); status != nil {
		t.Fatalf("snapshot begin returned status: %#v", status)
	}
	if status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Ctrl:    consts.CtrlListenerSyncSession,
		Session: &clientpb.Session{RawId: 2, SessionId: "new"},
	}); status != nil {
		t.Fatalf("snapshot session returned status: %#v", status)
	}
	if got := core.ListenerSessions.Get(1); got == nil {
		t.Fatal("old active session disappeared before snapshot commit")
	}
	if got := core.ListenerSessions.Get(2); got != nil {
		t.Fatalf("staged session visible before commit: %#v", got)
	}

	if status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Ctrl: core.CtrlListenerSessionSnapshotEnd,
	}); status != nil {
		t.Fatalf("snapshot end returned status: %#v", status)
	}
	if got := core.ListenerSessions.Get(1); got != nil {
		t.Fatalf("old session remained after snapshot commit: %#v", got)
	}
	if got := core.ListenerSessions.Get(2); got == nil || got.SessionId != "new" {
		t.Fatalf("committed session = %#v, want new", got)
	}
}

func TestReconnectSnapshotEndSchedulesActiveRuntimeReregistration(t *testing.T) {
	oldRestore := restoreListenerRuntimeRPC
	defer func() {
		restoreListenerRuntimeRPC = oldRestore
	}()

	lns := &listener{
		Name:      "runtime-restore-listener",
		pipelines: core.NewPipelines(),
		websites:  make(map[string]*Website),
	}
	lns.pipelines.Add(NewCustomPipeline(&clientpb.Pipeline{
		Name:       "active-pipeline",
		ListenerId: lns.Name,
		Enable:     true,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{
				Name:       "active-pipeline",
				ListenerId: lns.Name,
			},
		},
	}))

	restored := make(chan []*clientpb.Pipeline, 1)
	restoreListenerRuntimeRPC = func(_ *listener, pipelines, _ []*clientpb.Pipeline) error {
		restored <- pipelines
		return nil
	}

	lns.restoreOnSnapshot.Store(true)
	lns.handleJobCtrl(&clientpb.JobCtrl{Ctrl: core.CtrlListenerSessionSnapshotBegin})
	lns.handleJobCtrl(&clientpb.JobCtrl{Ctrl: core.CtrlListenerSessionSnapshotEnd})

	select {
	case pipelines := <-restored:
		if len(pipelines) != 1 || pipelines[0].Name != "active-pipeline" {
			t.Fatalf("restored pipelines = %#v, want active-pipeline", pipelines)
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot completion did not schedule runtime reregistration")
	}
}

func TestInitialSnapshotEndDoesNotScheduleRuntimeReregistration(t *testing.T) {
	oldRestore := restoreListenerRuntimeRPC
	defer func() {
		restoreListenerRuntimeRPC = oldRestore
	}()

	restored := make(chan struct{}, 1)
	restoreListenerRuntimeRPC = func(_ *listener, _, _ []*clientpb.Pipeline) error {
		restored <- struct{}{}
		return nil
	}

	lns := &listener{
		Name:      "initial-snapshot-listener",
		pipelines: core.NewPipelines(),
		websites:  make(map[string]*Website),
	}
	lns.pipelines.Add(NewCustomPipeline(&clientpb.Pipeline{
		Name:       "initializing-pipeline",
		ListenerId: lns.Name,
		Enable:     true,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{
				Name:       "initializing-pipeline",
				ListenerId: lns.Name,
			},
		},
	}))
	lns.handleJobCtrl(&clientpb.JobCtrl{Ctrl: core.CtrlListenerSessionSnapshotBegin})
	lns.handleJobCtrl(&clientpb.JobCtrl{Ctrl: core.CtrlListenerSessionSnapshotEnd})

	select {
	case <-restored:
		t.Fatal("initial snapshot unexpectedly scheduled runtime reregistration")
	case <-time.After(50 * time.Millisecond):
	}
}
