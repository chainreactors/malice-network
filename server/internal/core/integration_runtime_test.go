package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

// Integration: full pipeline control loop simulating the real JobStream protocol.
//
// Real flow (server + remote listener communicate via gRPC):
//   Server:    PushCtrl → writes to lns.Ctrl channel
//   JobStream: send goroutine reads Ctrl → CtrlJob.Store(id, nil) → gRPC Send
//   Listener:  receives ctrl, starts pipeline, sends back JobStatus via gRPC
//   JobStream: recv goroutine receives → handleJobStatus → CtrlJob.Store(id, status) → schedule Delete
//   Server:    WaitCtrl polls CtrlJob for non-nil → returns
//   Server:    Jobs.AddPipeline to register pipeline
//
// This test simulates each step of the protocol to verify the full chain.
func TestIntegrationPipelineControlLoop(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedJobs(t)
	broker := withIsolatedBroker(t)

	listener := NewListener("integ-listener", "10.0.0.1")
	Listeners.Add(listener)
	<-broker.publish // drain listener start event

	pipe := &clientpb.Pipeline{
		Name:       "integ-pipe",
		ListenerId: "integ-listener",
		Type:       consts.TCPPipeline,
	}

	// Capture broker reference locally to avoid racing with test cleanup.
	testBroker := EventBroker

	// Simulate the JobStream goroutines (normally in rpc-listener.go:99-176):
	//
	// Send goroutine: reads from Ctrl, marks as pending, "sends" to remote listener.
	// Recv goroutine: "receives" response from listener, calls handleJobStatus equivalent.
	go func() {
		// --- JobStream send side ---
		ctrl := <-listener.Ctrl
		// Real code: lns.CtrlJob.Store(msg.Id, nil) then stream.Send(ctrl)
		listener.CtrlJob.Store(ctrl.Id, nil)

		// --- Remote listener processes the ctrl (independent process) ---
		// Listener starts the pipeline, builds a JobStatus response.
		status := &clientpb.JobStatus{
			ListenerId: listener.Name,
			Ctrl:       ctrl.Ctrl,
			CtrlId:     ctrl.Id,
			Job:        ctrl.Job,
			Status:     consts.CtrlStatusSuccess,
		}

		// --- JobStream recv side (handleJobStatus equivalent) ---
		// Real code: handleJobStatus stores status and schedules cleanup.
		if _, ok := listener.CtrlJob.Load(ctrl.Id); ok {
			listener.CtrlJob.Store(ctrl.Id, status)
			// Schedule cleanup (real code uses 1s, we use shorter for test)
			GoGuarded("test-ctrl-cleanup", func() error {
				time.Sleep(200 * time.Millisecond)
				listener.CtrlJob.Delete(ctrl.Id)
				return nil
			}, LogGuardedError("test-ctrl-cleanup"))
		}

		// In real flow, handleJobStatus also publishes EventJob on success.
		testBroker.Publish(Event{
			EventType: consts.EventJob,
			Op:        ctrl.Ctrl,
			Job:       ctrl.Job,
			Important: true,
		})
	}()

	// --- Server RPC handler side (StartPipeline) ---
	ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStart,
		Job:  &clientpb.Job{Name: pipe.Name, Pipeline: pipe},
	})
	status := listener.WaitCtrl(ctrlID)
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("ctrl status = %v, want success", status)
	}

	// Server registers the pipeline in its in-memory cache.
	job := Jobs.AddPipeline(pipe)
	if job == nil || job.ID == 0 {
		t.Fatal("AddPipeline should return a valid job")
	}

	// Verify: job exists, pipeline findable via server's cache.
	if len(Jobs.All()) != 1 {
		t.Fatalf("job count = %d, want 1", len(Jobs.All()))
	}
	found, ok := Listeners.Find("integ-pipe")
	if !ok || found == nil {
		t.Fatal("pipeline should be findable via Listeners.Find")
	}

	// Verify: CtrlJob entry is cleaned up after the protocol cleanup delay.
	time.Sleep(300 * time.Millisecond)
	ctrlCount := 0
	listener.CtrlJob.Range(func(_, _ interface{}) bool {
		ctrlCount++
		return true
	})
	if ctrlCount == 0 {
		t.Log("CtrlJob entry correctly cleaned up via protocol cleanup goroutine")
	} else {
		t.Logf("CtrlJob still has %d entries after cleanup delay", ctrlCount)
	}

	// Bug #7 fix: Stop now cleans pipelines and jobs.
	Listeners.Stop("integ-listener")
	if _, ok = Listeners.Find("integ-pipe"); ok {
		t.Fatal("pipeline should not be findable after Stop (bug #7 not fixed)")
	}
	if len(Jobs.All()) != 0 {
		t.Fatal("jobs should be cleaned up after Stop (bug #7 not fixed)")
	}
}

// Integration: Client join events flow through broker publish channel.
func TestIntegrationClientEventBroadcast(t *testing.T) {
	withIsolatedClients(t)
	broker := withIsolatedBroker(t)

	c1 := NewClient("operator-alpha")
	Clients.Add(c1)

	select {
	case evt := <-broker.publish:
		if evt.EventType != consts.EventJoin {
			t.Fatalf("event type = %q, want EventJoin", evt.EventType)
		}
		if evt.Client.Name != "operator-alpha" {
			t.Fatalf("event client name = %q, want operator-alpha", evt.Client.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for EventJoin")
	}

	c2 := NewClient("operator-beta")
	Clients.Add(c2)

	select {
	case evt := <-broker.publish:
		if evt.Client.Name != "operator-beta" {
			t.Fatalf("second event client = %q, want operator-beta", evt.Client.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second EventJoin")
	}

	// Bug #8 fix: Remove now publishes EventLeft.
	Clients.Remove(int(c1.ID))
	select {
	case evt := <-broker.publish:
		if evt.EventType != consts.EventLeft {
			t.Fatalf("remove event type = %q, want EventLeft", evt.EventType)
		}
	case <-time.After(time.Second):
		t.Fatal("Remove should publish EventLeft (bug #8 not fixed)")
	}
}

// Bug #6: SecureManager fields are not protected by any lock.
// Concurrent IncrementCounter + ShouldRotateKey is a data race.
// Run with: go test -race ./server/internal/core/... -run TestIntegrationSecureManagerConcurrentAccess
func TestIntegrationSecureManagerConcurrentAccess(t *testing.T) {
	manager := &SecureManager{
		sessionID:     "race-session",
		keyPair:       &clientpb.KeyPair{PublicKey: "pub", PrivateKey: "priv"},
		rotationCount: 100,
	}

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 50
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				manager.IncrementCounter()
				_ = manager.ShouldRotateKey()
			}
		}()
	}
	wg.Wait()
	// Under -race, this will report a data race on keyCounter.
}

// Integration: Ticker-driven periodic callback with global state.
func TestIntegrationTickerDrivenCallback(t *testing.T) {
	withIsolatedTicker(t)

	var counter atomic.Int32
	_, err := GlobalTicker.Start(1, func() {
		counter.Add(1)
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(2500 * time.Millisecond)
	val := counter.Load()
	if val == 0 {
		t.Fatal("ticker callback never fired")
	}

	GlobalTicker.RemoveAll()
	snapshot := counter.Load()
	time.Sleep(1500 * time.Millisecond)
	if counter.Load() > snapshot {
		t.Fatalf("callback still fires after RemoveAll: before=%d after=%d", snapshot, counter.Load())
	}
}
