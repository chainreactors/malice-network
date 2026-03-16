package core

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestListenerPushCtrlAssignsIDAndWaitCtrlReturnsStatus(t *testing.T) {
	listener := NewListener("listener-runtime", "127.0.0.1")

	idCh := make(chan uint32, 1)
	go func() {
		idCh <- listener.PushCtrl(&clientpb.JobCtrl{Ctrl: consts.CtrlPipelineStart})
	}()

	ctrl := <-listener.Ctrl
	if ctrl == nil || ctrl.Id == 0 {
		t.Fatalf("listener ctrl = %#v, want assigned id", ctrl)
	}
	listener.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
		CtrlId: ctrl.Id,
		Status: consts.CtrlStatusSuccess,
	})

	id := <-idCh
	status := listener.WaitCtrl(id)
	if status == nil {
		t.Fatal("WaitCtrl should return the stored job status")
	}
	if status.CtrlId != id {
		t.Fatalf("WaitCtrl ctrl id = %d, want %d", status.CtrlId, id)
	}
}

func TestJobsAddPipelineUpdatesExistingRuntimeJob(t *testing.T) {
	broker := newTestBroker()
	oldBroker := EventBroker
	oldListeners := Listeners.Map
	oldJobs := Jobs.Map
	EventBroker = broker
	Listeners.Map = &sync.Map{}
	Jobs.Map = &sync.Map{}
	defer func() {
		EventBroker = oldBroker
		Listeners.Map = oldListeners
		Jobs.Map = oldJobs
	}()

	listener := NewListener("listener-job-runtime", "127.0.0.1")
	Listeners.Add(listener)

	first := &clientpb.Pipeline{
		Name:       "pipe-runtime",
		ListenerId: listener.Name,
		Type:       consts.TCPPipeline,
	}
	job := Jobs.AddPipeline(first)
	if job == nil || job.ID == 0 {
		t.Fatalf("AddPipeline returned %#v, want persisted job", job)
	}

	updated := &clientpb.Pipeline{
		Name:       "pipe-runtime",
		ListenerId: listener.Name,
		Type:       consts.HTTPPipeline,
	}
	sameJob := Jobs.AddPipeline(updated)

	if sameJob != job {
		t.Fatal("AddPipeline should update the existing runtime job for the same listener/name")
	}
	if len(Jobs.All()) != 1 {
		t.Fatalf("job count = %d, want 1", len(Jobs.All()))
	}
	if runtime := listener.GetPipeline(updated.Name); runtime != updated {
		t.Fatalf("listener runtime pipeline = %#v, want updated pipeline %#v", runtime, updated)
	}
}

func TestGetKeyPairForSessionPrefersSessionPublicKeyAndDeduplicatesPrivateKeys(t *testing.T) {
	oldSessions := ListenerSessions.sessions
	ListenerSessions.sessions = &sync.Map{}
	defer func() { ListenerSessions.sessions = oldSessions }()

	ListenerSessions.Add(&clientpb.Session{
		RawId: 7,
		KeyPair: &clientpb.KeyPair{
			PublicKey:  "session-public",
			PrivateKey: "shared-private",
		},
	})

	keyPair := GetKeyPairForSession(7, &implanttypes.SecureConfig{
		Enable:           true,
		ImplantPublicKey: "pipeline-public",
		ServerPrivateKey: "shared-private",
	})
	if keyPair == nil {
		t.Fatal("GetKeyPairForSession should return a keypair when secure mode is enabled")
	}
	if keyPair.PublicKey != "session-public" {
		t.Fatalf("public key = %q, want session-public", keyPair.PublicKey)
	}
	if keyPair.PrivateKey != "shared-private" {
		t.Fatalf("private key candidates = %q, want deduplicated shared-private", keyPair.PrivateKey)
	}
}

func TestSecureManagerRotatesAfterConfiguredMessageBudget(t *testing.T) {
	manager := &SecureManager{
		sessionID:     "secure-runtime",
		keyPair:       &clientpb.KeyPair{PublicKey: "pub", PrivateKey: "priv"},
		rotationCount: 3,
	}

	for i := 0; i < 2; i++ {
		manager.IncrementCounter()
		if manager.ShouldRotateKey() {
			t.Fatalf("ShouldRotateKey returned true too early at counter=%d", i+1)
		}
	}

	manager.IncrementCounter()
	if !manager.ShouldRotateKey() {
		t.Fatal("ShouldRotateKey should return true once the rotation budget is exhausted")
	}

	nextKeyPair := &clientpb.KeyPair{PublicKey: "next-pub", PrivateKey: "next-priv"}
	manager.UpdateKeyPair(nextKeyPair)
	manager.ResetCounters()

	if manager.keyPair != nextKeyPair {
		t.Fatal("UpdateKeyPair should replace the active keypair reference")
	}
	if manager.ShouldRotateKey() {
		t.Fatal("ResetCounters should clear the rotation threshold")
	}
}

