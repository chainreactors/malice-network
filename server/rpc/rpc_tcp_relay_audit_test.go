package rpc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

// clearTcpRelayManagers removes all managers so tests do not leak across cases.
func clearTcpRelayManagers(t *testing.T) {
	t.Helper()
	tcpRelayManagers.Range(func(key, _ any) bool {
		tcpRelayManagers.Delete(key)
		return true
	})
	t.Cleanup(func() {
		tcpRelayManagers.Range(func(key, _ any) bool {
			tcpRelayManagers.Delete(key)
			return true
		})
	})
}

func seedTcpRelayLiveSession(t *testing.T, sessionID string, taskID uint32) (*core.Session, *TcpRelayManager, *core.Task) {
	t.Helper()
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, sessionID, sessionID+"-pipe", true)
	task := sess.NewTask("tcp_relay", -1)
	// Align task id if needed for assertions (NewTask assigns seq).
	if taskID != 0 && task.Id != taskID {
		// keep whatever NewTask assigned; callers should use task.Id
	}
	greq := &GenericRequest{Task: task, Session: sess}
	mgr := getTcpRelayManager(sess.ID)
	// Minimal live registration: writer nil is enough for GetTaskProto;
	// Send() needs writer — tests that need Send seed a real writer.
	mgr.Register(nil, greq, nil)
	return sess, mgr, task
}

// ---------------------------------------------------------------------------
// P0: START is idempotent when a live stream is registered
// ---------------------------------------------------------------------------

func TestAudit_TcpRelayStartIdempotentReusesTask(t *testing.T) {
	clearTcpRelayManagers(t)
	sess, mgr, task := seedTcpRelayLiveSession(t, "tcp-relay-idem-start", 0)

	startSeq := sess.Taskseq.Load()
	got, err := (&Server{}).TcpRelay(incomingSessionContext(sess.ID), &implantpb.TunnelCtrl{
		Action: implantpb.TunnelCtrl_START,
	})
	if err != nil {
		t.Fatalf("TcpRelay START: %v", err)
	}
	if got.GetTaskId() != task.Id {
		t.Fatalf("idempotent START task id=%d want existing %d", got.GetTaskId(), task.Id)
	}
	if seq := sess.Taskseq.Load(); seq != startSeq {
		t.Errorf("AUDIT: START created new task seq %d -> %d (not idempotent)", startSeq, seq)
	} else {
		t.Logf("AUDIT OK: START reused task %d without bumping Taskseq", task.Id)
	}
	// Manager still points at same greq.
	if pb, ok := mgr.GetTaskProto(); !ok || pb.GetTaskId() != task.Id {
		t.Fatalf("manager task missing after idempotent START")
	}
}

// ---------------------------------------------------------------------------
// P0-2: LIST without live stream must NOT create a long-lived START stream
// ---------------------------------------------------------------------------

