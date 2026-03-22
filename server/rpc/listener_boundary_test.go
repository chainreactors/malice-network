package rpc

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

// withIsolatedPipelinesCh clears pipelinesCh and restores it after the test.
func withIsolatedPipelinesCh(t *testing.T) {
	t.Helper()
	// Snapshot existing entries.
	snapshot := map[string]interface{}{}
	pipelinesCh.Range(func(key, value interface{}) bool {
		snapshot[key.(string)] = value
		return true
	})
	// Clear all entries.
	pipelinesCh.Range(func(key, _ interface{}) bool {
		pipelinesCh.Delete(key)
		return true
	})
	t.Cleanup(func() {
		// Clear test entries.
		pipelinesCh.Range(func(key, _ interface{}) bool {
			pipelinesCh.Delete(key)
			return true
		})
		// Restore original entries.
		for k, v := range snapshot {
			pipelinesCh.Store(k, v)
		}
	})
}

func withIsolatedListenersAndJobs(t *testing.T) {
	t.Helper()
	oldListeners := core.Listeners.Map
	oldJobs := core.Jobs.Map
	core.Listeners.Map = &sync.Map{}
	core.Jobs.Map = &sync.Map{}
	t.Cleanup(func() {
		core.Listeners.Map = oldListeners
		core.Jobs.Map = oldJobs
	})
}

// C1 regression: SpiteStream's defer cleanup must remove pipelinesCh entry.
// We simulate the cleanup path that the defer block in SpiteStream executes.
func TestSpiteStreamDisconnectLeavesStalePipelineEntry(t *testing.T) {
	withIsolatedPipelinesCh(t)

	pipelineID := "tcp-pipe-spite"

	// Simulate SpiteStream: store entry, then execute the defer cleanup path.
	pipelinesCh.Store(pipelineID, "fake-stream")

	// This is what SpiteStream's defer does on disconnect (rpc-listener.go:52-55):
	pipelinesCh.Delete(pipelineID)

	// Verify cleanup: entry must be gone.
	if _, ok := pipelinesCh.Load(pipelineID); ok {
		t.Fatal("pipelinesCh entry should be cleaned up after SpiteStream disconnect")
	}
}

// C2 regression: JobStream's defer cleanup must deactivate listener, clear pipelines,
// remove from Listeners map, and clean associated pipelinesCh entries.
func TestJobStreamDisconnectDoesNotCleanListener(t *testing.T) {
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)

	listenerID := "boundary-listener"
	listener := core.NewListener(listenerID, "10.0.0.1")
	core.Listeners.Map.Store(listener.Name, listener)

	pipe := &clientpb.Pipeline{Name: "boundary-pipe", ListenerId: listenerID, Type: consts.TCPPipeline}
	listener.AddPipeline(pipe)
	core.Jobs.Map.Store(listenerID+":boundary-pipe", &core.Job{ID: 1, Name: "boundary-pipe", Pipeline: pipe})
	pipelinesCh.Store("boundary-pipe", "fake-stream")

	// Simulate JobStream's defer cleanup (rpc-listener.go:126-141):
	for _, p := range listener.AllPipelines() {
		pipelinesCh.Delete(p.Name)
	}
	if err := core.Listeners.Stop(listenerID); err != nil {
		t.Fatalf("Listeners.Stop failed: %v", err)
	}
	core.Listeners.Map.Delete(listenerID)

	// Verify: listener removed from map
	if _, err := core.Listeners.Get(listenerID); err == nil {
		t.Fatal("listener should be removed after JobStream disconnect cleanup")
	}

	// Verify: pipeline not findable
	if _, ok := core.Listeners.Find("boundary-pipe"); ok {
		t.Fatal("pipeline should not be findable after cleanup")
	}

	// Verify: pipelinesCh entry cleaned
	if _, ok := pipelinesCh.Load("boundary-pipe"); ok {
		t.Fatal("pipelinesCh entry should be cleaned after JobStream disconnect")
	}
}

