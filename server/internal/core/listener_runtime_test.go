package core

import (
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
