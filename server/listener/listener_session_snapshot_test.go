package listener

import (
	"testing"

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
