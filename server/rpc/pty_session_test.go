package rpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestPumpOutputFinishCalledOnceOnNormalExit(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "pty-finish-test", "pty-pipe", true)

	task := sess.NewTask("pty", -1)
	task.Ctx, task.Cancel = context.WithCancel(context.Background())

	var callbackCount atomic.Int32
	task.Callback = func() {
		callbackCount.Add(1)
	}

	greq := &GenericRequest{
		Task:    task,
		Session: sess,
	}

	respCh := make(chan *implantpb.Spite, 4)
	for i := 0; i < 3; i++ {
		respCh <- &implantpb.Spite{
			Name: "pty_response",
			Body: &implantpb.Spite_PtyResponse{PtyResponse: &implantpb.PtyResponse{
				SessionActive: true,
				OutputData:    []byte("output"),
			}},
		}
	}
	respCh <- &implantpb.Spite{
		Name: "pty_response",
		Body: &implantpb.Spite_PtyResponse{PtyResponse: &implantpb.PtyResponse{
			SessionActive: false,
		}},
	}
	close(respCh)

	mgr := NewImplantPTYManager()
	mgr.Register("pty-sub-1", &core.SpiteStreamWriter{}, greq)
	mgr.PumpOutput("pty-sub-1", greq, respCh)

	got := callbackCount.Load()
	if got != 1 {
		t.Fatalf("Task.Callback called %d times, want exactly 1 (Finish should only fire on session exit)", got)
	}
}

func TestHandlePtyStopReturnsExistingTaskWithoutCreatingOrphan(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "pty-stop-existing-task", "pty-stop-pipe", true)
	t.Cleanup(func() {
		implantPTYManagers.Delete(sess.ID)
	})

	task := sess.NewTask("pty", -1)
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
		t.Fatalf("RequestWithStream failed: %v", err)
	}
	t.Cleanup(writer.Close)

	greq := &GenericRequest{
		Task:    task,
		Session: sess,
	}
	mgr := getImplantPTYManager(sess.ID)
	mgr.Register("pty-sub-1", writer, greq)

	startSeq := sess.Taskseq.Load()
	startUnfinished := len(sess.Tasks.GetNotFinish())

	got, err := (&Server{}).handlePtyStop(incomingSessionContext(sess.ID), &implantpb.PtyRequest{
		Type:      consts.ModulePtyStop,
		SessionId: "pty-sub-1",
	})
	if err != nil {
		t.Fatalf("handlePtyStop returned error: %v", err)
	}
	if got.GetTaskId() != task.Id {
		t.Fatalf("handlePtyStop returned task id %d, want existing task id %d", got.GetTaskId(), task.Id)
	}
	if seq := sess.Taskseq.Load(); seq != startSeq {
		t.Fatalf("Taskseq changed from %d to %d; stop should not create a new task", startSeq, seq)
	}
	if unfinished := len(sess.Tasks.GetNotFinish()); unfinished != startUnfinished {
		t.Fatalf("unfinished task count = %d, want %d", unfinished, startUnfinished)
	}
	if _, ok := mgr.Get("pty-sub-1"); ok {
		t.Fatal("PTY manager still contains stopped session")
	}

	waitForCondition(t, time.Second, func() bool {
		sentMu.Lock()
		defer sentMu.Unlock()
		for _, req := range sent {
			if req.GetSpite().GetPtyRequest().GetType() == consts.ModulePtyStop {
				return true
			}
		}
		return false
	}, "PTY stop command to be sent to existing stream")
}

func TestHandlePtyCommandReturnsExistingTaskWithoutCreatingOrphan(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "pty-command-existing-task", "pty-command-pipe", true)
	t.Cleanup(func() {
		implantPTYManagers.Delete(sess.ID)
	})

	task := sess.NewTask("pty", -1)
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
		t.Fatalf("RequestWithStream failed: %v", err)
	}
	t.Cleanup(writer.Close)

	greq := &GenericRequest{
		Task:    task,
		Session: sess,
	}
	mgr := getImplantPTYManager(sess.ID)
	mgr.Register("pty-sub-1", writer, greq)

	startSeq := sess.Taskseq.Load()
	startUnfinished := len(sess.Tasks.GetNotFinish())
	tests := []struct {
		name string
		req  *implantpb.PtyRequest
	}{
		{
			name: "input",
			req: &implantpb.PtyRequest{
				Type:      consts.ModulePtyInput,
				SessionId: "pty-sub-1",
				InputData: []byte("whoami\n"),
			},
		},
		{
			name: "resize",
			req: &implantpb.PtyRequest{
				Type:      "resize",
				SessionId: "pty-sub-1",
				Cols:      120,
				Rows:      40,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Server{}).handlePtyCommand(incomingSessionContext(sess.ID), tt.req)
			if err != nil {
				t.Fatalf("handlePtyCommand returned error: %v", err)
			}
			if got.GetTaskId() != task.Id {
				t.Fatalf("handlePtyCommand returned task id %d, want existing task id %d", got.GetTaskId(), task.Id)
			}
		})
	}

	if seq := sess.Taskseq.Load(); seq != startSeq {
		t.Fatalf("Taskseq changed from %d to %d; PTY commands should not create a new task", startSeq, seq)
	}
	if unfinished := len(sess.Tasks.GetNotFinish()); unfinished != startUnfinished {
		t.Fatalf("unfinished task count = %d, want %d", unfinished, startUnfinished)
	}
	if _, ok := mgr.Get("pty-sub-1"); !ok {
		t.Fatal("PTY manager removed an active session after a command")
	}

	wantedTypes := map[string]bool{
		consts.ModulePtyInput: false,
		"resize":              false,
	}
	waitForCondition(t, time.Second, func() bool {
		sentMu.Lock()
		defer sentMu.Unlock()
		for _, sentReq := range sent {
			ptyReq := sentReq.GetSpite().GetPtyRequest()
			if ptyReq == nil {
				continue
			}
			if _, ok := wantedTypes[ptyReq.GetType()]; ok {
				if sentReq.GetSpite().GetTaskId() != task.Id {
					return false
				}
				wantedTypes[ptyReq.GetType()] = true
			}
		}
		return wantedTypes[consts.ModulePtyInput] && wantedTypes["resize"]
	}, "PTY input and resize commands to use the existing stream task")
}