func TestAudit_TcpRelayListWithoutStreamDoesNotStart(t *testing.T) {
	clearTcpRelayManagers(t)
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "tcp-relay-list-nostream", "tcp-relay-list-pipe", true)

	startSeq := sess.Taskseq.Load()
	startUnfinished := len(sess.Tasks.GetNotFinish())

	// No manager registered for this session.
	ctx, cancel := context.WithTimeout(incomingSessionContext(sess.ID), 2*time.Second)
	defer cancel()

	// LIST with no stream: correct behavior is error or empty one-shot without
	// registering a long-lived manager. Current code may fall through to START
	// which needs StreamGenericHandler + pipeline stream — often errors.
	got, err := (&Server{}).TcpRelay(ctx, &implantpb.TunnelCtrl{
		Action: implantpb.TunnelCtrl_LIST,
	})

	endSeq := sess.Taskseq.Load()
	endUnfinished := len(sess.Tasks.GetNotFinish())
	if endSeq != startSeq || endUnfinished != startUnfinished {
		t.Fatalf("LIST without stream changed Taskseq %d->%d unfinished %d->%d err=%v task=%v",
			startSeq, endSeq, startUnfinished, endUnfinished, err, got)
	}
	if err == nil {
		t.Fatal("LIST without stream should return error")
	}
	if !errors.Is(err, types.ErrNotFoundTask) {
		t.Fatalf("LIST no-stream err=%v, want ErrNotFoundTask", err)
	}
	if _, ok := tcpRelayManagers.Load(sess.ID); ok {
		if m := getTcpRelayManager(sess.ID); m != nil {
			if _, live := m.GetTaskProto(); live {
				t.Fatal("LIST left a live tcp_relay manager/task")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// STOP on live manager removes registration
// ---------------------------------------------------------------------------

func TestAudit_TcpRelayStopRemovesManager(t *testing.T) {
	clearTcpRelayManagers(t)
	sess, mgr, task := seedTcpRelayLiveSession(t, "tcp-relay-stop-live", 0)

	// Register a writer that accepts Send so STOP path uses existing stream.
	var sentMu sync.Mutex
	var sent []*clientpb.SpiteRequest
	stream := &testRPCServerStream{
		sendMsg: func(msg interface{}) error {
			if req, ok := msg.(*clientpb.SpiteRequest); ok {
				sentMu.Lock()
				sent = append(sent, req)
				sentMu.Unlock()
			}
			return nil
		},
	}
	writer, _, err := sess.RequestWithStream(&clientpb.SpiteRequest{
		Session: &clientpb.Session{SessionId: sess.ID},
		Task:    &clientpb.Task{TaskId: task.Id, SessionId: sess.ID},
	}, stream, 0)
	if err != nil {
		t.Fatalf("RequestWithStream: %v", err)
	}
	t.Cleanup(writer.Close)
	greq := &GenericRequest{Task: task, Session: sess}
	mgr.Register(writer, greq, nil)

	got, err := (&Server{}).TcpRelay(incomingSessionContext(sess.ID), &implantpb.TunnelCtrl{
		Action: implantpb.TunnelCtrl_STOP,
	})
	if err != nil {
		t.Fatalf("STOP: %v", err)
	}
	if got.GetTaskId() != task.Id {
		t.Fatalf("STOP task id=%d want %d", got.GetTaskId(), task.Id)
	}
	if _, ok := mgr.GetTaskProto(); ok {
		t.Errorf("AUDIT: manager still has task after STOP")
	} else {
		t.Log("AUDIT OK: STOP cleared manager session")
	}
}

// ---------------------------------------------------------------------------
// TunnelOpen without stream → ErrNotFoundTask
// ---------------------------------------------------------------------------

func TestAudit_TunnelOpenWithoutStream(t *testing.T) {
	clearTcpRelayManagers(t)
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "tcp-relay-open-nostream", "open-pipe", true)

	_, err := (&Server{}).TunnelOpen(incomingSessionContext(sess.ID), &implantpb.TunnelOpen{
		ConnId: 1,
		Host:   "127.0.0.1",
		Port:   80,
	})
	if err == nil {
		t.Fatal("expected error without stream")
	}
	if !errors.Is(err, types.ErrNotFoundTask) {
		t.Logf("AUDIT: TunnelOpen no-stream err=%v (want ErrNotFoundTask)", err)
	} else {
		t.Log("AUDIT OK: TunnelOpen without stream → ErrNotFoundTask")
	}
}

// ---------------------------------------------------------------------------
// Register replaces previous writer (documents non-idempotent Register itself)
// ---------------------------------------------------------------------------

func TestAudit_TcpRelayManagerRegisterReplacesPrevious(t *testing.T) {
	clearTcpRelayManagers(t)
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "tcp-relay-replace", "replace-pipe", true)

	task1 := sess.NewTask("tcp_relay", -1)
	task2 := sess.NewTask("tcp_relay", -1)
	mgr := getTcpRelayManager(sess.ID)
	mgr.Register(nil, &GenericRequest{Task: task1, Session: sess}, nil)
	mgr.Register(nil, &GenericRequest{Task: task2, Session: sess}, nil)

	pb, ok := mgr.GetTaskProto()
	if !ok {
		t.Fatal("no task after second Register")
	}
	if pb.GetTaskId() != task2.Id {
		t.Fatalf("expected second task %d got %d", task2.Id, pb.GetTaskId())
	}
	// Documents: Register itself is last-write-wins; only TcpRelay START path is idempotent.
	t.Logf("AUDIT NOTE: Register is last-write-wins (task %d replaced %d); START RPC path avoids this when GetTaskProto ok",
		task2.Id, task1.Id)
}