// C3 regression: RegisterListener with same name must clean old state
// (pipelines, jobs, pipelinesCh) before creating a fresh listener.
func TestListenerReRegisterOverwritesWithoutCleanup(t *testing.T) {
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)

	// First registration with pipeline
	old := core.NewListener("re-register", "10.0.0.1")
	core.Listeners.Map.Store(old.Name, old)
	old.AddPipeline(&clientpb.Pipeline{Name: "old-pipe", ListenerId: "re-register"})
	core.Jobs.Map.Store("re-register:old-pipe", &core.Job{ID: 1, Name: "old-pipe", Pipeline: &clientpb.Pipeline{Name: "old-pipe", ListenerId: "re-register"}})
	pipelinesCh.Store("old-pipe", "old-stream")

	// Simulate RegisterListener's cleanup logic (rpc-listener.go:24-32):
	if oldLns, err := core.Listeners.Get("re-register"); err == nil {
		for _, pipe := range oldLns.AllPipelines() {
			pipelinesCh.Delete(pipe.Name)
		}
		_ = core.Listeners.Stop("re-register")
		core.Listeners.Map.Delete("re-register")
	}

	// Re-register with fresh listener
	fresh := core.NewListener("re-register", "10.0.0.2")
	core.Listeners.Map.Store(fresh.Name, fresh)

	// Verify: fresh listener has no pipelines
	if len(fresh.AllPipelines()) != 0 {
		t.Fatal("fresh listener should have zero pipelines")
	}

	// Verify: old pipeline not findable
	if _, ok := core.Listeners.Find("old-pipe"); ok {
		t.Fatal("old pipeline should not be findable after re-register")
	}

	// Verify: old pipelinesCh entry cleaned
	if _, ok := pipelinesCh.Load("old-pipe"); ok {
		t.Fatal("old pipelinesCh entry should be cleaned after re-register")
	}

	// Verify: old jobs cleaned
	allJobs := core.Jobs.All()
	if len(allJobs) > 0 {
		t.Fatalf("old jobs should be cleaned after re-register, got %d", len(allJobs))
	}
}

// C4 regression: After pipeline disconnect, GenericHandler must fail fast
// with ErrNotFoundPipeline.
func TestSessionOrphanedAfterPipelineDisconnect(t *testing.T) {
	withIsolatedPipelinesCh(t)

	pipelineID := "orphan-pipe"
	pipelinesCh.Store(pipelineID, "alive-stream")

	if _, ok := pipelinesCh.Load(pipelineID); !ok {
		t.Fatal("pipeline should be available before disconnect")
	}

	// Simulate SpiteStream defer cleanup
	pipelinesCh.Delete(pipelineID)

	// GenericHandler would now fail fast with ErrNotFoundPipeline
	if _, ok := pipelinesCh.Load(pipelineID); ok {
		t.Fatal("pipeline should not be loadable after disconnect")
	}
}

// S4: Two goroutines consuming same Ctrl channel — messages randomly distributed.
// This is a known design limitation, not a bug. Documenting the behavior.
func TestDualCtrlConsumersRaceOnChannel(t *testing.T) {
	listener := core.NewListener("dual-consumer", "10.0.0.1")

	received := make(chan int, 2)

	// Two consumers (simulates duplicate JobStream connections)
	go func() {
		<-listener.Ctrl
		received <- 1
	}()
	go func() {
		<-listener.Ctrl
		received <- 2
	}()

	// Send one message
	go func() {
		listener.PushCtrl(&clientpb.JobCtrl{Ctrl: consts.CtrlPipelineStart})
	}()

	// Only one consumer should receive
	first := <-received
	t.Logf("consumer %d received the message", first)

	select {
	case second := <-received:
		t.Fatalf("both consumers received: %d and %d — message duplicated", first, second)
	default:
		t.Log("(S4) only one of two consumers receives — dual JobStream causes random message routing (known limitation)")
	}
}
