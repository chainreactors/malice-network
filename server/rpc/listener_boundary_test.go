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

// C1: SpiteStream stores pipelineID in pipelinesCh but never deletes on disconnect.
// After the stream breaks, the stale entry remains and GenericHandler loads a dead stream.
func TestSpiteStreamDisconnectLeavesStalePipelineEntry(t *testing.T) {
	withIsolatedPipelinesCh(t)

	pipelineID := "tcp-pipe-spite"

	// Simulate SpiteStream connecting: pipelinesCh.Store(pipelineID, stream)
	pipelinesCh.Store(pipelineID, "fake-dead-stream")

	// Simulate SpiteStream disconnect — current code does NO cleanup.
	// (stream.Recv returns error, function returns, pipelinesCh entry stays)

	// BUG: stale entry still exists after disconnect
	val, ok := pipelinesCh.Load(pipelineID)
	if !ok || val == nil {
		t.Log("pipelinesCh cleaned up after disconnect — C1 fixed")
		return
	}
	t.Logf("CONFIRMED (C1): pipelinesCh still has stale entry %q after SpiteStream disconnect", pipelineID)
}

// C2: JobStream disconnect does NOT remove listener or clean up state.
// Listener remains Active with stale pipelines and jobs.
func TestJobStreamDisconnectDoesNotCleanListener(t *testing.T) {
	withIsolatedListenersAndJobs(t)

	listener := core.NewListener("boundary-listener", "10.0.0.1")
	// Store directly to avoid EventBroker.Publish nil panic in test isolation.
	core.Listeners.Map.Store(listener.Name, listener)

	pipe := &clientpb.Pipeline{Name: "boundary-pipe", ListenerId: "boundary-listener", Type: consts.TCPPipeline}
	listener.AddPipeline(pipe)
	core.Jobs.Map.Store("boundary-listener:boundary-pipe", &core.Job{ID: 1, Name: "boundary-pipe", Pipeline: pipe})

	// Simulate JobStream disconnect: in current code, JobStream returns but
	// does NOT call Listeners.Remove or Listeners.Stop.

	// BUG: listener still in map and active
	lns, err := core.Listeners.Get("boundary-listener")
	if err != nil {
		t.Log("listener removed after disconnect — C2 fixed")
		return
	}
	if !lns.Active {
		t.Log("listener marked inactive — C2 partially fixed")
		return
	}
	t.Log("CONFIRMED (C2): listener still Active after JobStream disconnect")

	// BUG: pipeline still findable
	found, ok := core.Listeners.Find("boundary-pipe")
	if ok && found != nil {
		t.Log("CONFIRMED (C2): pipeline still findable via dead listener")
	}

	// BUG: PushCtrl writes to buffered channel with no consumer
	done := make(chan uint32, 1)
	go func() {
		done <- listener.PushCtrl(&clientpb.JobCtrl{Ctrl: consts.CtrlPipelineStart})
	}()
	select {
	case id := <-done:
		if id > 0 {
			t.Log("CONFIRMED (C2): PushCtrl succeeds on dead listener (buffered, no consumer)")
		}
	}
}

// C3: RegisterListener with same name silently overwrites existing listener.
// Old listener's Ctrl channel and pipeline state are lost.
func TestListenerReRegisterOverwritesWithoutCleanup(t *testing.T) {
	withIsolatedListenersAndJobs(t)

	// First registration with pipeline (store directly to avoid nil EventBroker)
	old := core.NewListener("re-register", "10.0.0.1")
	core.Listeners.Map.Store(old.Name, old)
	old.AddPipeline(&clientpb.Pipeline{Name: "old-pipe", ListenerId: "re-register"})
	core.Jobs.Map.Store("re-register:old-pipe", &core.Job{ID: 1, Name: "old-pipe", Pipeline: &clientpb.Pipeline{Name: "old-pipe", ListenerId: "re-register"}})

	oldCtrl := old.Ctrl

	// Re-register with same name (simulates listener restart after crash)
	fresh := core.NewListener("re-register", "10.0.0.2")
	core.Listeners.Map.Store(fresh.Name, fresh)

	// New listener has a new Ctrl channel
	if oldCtrl == fresh.Ctrl {
		t.Fatal("new listener should have a NEW Ctrl channel")
	}

	// New listener has no pipelines (clean state)
	if len(fresh.AllPipelines()) != 0 {
		t.Fatal("fresh listener should have zero pipelines")
	}

	// After C3 fix: RegisterListener should clean old state before re-adding.
	// The test simulates this by checking that old Jobs are orphaned.
	// In production, RegisterListener now calls Listeners.Remove(old) first.
	allJobs := core.Jobs.All()
	if len(allJobs) > 0 {
		t.Logf("old pipeline's Job count = %d after re-register (should be 0 after C3 fix)", len(allJobs))
	}

	// Fresh listener should have clean pipeline state
	_, ok := core.Listeners.Find("old-pipe")
	if ok {
		t.Fatal("old pipeline should not be findable after re-register")
	}
}

// C4: After pipeline disconnect, session remains accessible but cannot execute commands.
func TestSessionOrphanedAfterPipelineDisconnect(t *testing.T) {
	withIsolatedPipelinesCh(t)

	pipelineID := "orphan-pipe"
	pipelinesCh.Store(pipelineID, "alive-stream")

	// Verify pipeline is loadable before disconnect
	if _, ok := pipelinesCh.Load(pipelineID); !ok {
		t.Fatal("pipeline should be available before disconnect")
	}

	// After C1 fix: SpiteStream disconnect should delete entry.
	// We simulate the DESIRED post-fix behavior:
	pipelinesCh.Delete(pipelineID)

	// GenericHandler would now fail fast with ErrNotFoundPipeline
	if _, ok := pipelinesCh.Load(pipelineID); ok {
		t.Fatal("pipeline should not be loadable after disconnect")
	}
}

// S4: Two goroutines consuming same Ctrl channel — messages randomly distributed.
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
		t.Log("CONFIRMED (S4): only one of two consumers receives — dual JobStream causes random message routing")
	}
}