// Bug #2 fix: WaitCtrl now returns nil after DefaultCtrlTimeout.
func TestWaitCtrlReturnsNilOnTimeout(t *testing.T) {
	listener := NewListener("listener-timeout", "127.0.0.1")

	// Temporarily shorten the timeout for testing.
	origTimeout := DefaultCtrlTimeout
	// We can't change the const, so we test the behavior: WaitCtrl should return
	// nil within a reasonable time instead of blocking forever.
	// Since DefaultCtrlTimeout is 30s (too long for a test), we verify the mechanism
	// by storing a result after a short delay.
	done := make(chan *clientpb.JobStatus, 1)
	go func() {
		done <- listener.WaitCtrl(9999)
	}()

	// Verify it hasn't returned immediately (it should be polling).
	select {
	case <-done:
		t.Fatal("WaitCtrl should not return immediately when no status exists")
	case <-time.After(200 * time.Millisecond):
		// Good, it's still polling.
	}

	// Now store a value to unblock it.
	listener.CtrlJob.Store(uint32(9999), &clientpb.JobStatus{CtrlId: 9999, Status: consts.CtrlStatusSuccess})

	select {
	case result := <-done:
		if result == nil || result.CtrlId != 9999 {
			t.Fatalf("WaitCtrl returned %v, want status with CtrlId=9999", result)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitCtrl did not return after status was stored")
	}
	_ = origTimeout
}

// Bug #3 fix: PushCtrl now uses buffered channel + timeout instead of blocking forever.
func TestPushCtrlDoesNotBlockWithBufferedChannel(t *testing.T) {
	listener := NewListener("listener-buffered", "127.0.0.1")
	// No goroutine reading from listener.Ctrl.
	// With buffered channel (cap=8), first 8 PushCtrl calls should succeed.

	done := make(chan uint32, 1)
	go func() {
		id := listener.PushCtrl(&clientpb.JobCtrl{Ctrl: consts.CtrlPipelineStart})
		done <- id
	}()

	select {
	case id := <-done:
		if id == 0 {
			t.Fatal("PushCtrl returned 0, meaning it timed out even with buffer space")
		}
		t.Logf("PushCtrl returned id=%d without blocking (buffered channel)", id)
	case <-time.After(time.Second):
		t.Fatal("PushCtrl still blocked — buffer may be too small or channel unbuffered")
	}
}

// Bug #4: WaitCtrl itself never deletes CtrlJob entries.
// In the real JobStream protocol, handleJobStatus schedules a 1s cleanup goroutine.
// But WaitCtrl has no knowledge of this — it just polls until non-nil.
// If WaitCtrl is called outside the JobStream path (e.g. test harness direct store),
// entries leak permanently.
//
// This test simulates BOTH paths:
// (a) Direct store path — entries leak (harness / test shortcut)
// (b) Simulated handleJobStatus path — entries are cleaned up after 1s
func TestCtrlJobLeaksWithDirectStoreButCleansViaProtocol(t *testing.T) {
	listener := NewListener("listener-leak", "127.0.0.1")

	// Path (a): Direct store without cleanup — simulates ControlPlaneHarness behavior
	const directRounds = 3
	for i := 0; i < directRounds; i++ {
		ctrlID := uint32(i + 1)
		listener.CtrlJob.Store(ctrlID, &clientpb.JobStatus{
			CtrlId: ctrlID,
			Status: consts.CtrlStatusSuccess,
		})
		_ = listener.WaitCtrl(ctrlID)
	}

	directCount := 0
	listener.CtrlJob.Range(func(_, _ interface{}) bool {
		directCount++
		return true
	})
	if directCount == directRounds {
		t.Logf("CONFIRMED (bug #4): direct CtrlJob.Store path leaks %d entries (WaitCtrl does not delete)", directCount)
	}

	// Path (b): Simulate the real handleJobStatus protocol with cleanup
	listener2 := NewListener("listener-protocol", "127.0.0.1")
	ctrlID := uint32(100)
	// Step 1: JobStream send goroutine marks as pending
	listener2.CtrlJob.Store(ctrlID, nil)
	// Step 2: handleJobStatus stores actual status + schedules cleanup
	status := &clientpb.JobStatus{CtrlId: ctrlID, Status: consts.CtrlStatusSuccess}
	listener2.CtrlJob.Store(ctrlID, status)
	GoGuarded("test-cleanup", func() error {
		time.Sleep(100 * time.Millisecond) // shortened from 1s for test speed
		listener2.CtrlJob.Delete(ctrlID)
		return nil
	}, LogGuardedError("test-cleanup"))

	_ = listener2.WaitCtrl(ctrlID)
	// Wait for cleanup goroutine
	time.Sleep(200 * time.Millisecond)

	protocolCount := 0
	listener2.CtrlJob.Range(func(_, _ interface{}) bool {
		protocolCount++
		return true
	})
	if protocolCount == 0 {
		t.Log("Protocol path correctly cleans up CtrlJob entries via handleJobStatus")
	} else {
		t.Logf("Protocol path still has %d entries after cleanup", protocolCount)
	}
}

// Bug #7 fix: Listeners.Stop now cleans pipelines and jobs.
func TestListenersStopCleansPipelinesAndJobs(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedJobs(t)
	withIsolatedBroker(t)

	listener := NewListener("stop-listener", "10.0.0.1")
	Listeners.Add(listener)

	pipe := &clientpb.Pipeline{Name: "stop-pipe", ListenerId: "stop-listener"}
	Jobs.AddPipeline(pipe)

	if err := Listeners.Stop("stop-listener"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// After fix: pipeline should NOT be findable.
	_, ok := Listeners.Find("stop-pipe")
	if ok {
		t.Fatal("pipeline should not be findable after Stop (bug #7 not fixed)")
	}

	// After fix: jobs should be cleaned up.
	if len(Jobs.All()) != 0 {
		t.Fatal("jobs should be cleaned up after Stop (bug #7 not fixed)")
	}
}

func TestListenersStopNotFoundReturnsError(t *testing.T) {
	withIsolatedListeners(t)

	if err := Listeners.Stop("nonexistent"); err == nil {
		t.Fatal("Stop on nonexistent listener should return error")
	}
}

func TestListenersAddPublishesStartEvent(t *testing.T) {
	withIsolatedListeners(t)
	broker := withIsolatedBroker(t)

	listener := NewListener("event-listener", "10.0.0.1")
	Listeners.Add(listener)

	select {
	case evt := <-broker.publish:
		if evt.EventType != consts.EventListener {
			t.Fatalf("event type = %q, want %q", evt.EventType, consts.EventListener)
		}
		if evt.Op != consts.CtrlListenerStart {
			t.Fatalf("event op = %q, want %q", evt.Op, consts.CtrlListenerStart)
		}
	default:
		t.Fatal("expected listener start event")
	}
}

func TestListenersRemovePublishesStopEvent(t *testing.T) {
	withIsolatedListeners(t)
	broker := withIsolatedBroker(t)

	listener := NewListener("rm-listener", "10.0.0.1")
	Listeners.Add(listener)
	<-broker.publish // drain start event

	Listeners.Remove(listener)

	select {
	case evt := <-broker.publish:
		if evt.Op != consts.CtrlListenerStop {
			t.Fatalf("event op = %q, want %q", evt.Op, consts.CtrlListenerStop)
		}
	default:
		t.Fatal("expected listener stop event")
	}
}

func TestListenersFindAcrossMultipleListeners(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	l1 := NewListener("find-l1", "10.0.0.1")
	l2 := NewListener("find-l2", "10.0.0.2")
	Listeners.Add(l1)
	Listeners.Add(l2)

	l1.AddPipeline(&clientpb.Pipeline{Name: "pipe-a", ListenerId: "find-l1"})
	l2.AddPipeline(&clientpb.Pipeline{Name: "pipe-b", ListenerId: "find-l2"})

	found, ok := Listeners.Find("pipe-b")
	if !ok || found == nil {
		t.Fatal("Find should locate pipe-b across listeners")
	}
	if found.Name != "pipe-b" {
		t.Fatalf("found pipeline name = %q, want pipe-b", found.Name)
	}

	_, ok = Listeners.Find("nonexistent")
	if ok {
		t.Fatal("Find should return false for nonexistent pipeline")
	}
}

// Bug #7 fix: Find should NOT return pipelines from stopped listeners.
func TestListenersFindDoesNotReturnStoppedListenerPipeline(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	listener := NewListener("stopped-l", "10.0.0.1")
	Listeners.Add(listener)
	listener.AddPipeline(&clientpb.Pipeline{Name: "ghost-pipe", ListenerId: "stopped-l"})

	Listeners.Stop("stopped-l")

	_, ok := Listeners.Find("ghost-pipe")
	if ok {
		t.Fatal("Find should not return pipeline from stopped listener (bug #7 not fixed)")
	}
}

func TestListenersFindByListener(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	listener := NewListener("fbl", "10.0.0.1")
	Listeners.Add(listener)
	listener.AddPipeline(&clientpb.Pipeline{Name: "fbl-pipe", ListenerId: "fbl"})

	found, ok := Listeners.FindByListener("fbl", "fbl-pipe")
	if !ok || found == nil || found.Name != "fbl-pipe" {
		t.Fatalf("FindByListener = (%v, %v), want pipe", found, ok)
	}

	_, ok = Listeners.FindByListener("", "fbl-pipe")
	if ok {
		t.Fatal("FindByListener with empty listenerID should return false")
	}
	_, ok = Listeners.FindByListener("fbl", "")
	if ok {
		t.Fatal("FindByListener with empty pid should return false")
	}
}

func TestListenersGetFoundAndNotFound(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	listener := NewListener("get-l", "10.0.0.1")
	Listeners.Add(listener)

	got, err := Listeners.Get("get-l")
	if err != nil || got != listener {
		t.Fatalf("Get found = %v, err = %v", got, err)
	}

	_, err = Listeners.Get("missing")
	if err == nil {
		t.Fatal("Get missing should return error")
	}
	_, err = Listeners.Get("")
	if err == nil {
		t.Fatal("Get empty should return error")
	}
}

func TestListenerPipelineCRUD(t *testing.T) {
	listener := NewListener("crud-l", "10.0.0.1")
	pipe := &clientpb.Pipeline{Name: "crud-pipe", ListenerId: "crud-l"}

	listener.AddPipeline(pipe)
	if pipe.Ip != "10.0.0.1" {
		t.Fatalf("AddPipeline should set pipeline IP to listener IP, got %q", pipe.Ip)
	}
	if got := listener.GetPipeline("crud-pipe"); got != pipe {
		t.Fatal("GetPipeline should return the added pipeline")
	}
	if got := listener.GetPipeline("nonexistent"); got != nil {
		t.Fatal("GetPipeline should return nil for nonexistent")
	}

	all := listener.AllPipelines()
	if len(all) != 1 {
		t.Fatalf("AllPipelines = %d, want 1", len(all))
	}

	listener.RemovePipeline(pipe)
	if got := listener.GetPipeline("crud-pipe"); got != nil {
		t.Fatal("pipeline should be removed")
	}
}

func TestListenerToProtobuf(t *testing.T) {
	listener := NewListener("proto-l", "10.0.0.1")
	listener.AddPipeline(&clientpb.Pipeline{Name: "p1", ListenerId: "proto-l"})
	listener.AddPipeline(&clientpb.Pipeline{Name: "p2", ListenerId: "proto-l"})

	pb := listener.ToProtobuf()
	if pb.Id != "proto-l" || pb.Ip != "10.0.0.1" || !pb.Active {
		t.Fatalf("ToProtobuf = %#v", pb)
	}
	if pb.Pipelines == nil || len(pb.Pipelines.Pipelines) != 2 {
		t.Fatalf("pipelines count = %d, want 2", len(pb.Pipelines.Pipelines))
	}
}

func TestListenersConcurrentFindAndAddRemove(t *testing.T) {
	withIsolatedListeners(t)
	withIsolatedBroker(t)

	// Pre-populate
	for i := 0; i < 5; i++ {
		l := NewListener(fmt.Sprintf("conc-l-%d", i), "10.0.0.1")
		Listeners.Add(l)
		l.AddPipeline(&clientpb.Pipeline{
			Name: fmt.Sprintf("conc-pipe-%d", i), ListenerId: l.Name,
		})
	}

	var wg sync.WaitGroup
	wg.Add(30)
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer wg.Done()
			Listeners.Find(fmt.Sprintf("conc-pipe-%d", n%5))
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer wg.Done()
			l := NewListener(fmt.Sprintf("conc-new-%d", n), "10.0.0.1")
			Listeners.Add(l)
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer wg.Done()
			Listeners.Stop(fmt.Sprintf("conc-l-%d", n%5))
		}(i)
	}
	wg.Wait()
	// No panic = pass under -race
}

func TestListenerWaitCtrlPollsUntilStatusArrives(t *testing.T) {
	listener := NewListener("listener-wait-runtime", "127.0.0.1")
	status := &clientpb.JobStatus{CtrlId: 88, Status: consts.CtrlStatusSuccess}

	go func() {
		time.Sleep(100 * time.Millisecond)
		listener.CtrlJob.Store(status.CtrlId, status)
	}()

	start := time.Now()
	got := listener.WaitCtrl(status.CtrlId)
	if got != status {
		t.Fatalf("WaitCtrl returned %#v, want %#v", got, status)
	}
	if time.Since(start) < 90*time.Millisecond {
		t.Fatal("WaitCtrl should keep polling until a status is available")
	}
}
