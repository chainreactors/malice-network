package rpc

import (
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func waitEventBrokerReady(t testing.TB, broker interface{ TryPublish(core.Event) error }) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		err := broker.TryPublish(core.Event{EventType: "test", Op: "ready"})
		if err == nil {
			return
		}
		if !errors.Is(err, core.ErrEventBrokerUnavailable) {
			t.Fatalf("unexpected broker readiness error: %v", err)
		}
		select {
		case <-deadline:
			t.Fatal("broker did not become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func subscribeEventBrokerReady(t testing.TB, broker interface {
	Subscribe() (chan core.Event, error)
	Unsubscribe(chan core.Event)
	TryPublish(core.Event) error
}) chan core.Event {
	t.Helper()
	sub, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe error = %v", err)
	}
	readyOp := "subscriber-ready"
	deadline := time.After(2 * time.Second)
	for {
		err = broker.TryPublish(core.Event{EventType: "test", Op: readyOp})
		if err != nil && !errors.Is(err, core.ErrEventBrokerQueueFull) {
			t.Fatalf("publish subscriber readiness event: %v", err)
		}
		select {
		case evt := <-sub:
			if evt.EventType == "test" && evt.Op == readyOp {
				return sub
			}
		case <-deadline:
			broker.Unsubscribe(sub)
			t.Fatal("subscriber did not become ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestGenericRequestNewSpiteUsesTaskTimeout(t *testing.T) {
	spite, err := (&GenericRequest{}).NewSpite(&implantpb.Request{Name: "test"})
	if err != nil {
		t.Fatalf("NewSpite failed: %v", err)
	}
	if got := time.Duration(spite.Timeout) * time.Second; got != configs.DefaultTaskTimeout {
		t.Fatalf("spite timeout = %v, want %v", got, configs.DefaultTaskTimeout)
	}
}

func TestGenericRequestHandlerResponsePublishesTaskError(t *testing.T) {
	env := newRPCTestEnv(t)
	broker := core.EventBroker
	sub := subscribeEventBrokerReady(t, broker)
	defer broker.Unsubscribe(sub)
	sess := env.seedSession(t, "session-a", "task-error-pipeline", true)
	task := sess.NewTask("exec", 1)
	if err := db.AddTask(task.ToProtobuf()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	req := &GenericRequest{
		Task:    task,
		Session: sess,
	}
	ch := make(chan *implantpb.Spite, 1)

	req.HandlerResponse(ch, types.MsgExec)
	ch <- nil

	deadline := time.After(2 * time.Second)
	for {
		select {
		case evt := <-sub:
			if evt.Op != consts.CtrlTaskError {
				continue
			}
			if evt.Task == nil || evt.Task.TaskId != req.Task.Id {
				t.Fatalf("unexpected task payload: %#v", evt.Task)
			}
			if evt.Task.Status != consts.CtrlStatusFailed || evt.Task.Error == "" {
				t.Fatalf("task error was not terminal: %#v", evt.Task)
			}
			if !req.Task.IsClosed() || req.Task.FinishedAtTime().IsZero() {
				t.Fatal("task handler error did not close the task")
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for task error event")
		}
	}
}

func TestGenericRequestHandlerSpitePersistsIncrementalProgress(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-handler-progress", "rpc-handler-progress-pipe", true)
	task := sess.NewTask("multi-stage-task", 3)
	t.Cleanup(task.Close)

	if err := db.AddTask(task.ToProtobuf()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	req := &GenericRequest{
		Task:    task,
		Session: sess,
	}
	spite := &implantpb.Spite{
		TaskId: task.Id,
		Name:   task.Type,
		Body:   &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
	}

	if err := req.HandlerSpite(spite); err != nil {
		t.Fatalf("HandlerSpite failed: %v", err)
	}

	stored, err := db.GetTask(task.TaskID())
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if stored.Cur != 1 {
		t.Fatalf("persisted task cur = %d, want 1 after first callback", stored.Cur)
	}
	if task.Cur != 1 {
		t.Fatalf("in-memory task cur = %d, want 1 after first callback", task.Cur)
	}
}
