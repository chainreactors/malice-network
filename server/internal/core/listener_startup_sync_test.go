package core

import (
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestPendingListenerBlocksControlUntilSnapshotReady(t *testing.T) {
	listener := NewPendingListener("pending-listener", "127.0.0.1")
	result := make(chan uint32, 1)

	go func() {
		result <- listener.PushCtrl(&clientpb.JobCtrl{Ctrl: consts.CtrlPipelineStart})
	}()

	select {
	case id := <-result:
		t.Fatalf("PushCtrl returned %d before snapshot readiness", id)
	case <-time.After(50 * time.Millisecond):
	}
	if got := len(listener.Ctrl); got != 0 {
		t.Fatalf("queued controls before snapshot readiness = %d, want 0", got)
	}

	listener.MarkReady()
	select {
	case id := <-result:
		if id == 0 {
			t.Fatal("PushCtrl returned zero after snapshot readiness")
		}
	case <-time.After(time.Second):
		t.Fatal("PushCtrl did not unblock after snapshot readiness")
	}
}

func TestListenerSessionsCommitSnapshotAtomicallyReplacesOldEntries(t *testing.T) {
	old := ListenerSessions
	ListenerSessions = &listenerSessions{sessions: &sync.Map{}}
	t.Cleanup(func() {
		ListenerSessions = old
	})

	ListenerSessions.Add(&clientpb.Session{RawId: 1, SessionId: "old"})
	ListenerSessions.BeginSnapshot()
	ListenerSessions.AddSnapshot(&clientpb.Session{RawId: 2, SessionId: "new"})

	if got := ListenerSessions.Get(1); got == nil || got.SessionId != "old" {
		t.Fatalf("active snapshot changed before commit: %#v", got)
	}
	if got := ListenerSessions.Get(2); got != nil {
		t.Fatalf("staged session visible before commit: %#v", got)
	}

	ListenerSessions.CommitSnapshot()
	if got := ListenerSessions.Get(1); got != nil {
		t.Fatalf("stale session remained after commit: %#v", got)
	}
	if got := ListenerSessions.Get(2); got == nil || got.SessionId != "new" {
		t.Fatalf("committed session = %#v, want new snapshot entry", got)
	}
}
